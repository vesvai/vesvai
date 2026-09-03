package subagent

import (
	"context"
	"errors"
	"fmt"
	json "github.com/goccy/go-json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/agent/tools"
	builtinmw "github.com/vesvai/vesvai/internal/builtin/middlewares"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/core/logger"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "subagent-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "subagent: mkdir temp:", err)
		os.Exit(1)
	}
	if err := os.Chdir(tmp); err != nil {
		fmt.Fprintln(os.Stderr, "subagent: chdir temp:", err)
		os.Exit(1)
	}
	store, err := session.NewJSONStoreAt(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "subagent: session store:", err)
		os.Exit(1)
	}
	testMgr = session.NewManager(store, testBus, logger.New(logger.LevelDebug, discardHandler{}))
	rec := session.NewRecorder(testMgr, logger.New(logger.LevelDebug, discardHandler{}))
	if err := rec.Start(testBus); err != nil {
		fmt.Fprintln(os.Stderr, "subagent: recorder:", err)
		os.Exit(1)
	}
	agents.Register(func() (*agent.Agent, error) {
		return agent.New("sub-agent",
			agent.WithToolNames("now"),
			agent.WithMiddlewareNames("loop-detector", "redaction"),
		), nil
	})
	SubAgentTools(testMgr)
	builtinmw.Create()
	tools.Register(tool.NewSpec("now", "stub", map[string]any{}, func(ctx context.Context, args string) (string, error) {
		return "now", nil
	}))
	os.Exit(m.Run())
}

var testBus = event.New()
var testMgr *session.Manager

type discardHandler struct{}

func (discardHandler) Write(logger.Record) error { return nil }
func (discardHandler) Close() error              { return nil }

type stubProvider struct{}

func (stubProvider) Name() string { return "stub" }
func (stubProvider) Chat(context.Context, *llm.Request) (*llm.Response, error) {
	return &llm.Response{Choices: []llm.Choice{{
		Index: 0,
		Message: func() *llm.Message {
			m := llm.AssistantMessage("sub answer")
			return &m
		}(),
		FinishReason: func() *llm.FinishReason { fr := llm.FinishReasonStop; return &fr }(),
	}}}, nil
}
func (stubProvider) ChatStream(context.Context, *llm.Request, llm.StreamHandler) error { return nil }
func (stubProvider) ListModels(context.Context) ([]llm.Model, error)                   { return nil, nil }

func parentCtx(t *testing.T, names ...string) context.Context {
	t.Helper()
	parent := agent.New("parent")
	parent.Provider = stubProvider{}
	parent.Model = llm.Model{ID: "stub-model"}
	return agent.WithAgent(context.Background(), parent)
}

func spec(name, agent, task string, extra ...map[string]any) map[string]any {
	s := map[string]any{"name": name, "agent": agent, "task": task}
	for _, e := range extra {
		for k, v := range e {
			s[k] = v
		}
	}
	return s
}

func batchArgs(t *testing.T, bg bool, specs ...map[string]any) string {
	t.Helper()
	if len(specs) == 0 {
		specs = []map[string]any{}
	}
	return mustJSON(t, map[string]any{"subagents": specs, "background": bg})
}

func TestSubAgentToolsRegistered(t *testing.T) {
	for _, name := range []string{"subagent", "wait-for-subagents", "subagents-status"} {
		if _, ok := tools.Get(name); !ok {
			t.Errorf("tool %q not registered", name)
		}
	}
}

func TestSubAgentTool_Schema_AgentEnum(t *testing.T) {
	tt, _ := tools.Get("subagent")
	var schema map[string]any
	if err := json.Unmarshal([]byte(mustJSON(t, tt.Parameters())), &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	props := schema["properties"].(map[string]any)
	subagentsProp := props["subagents"].(map[string]any)
	items := subagentsProp["items"].(map[string]any)
	itemProps := items["properties"].(map[string]any)
	agentProp := itemProps["agent"].(map[string]any)
	enum, ok := agentProp["enum"].([]any)
	if !ok {
		t.Fatalf("agent enum missing: %v", agentProp)
	}
	registered := agents.List()
	if len(enum) != len(registered) {
		t.Errorf("enum %v does not match registered agents %v", enum, registered)
	}
	for _, r := range registered {
		found := false
		for _, e := range enum {
			if e == r {
				found = true
			}
		}
		if !found {
			t.Errorf("registered agent %q missing from enum %v", r, enum)
		}
	}
	required := schema["required"].([]any)
	if len(required) != 1 || required[0] != "subagents" {
		t.Errorf("required = %v, want [subagents]", required)
	}
	if _, ok := props["background"]; !ok {
		t.Error("background property missing")
	}
}

func TestSubAgentTool_Validation(t *testing.T) {
	tt, _ := tools.Get("subagent")
	ctx := parentCtx(t)

	cases := []struct {
		name string
		args string
	}{
		{"empty array", `{"subagents": []}`},
		{"missing subagents", `{}`},
		{"missing name", `{"subagents": [{"agent": "sub-agent", "task": "do x"}]}`},
		{"missing task", `{"subagents": [{"name": "a-1", "agent": "sub-agent"}]}`},
		{"unknown agent", `{"subagents": [{"name": "a-1", "agent": "nope", "task": "do x"}]}`},
		{"duplicate in batch", `{"subagents": [{"name": "a-1", "agent": "sub-agent", "task": "x"}, {"name": "a-1", "agent": "sub-agent", "task": "y"}]}`},
	}
	for _, c := range cases {
		if _, err := tt.Execute(ctx, c.args); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}

func TestSubAgentTool_NoParent(t *testing.T) {
	tt, _ := tools.Get("subagent")
	if _, err := tt.Execute(context.Background(), `{"subagents": [{"name": "a-1", "agent": "sub-agent", "task": "do x"}]}`); err == nil {
		t.Error("expected error without parent agent in context")
	}
}

func TestSubAgentTool_Foreground(t *testing.T) {
	tt, _ := tools.Get("subagent")
	ctx := parentCtx(t)
	out, err := tt.Execute(ctx, batchArgs(t, false, spec("fg-1", "sub-agent", "do x", map[string]any{"task_id": []string{"todo-1"}})))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "sub answer") {
		t.Errorf("output = %q, want sub answer", out)
	}
	sa, ok := store.get("fg-1")
	if !ok {
		t.Fatal("subagent not recorded")
	}
	if sa.Status != StatusCompleted {
		t.Errorf("status = %q, want completed", sa.Status)
	}
	if len(sa.TaskIDs) != 1 || sa.TaskIDs[0] != "todo-1" {
		t.Errorf("task ids = %v, want [todo-1]", sa.TaskIDs)
	}
}

func TestSubAgentTool_DuplicateName(t *testing.T) {
	tt, _ := tools.Get("subagent")
	ctx := parentCtx(t)
	if _, err := tt.Execute(ctx, batchArgs(t, false, spec("dup-1", "sub-agent", "first"))); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if _, err := tt.Execute(ctx, batchArgs(t, false, spec("dup-1", "sub-agent", "second"))); err == nil {
		t.Error("expected duplicate name error")
	}
}

func TestSubAgentTool_BatchForeground(t *testing.T) {
	tt, _ := tools.Get("subagent")
	ctx := parentCtx(t)

	args := batchArgs(t, false,
		spec("batch-fg-1", "sub-agent", "t1"),
		spec("batch-fg-2", "sub-agent", "t2"),
		spec("batch-fg-3", "sub-agent", "t3"),
		spec("batch-fg-4", "sub-agent", "t4"),
		spec("batch-fg-5", "sub-agent", "t5"),
		spec("batch-fg-6", "sub-agent", "t6"),
		spec("batch-fg-7", "sub-agent", "t7"),
		spec("batch-fg-8", "sub-agent", "t8"),
	)
	out, err := tt.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Count(out, "Subagent \"batch-fg-") != 8 {
		t.Errorf("output = %q, want 8 finished blocks", out)
	}
	for i := 1; i <= 8; i++ {
		name := fmt.Sprintf("batch-fg-%d", i)
		sa, ok := store.get(name)
		if !ok {
			t.Fatalf("%s not recorded", name)
		}
		if sa.Status != StatusCompleted {
			t.Errorf("%s status = %q, want completed", name, sa.Status)
		}
		if !strings.Contains(out, name) {
			t.Errorf("output missing %s", name)
		}
	}
}

func TestSubAgentTool_BatchBackgroundAndWait(t *testing.T) {
	tt, _ := tools.Get("subagent")
	wait, _ := tools.Get("wait-for-subagents")
	ctx := parentCtx(t)

	specs := make([]map[string]any, 0, 8)
	names := make([]string, 0, 8)
	for i := 1; i <= 8; i++ {
		name := fmt.Sprintf("batch-bg-%d", i)
		specs = append(specs, spec(name, "sub-agent", "do x"))
		names = append(names, name)
	}
	out, err := tt.Execute(ctx, batchArgs(t, true, specs...))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "8 subagents") {
		t.Errorf("output = %q, want 8 subagents started", out)
	}

	waitOut, err := wait.Execute(ctx, `{"agent_names": `+mustJSON(t, names)+`}`)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	for _, n := range names {
		if !strings.Contains(waitOut, n) {
			t.Errorf("wait output missing %s", n)
		}
	}
	if strings.Count(waitOut, "completed") != 8 {
		t.Errorf("completed count = %d, want 8", strings.Count(waitOut, "completed"))
	}
	for _, n := range names {
		sa, _ := store.get(n)
		if sa.Status != StatusCompleted {
			t.Errorf("%s status = %q, want completed", n, sa.Status)
		}
	}
}

func TestSubAgentTool_BackgroundAndWait(t *testing.T) {
	tt, _ := tools.Get("subagent")
	wait, _ := tools.Get("wait-for-subagents")
	ctx := parentCtx(t)

	out, err := tt.Execute(ctx, batchArgs(t, true, spec("bg-1", "sub-agent", "do x")))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "background") {
		t.Errorf("output = %q, want started confirmation", out)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		sa, _ := store.get("bg-1")
		if sa.Status == StatusCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background subagent not completed, status = %q", sa.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	out, err = wait.Execute(ctx, `{"agent_names": ["bg-1"]}`)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if !strings.Contains(out, "sub answer") {
		t.Errorf("wait output = %q, want sub answer", out)
	}
}

func TestIsSubagentSession(t *testing.T) {
	sa, err := store.spawn("sess-test-global-1", "sub-agent", nil, false)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	store.setAgentID(sa.Name, "sess-agent-1")
	store.onSessionAttached("sess-agent-1", "subagent-sess-123")

	if !IsSubagentSession("subagent-sess-123") {
		t.Fatal("recorded subagent session must be recognized")
	}
	if IsSubagentSession("main-sess-999") {
		t.Fatal("main session must not be reported as subagent session")
	}
	if IsSubagentSession("") {
		t.Fatal("empty session must not be reported as subagent session")
	}
}

func TestWaitForSubAgents_UnknownName(t *testing.T) {
	wait, _ := tools.Get("wait-for-subagents")
	if _, err := wait.Execute(parentCtx(t), `{"agent_names": ["ghost"]}`); err == nil {
		t.Error("expected error for unknown subagent name")
	}
}

func TestSubAgentsStatus(t *testing.T) {
	tt, _ := tools.Get("subagent")
	status, _ := tools.Get("subagents-status")
	ctx := parentCtx(t)

	if _, err := tt.Execute(ctx, batchArgs(t, false, spec("stat-1", "sub-agent", "do x"))); err != nil {
		t.Fatalf("spawn: %v", err)
	}

	out, err := status.Execute(ctx, `{}`)
	if err != nil {
		t.Fatalf("status all: %v", err)
	}
	if !strings.Contains(out, "stat-1") {
		t.Errorf("status output missing stat-1: %q", out)
	}

	if _, err := status.Execute(ctx, `{"agent_names": ["ghost"]}`); err == nil {
		t.Error("expected error for unknown name filter")
	}
	out, err = status.Execute(ctx, `{"agent_names": ["stat-1"]}`)
	if err != nil {
		t.Fatalf("status filter: %v", err)
	}
	if !strings.Contains(out, "stat-1") {
		t.Errorf("status output missing stat-1: %q", out)
	}
}

func TestSubAgentTool_ConcurrentParents(t *testing.T) {
	tt, _ := tools.Get("subagent")
	wait, _ := tools.Get("wait-for-subagents")
	status, _ := tools.Get("subagents-status")

	const parents = 8
	var wg sync.WaitGroup
	for i := 0; i < parents; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx := parentCtx(t)
			name := fmt.Sprintf("conc-%d", i)
			_, err := tt.Execute(ctx, batchArgs(t, true, spec(name, "sub-agent", "do x")))
			if err != nil {
				t.Errorf("parent %d spawn: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	names := make([]string, parents)
	for i := range names {
		names[i] = fmt.Sprintf("conc-%d", i)
	}
	out, err := wait.Execute(parentCtx(t), `{"agent_names": `+mustJSON(t, names)+`}`)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	for i := 0; i < parents; i++ {
		if !strings.Contains(out, fmt.Sprintf("conc-%d", i)) {
			t.Errorf("wait output missing conc-%d", i)
		}
	}
	if strings.Count(out, "completed") != parents {
		t.Errorf("completed count = %d, want %d", strings.Count(out, "completed"), parents)
	}

	statusOut, err := status.Execute(parentCtx(t), `{}`)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for i := 0; i < parents; i++ {
		if !strings.Contains(statusOut, fmt.Sprintf("conc-%d", i)) {
			t.Errorf("status output missing conc-%d", i)
		}
	}
}

func TestRegistry_PersistAndLoad(t *testing.T) {
	r := newRegistry()
	sa, err := r.spawn("persist-1", "sub-agent", []string{"todo-9"}, false)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	r.start(sa)
	r.finish(sa, "persisted output", nil)

	if _, err := os.Stat(r.file); err != nil {
		t.Fatalf("subagents.json not written: %v", err)
	}

	loaded := newRegistry()
	if err := loaded.init(); err != nil {
		t.Fatalf("load: %v", err)
	}
	got, ok := loaded.get("persist-1")
	if !ok {
		t.Fatal("persisted subagent not loaded")
	}
	if got.Status != StatusCompleted || got.Output != "persisted output" {
		t.Errorf("loaded = %+v, want completed with output", got)
	}
	if len(got.TaskIDs) != 1 || got.TaskIDs[0] != "todo-9" {
		t.Errorf("task ids = %v, want [todo-9]", got.TaskIDs)
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := loaded.waitFor(waitCtx, []string{"persist-1"}); err != nil {
		t.Errorf("waitFor on loaded finished subagent: %v", err)
	}
}

func TestRegistry_LoadInterruptsRunning(t *testing.T) {
	r := newRegistry()
	sa, err := r.spawn("interrupt-1", "sub-agent", nil, true)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	r.start(sa)

	loaded := newRegistry()
	if err := loaded.init(); err != nil {
		t.Fatalf("load: %v", err)
	}
	got, ok := loaded.get("interrupt-1")
	if !ok {
		t.Fatal("running subagent not loaded")
	}
	if got.Status != StatusInterrupted {
		t.Errorf("status = %q, want interrupted", got.Status)
	}
	if got.Err == "" {
		t.Error("interrupted subagent should have an error message")
	}
}

func TestAgentOutput(t *testing.T) {
	if got := agentOutput(&agent.RunResult{Output: "final answer"}, nil); got != "final answer" {
		t.Errorf("expected final output, got %q", got)
	}
	res := &agent.RunResult{
		Output: "",
		History: []llm.Message{
			llm.SystemMessage("sys"),
			llm.UserMessage("do it"),
			llm.AssistantMessage("step one"),
			llm.AssistantMessage("step two"),
		},
	}
	if got := agentOutput(res, nil); got != "step one\nstep two" {
		t.Errorf("history fallback = %q, want 'step one\\nstep two'", got)
	}
	if got := agentOutput(nil, errSentinel); got != "boom" {
		t.Errorf("error fallback = %q, want boom", got)
	}
	if got := agentOutput(nil, nil); got != "" {
		t.Errorf("empty result should yield empty output, got %q", got)
	}
}

var errSentinel = errors.New("boom")

func TestSubAgentTool_EventDrivenForeground(t *testing.T) {
	tt, _ := tools.Get("subagent")

	parent := agent.New("event-parent", agent.WithBus(testBus))
	parent.Provider = stubProvider{}
	parent.Model = llm.Model{ID: "stub-model"}
	ctx := agent.WithAgent(context.Background(), parent)

	out, err := tt.Execute(ctx, batchArgs(t, false, spec("evt-fg", "sub-agent", "do x")))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "sub answer") {
		t.Errorf("output = %q, want sub answer", out)
	}
	sa, ok := store.get("evt-fg")
	if !ok {
		t.Fatal("subagent not recorded")
	}
	if sa.Status != StatusCompleted {
		t.Errorf("status = %q, want completed (event-driven)", sa.Status)
	}
	if sa.Output != "sub answer" {
		t.Errorf("output = %q, want sub answer (from agent.finished event)", sa.Output)
	}
}

func TestSubAgentTool_EventDrivenBackground(t *testing.T) {
	tt, _ := tools.Get("subagent")
	wait, _ := tools.Get("wait-for-subagents")

	parent := agent.New("event-parent-2", agent.WithBus(testBus))
	parent.Provider = stubProvider{}
	parent.Model = llm.Model{ID: "stub-model"}
	ctx := agent.WithAgent(context.Background(), parent)

	if _, err := tt.Execute(ctx, batchArgs(t, true, spec("evt-bg", "sub-agent", "do x"))); err != nil {
		t.Fatalf("execute: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		sa, _ := store.get("evt-bg")
		if sa.Status == StatusCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background subagent not completed, status = %q", sa.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	out, err := wait.Execute(ctx, `{"agent_names": ["evt-bg"]}`)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if !strings.Contains(out, "sub answer") {
		t.Errorf("wait output = %q, want sub answer", out)
	}
}

func TestSubAgentMessage_ResumeForeground(t *testing.T) {
	tt, _ := tools.Get("subagent")
	msg, _ := tools.Get("subagent-message")

	parent := agent.New("resume-parent", agent.WithBus(testBus))
	parent.Provider = stubProvider{}
	parent.Model = llm.Model{ID: "stub-model"}
	ctx := agent.WithAgent(context.Background(), parent)

	out, err := tt.Execute(ctx, batchArgs(t, false, spec("resume-1", "sub-agent", "do x")))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if !strings.Contains(out, "sub answer") {
		t.Errorf("spawn output = %q, want sub answer", out)
	}
	sa, ok := store.get("resume-1")
	if !ok {
		t.Fatal("subagent not recorded")
	}
	if sa.SessionID == "" {
		t.Fatal("session id not recorded for subagent")
	}

	out, err = msg.Execute(ctx, `{"name": "resume-1", "message": "fix it"}`)
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	if !strings.Contains(out, "sub answer") {
		t.Errorf("resume output = %q, want sub answer", out)
	}

	done, _ := store.get("resume-1")
	if done.Status != StatusCompleted {
		t.Errorf("status = %q, want completed", done.Status)
	}
	if done.SessionID != sa.SessionID {
		t.Errorf("session changed after resume: %q -> %q", sa.SessionID, done.SessionID)
	}

	sessMsgs, err := testMgr.Messages(sa.SessionID)
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	if len(sessMsgs) != 4 {
		t.Fatalf("session messages = %d, want 4 (user, assistant, user, assistant)", len(sessMsgs))
	}
	if sessMsgs[0].Role != llm.RoleUser || sessMsgs[1].Role != llm.RoleAssistant ||
		sessMsgs[2].Role != llm.RoleUser || sessMsgs[3].Role != llm.RoleAssistant {
		t.Errorf("session roles = %v", []string{
			string(sessMsgs[0].Role), string(sessMsgs[1].Role),
			string(sessMsgs[2].Role), string(sessMsgs[3].Role),
		})
	}
}

func TestSubAgentMessage_ResumeBackground(t *testing.T) {
	tt, _ := tools.Get("subagent")
	msg, _ := tools.Get("subagent-message")
	wait, _ := tools.Get("wait-for-subagents")

	parent := agent.New("resume-bg-parent", agent.WithBus(testBus))
	parent.Provider = stubProvider{}
	parent.Model = llm.Model{ID: "stub-model"}
	ctx := agent.WithAgent(context.Background(), parent)

	if _, err := tt.Execute(ctx, batchArgs(t, false, spec("resume-bg-1", "sub-agent", "do x"))); err != nil {
		t.Fatalf("spawn: %v", err)
	}

	out, err := msg.Execute(ctx, `{"name": "resume-bg-1", "message": "fix it", "background": true}`)
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	if !strings.Contains(out, "background") {
		t.Errorf("output = %q, want background confirmation", out)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		sa, _ := store.get("resume-bg-1")
		if sa.Status == StatusCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("resumed subagent not completed, status = %q", sa.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	out, err = wait.Execute(ctx, `{"agent_names": ["resume-bg-1"]}`)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if !strings.Contains(out, "sub answer") {
		t.Errorf("wait output = %q, want sub answer", out)
	}
}

func TestSubAgentMessage_Errors(t *testing.T) {
	msg, _ := tools.Get("subagent-message")
	ctx := parentCtx(t)

	if _, err := msg.Execute(ctx, `{"name": "ghost", "message": "hi"}`); err == nil {
		t.Error("expected error for unknown name")
	}

	sa, err := store.spawn("msg-pending", "sub-agent", nil, false)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if _, err := msg.Execute(ctx, `{"name": "msg-pending", "message": "hi"}`); err == nil {
		t.Error("expected error for still-running subagent")
	}
	store.finish(sa, "done", nil)

	if _, err := msg.Execute(ctx, `{"name": "msg-pending", "message": "hi"}`); err == nil {
		t.Error("expected error for subagent without session")
	}

	if _, err := msg.Execute(ctx, `{}`); err == nil {
		t.Error("expected error for missing args")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
