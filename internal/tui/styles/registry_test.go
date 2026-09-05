package styles

import (
	"sync"
	"testing"
)

func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]Theme{}
	currentName = ""
	current = Theme{}
	defaults = sync.Once{}
}

func TestRegisterDefaults(t *testing.T) {
	resetRegistry()
	RegisterDefaults()
	for _, name := range []string{"dark", "light", "dracula"} {
		if _, ok := Get(name); !ok {
			t.Errorf("default theme %q not registered", name)
		}
	}
}

func TestSetAndCurrent(t *testing.T) {
	resetRegistry()
	RegisterDefaults()
	if !Set("dracula") {
		t.Fatal("Set(dracula) should succeed")
	}
	if got := Current().Name; got != "dracula" {
		t.Errorf("Current().Name = %q, want dracula", got)
	}
	if Name() != "dracula" {
		t.Errorf("Name() = %q, want dracula", Name())
	}
}

func TestSetUnknown(t *testing.T) {
	resetRegistry()
	RegisterDefaults()
	if Set("nope") {
		t.Error("Set(nope) should return false")
	}
}

func TestCurrentFallsBackToDark(t *testing.T) {
	resetRegistry()
	got := Current()
	if got.Name != "dark" {
		t.Errorf("Current() fallback name = %q, want dark", got.Name)
	}
}

func TestNextCycles(t *testing.T) {
	resetRegistry()
	RegisterDefaults()
	Set("dark")
	names := Names()
	for i := 0; i < len(names); i++ {
		Next()
	}
	if Name() != "dark" {
		t.Errorf("after cycling through all themes, name = %q, want dark (wraps)", Name())
	}
}

func TestOnChangeHook(t *testing.T) {
	resetRegistry()
	RegisterDefaults()
	OnChange(func(th Theme) Theme {
		th.Accent = 0xDEADBEEF
		return th
	})
	Set("dark")
	if got := Current().Accent; got != 0xDEADBEEF {
		t.Errorf("hooked Accent = %v, want 0xDEADBEEF", got)
	}
}

func TestCustomRegister(t *testing.T) {
	resetRegistry()
	RegisterDefaults()
	Register("solarized", Theme{Background: 0x111111})
	if !Set("solarized") {
		t.Fatal("Set(solarized) should succeed after Register")
	}
	if Current().Name != "solarized" {
		t.Errorf("Current().Name = %q, want solarized", Current().Name)
	}
}

func TestThemesHaveColors(t *testing.T) {
	resetRegistry()
	RegisterDefaults()
	for _, name := range Names() {
		th, ok := Get(name)
		if !ok {
			t.Fatalf("theme %q missing", name)
		}
		if th.Background == 0 || th.Foreground == 0 || th.Accent == 0 {
			t.Errorf("theme %q has unset colors: %+v", name, th)
		}
	}
}
