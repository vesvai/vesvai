package session

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/core/logger"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/utils/query"
)

type stubProvider struct{ name string }

func (p stubProvider) Name() string { return p.name }
func (stubProvider) Chat(context.Context, *llm.Request) (*llm.Response, error) {
	return &llm.Response{}, nil
}
func (stubProvider) ChatStream(context.Context, *llm.Request, llm.StreamHandler) error {
	return nil
}
func (stubProvider) ListModels(context.Context) ([]llm.Model, error) { return nil, nil }

func newTestRecorder(t *testing.T) (*Manager, *Recorder, event.Bus) {
	t.Helper()
	store, err := newJSONStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	bus := event.New()
	mgr := NewManager(store, bus, logger.New(logger.LevelDebug, discardHandler{}))
	rec := NewRecorder(mgr, logger.New(logger.LevelDebug, discardHandler{}))
	if err := rec.Start(bus); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rec.Stop(bus) })
	return mgr, rec, bus
}

func publishStarted(bus event.Bus, id, name, provider, model string) {
	bus.Publish(agent.TopicAgentStarted, agent.AgentStarted{
		AgentID:   id,
		AgentName: name,
		Model:     llm.Model{ID: model},
		Provider:  stubProvider{name: provider},
	})
}

func TestRecorderCreatesSessionPerAgent(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)

	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "hello"})
	bus.Publish(agent.TopicAgentMessage, agent.AgentMessage{
		AgentID: "a1", AgentName: "agent-one",
		Message: llm.AssistantMessage("hi there"),
	})
	bus.Publish(agent.TopicAgentToolResult, agent.AgentToolResult{
		AgentID: "a1", AgentName: "agent-one",
		CallID: "c1", ToolName: "echo", Output: "echo:{}",
	})
	bus.Publish(agent.TopicAgentFinished, agent.AgentFinished{
		AgentID: "a1", AgentName: "agent-one",
		Usage: llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	})

	publishStarted(bus, "a2", "agent-two", "openai", "gpt-4o")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a2", Input: "second"})
	bus.Publish(agent.TopicAgentFinished, agent.AgentFinished{
		AgentID: "a2", AgentName: "agent-two",
		Usage: llm.Usage{TotalTokens: 3},
	})

	sessions, _, err := mgr.List(query.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("sessions = %+v, want 2", sessions)
	}

	var s1, s2 *Session
	for i := range sessions {
		if sessions[i].Provider == "groq" && sessions[i].Model == "llama-3.3" {
			s1 = &sessions[i]
		}
		if sessions[i].Provider == "openai" && sessions[i].Model == "gpt-4o" {
			s2 = &sessions[i]
		}
	}
	if s1 == nil || s2 == nil {
		t.Fatalf("sessions = %+v", sessions)
	}
	if s1.Provider != "groq" || s1.Model != "llama-3.3" || s1.ProjectDir == "" {
		t.Fatalf("s1 = %+v", s1)
	}
	if s2.Provider != "openai" || s2.Model != "gpt-4o" {
		t.Fatalf("s2 = %+v", s2)
	}
	if s1.Usage.TotalTokens != 15 {
		t.Fatalf("s1 usage = %+v", s1.Usage)
	}

	msgs, err := mgr.Messages(s1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("s1 messages = %+v", msgs)
	}
	if msgs[0].Role != llm.RoleUser || msgs[0].Content != "hello" {
		t.Fatalf("msgs[0] = %+v", msgs[0])
	}
	if msgs[1].Role != llm.RoleAssistant || msgs[1].Content != "hi there" {
		t.Fatalf("msgs[1] = %+v", msgs[1])
	}
	if msgs[2].Role != llm.RoleTool || msgs[2].ToolCallID != "c1" || msgs[2].Content != "echo:{}" {
		t.Fatalf("msgs[2] = %+v", msgs[2])
	}

	msgs2, err := mgr.Messages(s2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs2) != 1 || msgs2[0].Content != "second" {
		t.Fatalf("s2 messages = %+v", msgs2)
	}
}

func TestRecorderReusesSessionAcrossRuns(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)

	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "first"})
	bus.Publish(agent.TopicAgentFinished, agent.AgentFinished{
		AgentID: "a1", AgentName: "agent-one",
		Usage: llm.Usage{TotalTokens: 5},
	})

	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "second"})
	bus.Publish(agent.TopicAgentFinished, agent.AgentFinished{
		AgentID: "a1", AgentName: "agent-one",
		Usage: llm.Usage{TotalTokens: 7},
	})

	sessions, _, err := mgr.List(query.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %+v, want 1", sessions)
	}
	if sessions[0].Usage.TotalTokens != 7 {
		t.Fatalf("usage = %+v", sessions[0].Usage)
	}
	msgs, _ := mgr.Messages(sessions[0].ID)
	if len(msgs) != 2 {
		t.Fatalf("messages = %+v", msgs)
	}
}

func TestRecorderSkipsEventsWithoutStarted(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)

	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "ghost", Input: "hello"})
	bus.Publish(agent.TopicAgentMessage, agent.AgentMessage{
		AgentID: "ghost", Message: llm.AssistantMessage("hi"),
	})

	sessions, total, err := mgr.List(query.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(sessions) != 0 {
		t.Fatalf("sessions = %+v, total = %d", sessions, total)
	}
}

func TestRecorderCommitsTokenBatches(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)
	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "go"})

	for i := 0; i < 25; i++ {
		bus.Publish(agent.TopicAgentToken, agent.AgentToken{AgentID: "a1", Content: "x"})
	}
	bus.Publish(agent.TopicAgentError, agent.AgentError{AgentID: "a1", Err: context.Canceled})

	sessions, _, _ := mgr.List(query.Query{})
	msgs, err := mgr.Messages(sessions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 4 {
		t.Fatalf("messages = %+v, want 4 (user + 2 batches + residual)", msgs)
	}
	if msgs[1].Role != llm.RoleAssistant || msgs[1].Content != "xxxxxxxxxx" {
		t.Fatalf("msgs[1] = %+v", msgs[1])
	}
	if msgs[2].Role != llm.RoleAssistant || msgs[2].Content != "xxxxxxxxxx" {
		t.Fatalf("msgs[2] = %+v", msgs[2])
	}
	if msgs[3].Role != llm.RoleAssistant || msgs[3].Content != "xxxxx" {
		t.Fatalf("msgs[3] = %+v", msgs[3])
	}
}

func TestRecorderSavesPartialOutputOnError(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)
	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "go"})

	for i := 0; i < 3; i++ {
		bus.Publish(agent.TopicAgentToken, agent.AgentToken{AgentID: "a1", Content: "ab"})
	}
	bus.Publish(agent.TopicAgentError, agent.AgentError{
		AgentID: "a1", AgentName: "agent-one", Err: context.Canceled,
	})

	sessions, _, _ := mgr.List(query.Query{})
	msgs, err := mgr.Messages(sessions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("messages = %+v, want 2 (user + partial)", msgs)
	}
	if msgs[1].Role != llm.RoleAssistant || msgs[1].Content != "ababab" {
		t.Fatalf("partial message = %+v", msgs[1])
	}
}

func TestRecorderStreamingMessageNotDuplicated(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)
	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "go"})

	full := "abcdefghijklmno"
	for i := 0; i < len(full); i++ {
		bus.Publish(agent.TopicAgentToken, agent.AgentToken{AgentID: "a1", Content: string(full[i])})
	}
	bus.Publish(agent.TopicAgentMessage, agent.AgentMessage{
		AgentID: "a1", AgentName: "agent-one",
		Message: llm.AssistantMessage(full),
	})

	sessions, _, _ := mgr.List(query.Query{})
	msgs, err := mgr.Messages(sessions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	var assistant []Message
	for _, m := range msgs {
		if m.Role == llm.RoleAssistant {
			assistant = append(assistant, m)
		}
	}
	if len(assistant) != 2 {
		t.Fatalf("assistant messages = %+v, want 2 (batch + residual)", assistant)
	}
	if assistant[0].Content != "abcdefghij" || assistant[1].Content != "klmno" {
		t.Fatalf("assistant messages = %+v", assistant)
	}
}

func TestRecorderStreamingReasoningSaved(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)
	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "go"})

	bus.Publish(agent.TopicAgentToken, agent.AgentToken{AgentID: "a1", Reasoning: "thinking"})
	for i := 0; i < chunksPerCommit-1; i++ {
		bus.Publish(agent.TopicAgentToken, agent.AgentToken{AgentID: "a1", Content: "y"})
	}

	sessions, _, _ := mgr.List(query.Query{})
	msgs, err := mgr.Messages(sessions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	last := msgs[len(msgs)-1]
	if last.Role != llm.RoleAssistant || last.Reasoning != "thinking" || last.Content != "yyyyyyyyy" {
		t.Fatalf("last message = %+v", last)
	}
}

func TestRecorderResumeContinuesSession(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)

	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "first"})
	bus.Publish(agent.TopicAgentMessage, agent.AgentMessage{
		AgentID: "a1", AgentName: "agent-one",
		Message: llm.AssistantMessage("first answer"),
	})
	bus.Publish(agent.TopicAgentFinished, agent.AgentFinished{
		AgentID: "a1", AgentName: "agent-one",
		Usage: llm.Usage{TotalTokens: 5},
	})

	sessions, _, _ := mgr.List(query.Query{})
	if len(sessions) != 1 {
		t.Fatalf("sessions = %+v, want 1", sessions)
	}
	origID := sessions[0].ID

	bus.Publish(TopicSessionResume, SessionResume{AgentID: "a2", SessionID: origID})
	publishStarted(bus, "a2", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a2", Input: "follow-up"})
	bus.Publish(agent.TopicAgentMessage, agent.AgentMessage{
		AgentID: "a2", AgentName: "agent-one",
		Message: llm.AssistantMessage("second answer"),
	})
	bus.Publish(agent.TopicAgentFinished, agent.AgentFinished{
		AgentID: "a2", AgentName: "agent-one",
		Usage: llm.Usage{TotalTokens: 7},
	})

	sessions, _, _ = mgr.List(query.Query{})
	if len(sessions) != 1 {
		t.Fatalf("sessions = %+v, want 1 (resumed into same session)", sessions)
	}
	if sessions[0].ID != origID {
		t.Fatalf("session id = %q, want %q", sessions[0].ID, origID)
	}
	if sessions[0].Usage.TotalTokens != 7 {
		t.Fatalf("usage = %+v, want 7", sessions[0].Usage)
	}
	msgs, _ := mgr.Messages(origID)
	if len(msgs) != 4 {
		t.Fatalf("messages = %+v, want 4 (user, assistant, user, assistant)", msgs)
	}
	if msgs[0].Content != "first" || msgs[1].Content != "first answer" ||
		msgs[2].Content != "follow-up" || msgs[3].Content != "second answer" {
		t.Fatalf("messages = %+v", msgs)
	}
}

func TestRecorderResumeRepointsExistingAgent(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)

	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "first"})
	bus.Publish(agent.TopicAgentMessage, agent.AgentMessage{
		AgentID: "a1", AgentName: "agent-one",
		Message: llm.AssistantMessage("first answer"),
	})

	sessions, _, _ := mgr.List(query.Query{})
	if len(sessions) != 1 {
		t.Fatalf("sessions = %+v, want 1", sessions)
	}
	origID := sessions[0].ID

	second, err := mgr.Create(CreateOptions{Provider: "groq", Model: "llama-3.3"})
	if err != nil {
		t.Fatal(err)
	}

	bus.Publish(TopicSessionResume, SessionResume{AgentID: "a1", SessionID: second.ID})
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "switched"})
	bus.Publish(agent.TopicAgentMessage, agent.AgentMessage{
		AgentID: "a1", AgentName: "agent-one",
		Message: llm.AssistantMessage("second answer"),
	})

	firstMsgs, _ := mgr.Messages(origID)
	if len(firstMsgs) != 2 {
		t.Fatalf("first session messages = %+v, want 2 (unchanged)", firstMsgs)
	}
	secondMsgs, _ := mgr.Messages(second.ID)
	if len(secondMsgs) != 2 {
		t.Fatalf("second session messages = %+v, want 2", secondMsgs)
	}
	if secondMsgs[0].Content != "switched" || secondMsgs[1].Content != "second answer" {
		t.Fatalf("second session messages = %+v", secondMsgs)
	}
}

func TestRecorderDetachStartsNewSession(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)

	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "first"})

	sessions, _, _ := mgr.List(query.Query{})
	if len(sessions) != 1 {
		t.Fatalf("sessions = %+v, want 1", sessions)
	}
	origID := sessions[0].ID

	bus.Publish(TopicSessionResume, SessionResume{AgentID: "a1"})
	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "fresh"})
	bus.Publish(agent.TopicAgentMessage, agent.AgentMessage{
		AgentID: "a1", AgentName: "agent-one",
		Message: llm.AssistantMessage("fresh answer"),
	})

	sessions, _, _ = mgr.List(query.Query{})
	if len(sessions) != 2 {
		t.Fatalf("sessions = %+v, want 2 (new session after detach)", sessions)
	}
	var newID string
	for _, s := range sessions {
		if s.ID != origID {
			newID = s.ID
		}
	}
	origMsgs, _ := mgr.Messages(origID)
	if len(origMsgs) != 1 {
		t.Fatalf("original session messages = %+v, want 1 (unchanged)", origMsgs)
	}
	newMsgs, _ := mgr.Messages(newID)
	if len(newMsgs) != 2 {
		t.Fatalf("new session messages = %+v, want 2", newMsgs)
	}
	if newMsgs[0].Content != "fresh" || newMsgs[1].Content != "fresh answer" {
		t.Fatalf("new session messages = %+v", newMsgs)
	}
}

func TestRecorderPublishesSessionAttached(t *testing.T) {
	_, _, bus := newTestRecorder(t)

	var attached SessionAttached
	bus.Subscribe(TopicSessionAttached, func(e SessionAttached) {
		attached = e
	})

	publishStarted(bus, "a1", "agent-one", "groq", "llama-3.3")
	if attached.AgentID != "a1" || attached.SessionID == "" {
		t.Fatalf("attached = %+v, want agent a1 with session", attached)
	}
}

type titleCountingProvider struct {
	mu    sync.Mutex
	calls int
}

func (p *titleCountingProvider) Name() string { return "title" }

func (p *titleCountingProvider) Chat(_ context.Context, _ *llm.Request) (*llm.Response, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	msg := llm.AssistantMessage("Session title")
	fr := llm.FinishReasonStop
	return &llm.Response{Choices: []llm.Choice{{Message: &msg, FinishReason: &fr}}}, nil
}

func (p *titleCountingProvider) ChatStream(context.Context, *llm.Request, llm.StreamHandler) error {
	return nil
}

func (p *titleCountingProvider) ListModels(context.Context) ([]llm.Model, error) { return nil, nil }

func (p *titleCountingProvider) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func TestRecorderGeneratesTitleOnlyOnFirstMessage(t *testing.T) {
	mgr, _, bus := newTestRecorder(t)
	prov := &titleCountingProvider{}

	attached := make(chan string, 1)
	bus.Subscribe(TopicSessionAttached, func(e SessionAttached) {
		attached <- e.SessionID
	})
	bus.Publish(agent.TopicAgentStarted, agent.AgentStarted{
		AgentID:   "a1",
		AgentName: "agent-one",
		Model:     llm.Model{ID: "m"},
		Provider:  prov,
	})

	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "first message"})

	sessID := <-attached
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, err := mgr.Get(sessID)
		if err == nil && s.Title == "Session title" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s, err := mgr.Get(sessID)
	if err != nil || s.Title != "Session title" {
		t.Fatalf("title not generated after first message: %+v, err=%v", s, err)
	}
	if prov.Count() != 1 {
		t.Fatalf("title provider calls = %d, want 1 after first message", prov.Count())
	}

	bus.Publish(agent.TopicAgentInput, agent.AgentInput{AgentID: "a1", Input: "second message"})
	time.Sleep(200 * time.Millisecond)
	if prov.Count() != 1 {
		t.Fatalf("title provider calls = %d, want 1 (title must only be generated on the first message)", prov.Count())
	}
	if s, err := mgr.Get(sessID); err != nil || s.Title != "Session title" {
		t.Fatalf("title changed on follow-up message: %+v, err=%v", s, err)
	}
}
