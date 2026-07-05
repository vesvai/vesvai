package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/hook"
)

type mockPlugin struct {
	Base
	initCalled  bool
	startCalled bool
	stopCalled  bool
	initErr     error
	startErr    error
	stopErr     error
}

func newMockPlugin(name string) *mockPlugin {
	p := &mockPlugin{}
	p.Base = *NewBase(PluginMeta{
		Name:    name,
		Version: "1.0.0",
	})
	return p
}

func (m *mockPlugin) Init(ctx PluginContext) error {
	m.initCalled = true
	if m.initErr != nil {
		return m.initErr
	}
	return m.Base.Init(ctx)
}

func (m *mockPlugin) Start() error {
	m.startCalled = true
	if m.startErr != nil {
		return m.startErr
	}
	return m.Base.Start()
}

func (m *mockPlugin) Stop() error {
	m.stopCalled = true
	if m.stopErr != nil {
		return m.stopErr
	}
	return m.Base.Stop()
}

func TestBase_Meta(t *testing.T) {
	b := NewBase(PluginMeta{Name: "test", Version: "1.0"})
	meta := b.Meta()
	if meta.Name != "test" {
		t.Errorf("Name = %q, want %q", meta.Name, "test")
	}
	if meta.Version != "1.0" {
		t.Errorf("Version = %q, want %q", meta.Version, "1.0")
	}
}

func TestBase_State(t *testing.T) {
	b := NewBase(PluginMeta{Name: "test"})
	if b.State() != StateRegistered {
		t.Errorf("State = %v, want StateRegistered", b.State())
	}
}

func TestBase_Init(t *testing.T) {
	b := NewBase(PluginMeta{Name: "test"})
	ctx := PluginContext{}

	if err := b.Init(ctx); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if b.State() != StateInitialized {
		t.Errorf("State = %v, want StateInitialized", b.State())
	}
}

func TestBase_Init_WithCustomFn(t *testing.T) {
	b := NewBase(PluginMeta{Name: "test"})
	called := false
	b.SetInitFn(func(ctx PluginContext) error {
		called = true
		return nil
	})

	if err := b.Init(PluginContext{}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !called {
		t.Error("custom init function not called")
	}
}

func TestBase_Init_Error(t *testing.T) {
	b := NewBase(PluginMeta{Name: "test"})
	b.SetInitFn(func(ctx PluginContext) error {
		return errors.New("init failed")
	})

	if err := b.Init(PluginContext{}); err == nil {
		t.Fatal("Init() should return error")
	}
	if b.State() != StateError {
		t.Errorf("State = %v, want StateError", b.State())
	}
}

func TestBase_Init_WrongState(t *testing.T) {
	b := NewBase(PluginMeta{Name: "test"})
	b.Init(PluginContext{})

	if err := b.Init(PluginContext{}); err == nil {
		t.Fatal("Init() should return error when already initialized")
	}
}

func TestBase_Start(t *testing.T) {
	b := NewBase(PluginMeta{Name: "test"})
	b.Init(PluginContext{})

	if err := b.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if b.State() != StateRunning {
		t.Errorf("State = %v, want StateRunning", b.State())
	}
}

func TestBase_Stop(t *testing.T) {
	b := NewBase(PluginMeta{Name: "test"})
	b.Init(PluginContext{})
	b.Start()

	if err := b.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if b.State() != StateStopped {
		t.Errorf("State = %v, want StateStopped", b.State())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	factory := func() Plugin {
		return newMockPlugin("test")
	}

	if err := r.Register(factory); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if r.Count() != 1 {
		t.Errorf("Count() = %d, want 1", r.Count())
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := NewRegistry()
	factory := func() Plugin {
		return newMockPlugin("test")
	}

	r.Register(factory)
	if err := r.Register(factory); err == nil {
		t.Fatal("Register() should return error for duplicate")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	factory := func() Plugin {
		return newMockPlugin("test")
	}
	r.Register(factory)

	p, ok := r.Get("test")
	if !ok {
		t.Fatal("Get() returned false")
	}
	if p.Meta().Name != "test" {
		t.Errorf("Name = %q, want %q", p.Meta().Name, "test")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(func() Plugin { return newMockPlugin("a") })
	r.Register(func() Plugin { return newMockPlugin("b") })

	list := r.List()
	if len(list) != 2 {
		t.Errorf("List() len = %d, want 2", len(list))
	}
}

func TestManager_Load(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	h := hook.New(bus)
	reg := agent.NewToolRegistry()

	m := NewManager(
		WithHooks(h),
		WithEventBus(bus),
		WithToolRegistry(reg),
	)

	p := newMockPlugin("test")
	m.registry.MustRegister(func() Plugin { return p })

	if err := m.Load("test"); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !m.IsLoaded("test") {
		t.Error("IsLoaded() = false, want true")
	}
	if !p.initCalled {
		t.Error("Init() not called")
	}
	if !p.startCalled {
		t.Error("Start() not called")
	}
}

func TestManager_Load_Dependency(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(WithEventBus(bus))

	dep := newMockPlugin("dep")
	m.registry.MustRegister(func() Plugin { return dep })

	main := newMockPlugin("main")
	main.meta.Dependencies = []string{"dep"}
	m.registry.MustRegister(func() Plugin { return main })

	if err := m.Load("main"); err == nil {
		t.Fatal("Load() should fail without dependency")
	}

	if err := m.Load("dep"); err != nil {
		t.Fatalf("Load(dep) error = %v", err)
	}
	if err := m.Load("main"); err != nil {
		t.Fatalf("Load(main) error = %v", err)
	}
}

func TestManager_Unload(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(WithEventBus(bus))

	p := newMockPlugin("test")
	m.registry.MustRegister(func() Plugin { return p })

	m.Load("test")
	if err := m.Unload("test"); err != nil {
		t.Fatalf("Unload() error = %v", err)
	}
	if m.IsLoaded("test") {
		t.Error("IsLoaded() = true after Unload()")
	}
	if !p.stopCalled {
		t.Error("Stop() not called")
	}
}

func TestManager_Unload_DependencyCheck(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(WithEventBus(bus))

	dep := newMockPlugin("dep")
	m.registry.MustRegister(func() Plugin { return dep })

	main := newMockPlugin("main")
	main.meta.Dependencies = []string{"dep"}
	m.registry.MustRegister(func() Plugin { return main })

	m.Load("dep")
	m.Load("main")

	if err := m.Unload("dep"); err == nil {
		t.Fatal("Unload() should fail when dependents exist")
	}
}

func TestManager_LoadAll(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(WithEventBus(bus))

	p1 := newMockPlugin("a")
	p2 := newMockPlugin("b")
	m.registry.MustRegister(func() Plugin { return p1 })
	m.registry.MustRegister(func() Plugin { return p2 })

	if err := m.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if m.Count() != 2 {
		t.Errorf("Count() = %d, want 2", m.Count())
	}
}

func TestManager_Tools(t *testing.T) {
	m := NewManager()

	tool := agent.NewFuncTool("test_tool", "test", nil, func(ctx context.Context, params map[string]any) (string, error) {
		return "ok", nil
	})

	p := newPluginWithTools("test", []agent.Tool{tool})
	m.registry.MustRegister(func() Plugin { return p })

	m.Load("test")

	tools := m.Tools()
	if len(tools) != 1 {
		t.Errorf("Tools() len = %d, want 1", len(tools))
	}
}

func TestManager_Prompts(t *testing.T) {
	m := NewManager()

	p := newPluginWithPrompts("test", map[string]string{
		"system": "You are a test agent",
	})
	m.registry.MustRegister(func() Plugin { return p })

	m.Load("test")

	prompts := m.Prompts()
	if prompts["system"] != "You are a test agent" {
		t.Errorf("Prompts[system] = %q", prompts["system"])
	}
}

type pluginWithTools struct {
	Base
	tools []agent.Tool
}

func newPluginWithTools(name string, tools []agent.Tool) *pluginWithTools {
	p := &pluginWithTools{tools: tools}
	p.Base = *NewBase(PluginMeta{Name: name, Version: "1.0.0"})
	return p
}

func (p *pluginWithTools) Tools() []agent.Tool {
	return p.tools
}

type pluginWithPrompts struct {
	Base
	prompts map[string]string
}

func newPluginWithPrompts(name string, prompts map[string]string) *pluginWithPrompts {
	p := &pluginWithPrompts{prompts: prompts}
	p.Base = *NewBase(PluginMeta{Name: name, Version: "1.0.0"})
	return p
}

func (p *pluginWithPrompts) Prompts() map[string]string {
	return p.prompts
}

func TestPluginState_String(t *testing.T) {
	tests := []struct {
		state PluginState
		want  string
	}{
		{StateRegistered, "registered"},
		{StateInitialized, "initialized"},
		{StateRunning, "running"},
		{StateStopped, "stopped"},
		{StateError, "error"},
		{PluginState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestDiskPluginMeta_ToPluginMeta(t *testing.T) {
	diskMeta := DiskPluginMeta{
		Name:         "coding",
		Version:      "1.0.0",
		Description:  "Coding agent plugin",
		Author:       "vesvai",
		Entry:        "coding.so",
		Dependencies: []string{"core"},
	}

	meta := diskMeta.ToPluginMeta()

	if meta.Name != "coding" {
		t.Errorf("Name = %q, want %q", meta.Name, "coding")
	}
	if meta.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", meta.Version, "1.0.0")
	}
	if meta.Description != "Coding agent plugin" {
		t.Errorf("Description = %q, want %q", meta.Description, "Coding agent plugin")
	}
	if meta.Author != "vesvai" {
		t.Errorf("Author = %q, want %q", meta.Author, "vesvai")
	}
	if len(meta.Dependencies) != 1 || meta.Dependencies[0] != "core" {
		t.Errorf("Dependencies = %v, want [core]", meta.Dependencies)
	}
}

func TestDiskLoader_Discover(t *testing.T) {
	dir := t.TempDir()

	pluginDir := filepath.Join(dir, "coding")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}

	meta := DiskPluginMeta{
		Name:        "coding",
		Version:     "1.0.0",
		Description: "Coding plugin",
		Entry:       "coding.so",
	}

	data, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(pluginDir, "meta.json"), data, 0644)

	loader := NewDiskLoader(dir)
	metas, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(metas) != 1 {
		t.Fatalf("Discover() returned %d metas, want 1", len(metas))
	}

	if metas[0].Name != "coding" {
		t.Errorf("Name = %q, want %q", metas[0].Name, "coding")
	}
}

func TestDiskLoader_Discover_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	loader := NewDiskLoader(dir)

	metas, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(metas) != 0 {
		t.Errorf("Discover() returned %d metas, want 0", len(metas))
	}
}

func TestDiskLoader_Discover_NonExistentDir(t *testing.T) {
	loader := NewDiskLoader("/nonexistent/path")

	metas, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(metas) != 0 {
		t.Errorf("Discover() returned %d metas, want 0", len(metas))
	}
}
