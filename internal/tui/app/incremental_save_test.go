package app

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/layouts"
)

type blockingBackend struct {
	*stubBackend
	started chan struct{}
	release chan struct{}
	saves   atomic.Int32
	once    sync.Once
}

func (b *blockingBackend) SaveSession(s *session.Session) error {
	b.saves.Add(1)
	b.once.Do(func() { close(b.started) })
	<-b.release
	return nil
}

func newBlockingApp() (*App, *blockingBackend) {
	b := &blockingBackend{
		stubBackend: &stubBackend{},
		started:     make(chan struct{}),
		release:     make(chan struct{}),
	}
	d := &stubDriver{stubBackend: b.stubBackend}
	a := NewWithDriver(d)
	a.backend = b
	a.model = tui.NewModel("demo")
	a.layout = layouts.NewMainLayout(a.model, tui.DefaultDark())
	return a, b
}

func TestPeriodicSaveSkippedWhileInFlight(t *testing.T) {
	a, b := newBlockingApp()
	a.model.Conv.AddUser("hello")
	a.model.Busy = true

	a.saveSessionBestEffort()
	<-b.started

	a.saveSessionBestEffort()

	close(b.release)
	a.saveWG.Wait()

	if got := b.saves.Load(); got != 1 {
		t.Fatalf("saves = %d, want 1 (periodic dropped while in flight)", got)
	}
}

func TestFinalSaveQueuedWhileInFlight(t *testing.T) {
	a, b := newBlockingApp()
	a.model.Conv.AddUser("hello")
	a.model.Busy = true

	a.saveSessionBestEffort()
	<-b.started

	a.saveSession()

	close(b.release)
	a.saveWG.Wait()

	if got := b.saves.Load(); got != 2 {
		t.Fatalf("saves = %d, want 2 (final queued after in-flight)", got)
	}
}

func TestQueuedSaveKeepsLatestSnapshot(t *testing.T) {
	a, b := newBlockingApp()
	a.model.Conv.AddUser("first")
	a.model.Busy = true

	a.saveSessionBestEffort()
	<-b.started

	a.model.Conv.AddUser("second")
	a.saveSession()
	a.model.Conv.AddUser("third")
	a.saveSession()

	close(b.release)
	a.saveWG.Wait()

	if got := b.saves.Load(); got != 2 {
		t.Fatalf("saves = %d, want 2 (coalesced to latest)", got)
	}
}

func TestConvToMessagesSkipsInFlightTool(t *testing.T) {
	conv := &tui.Conversation{}
	conv.AddUser("hi")
	conv.StartAssistant()
	done := conv.AddToolCall("read", map[string]any{"path": "x"})
	conv.FinishToolCall(done, "contents", nil, time.Millisecond)
	conv.AddToolCall("bash", map[string]any{"command": "sleep 100"})

	msgs := ConvToMessages(conv)

	calls := 0
	results := 0
	for _, m := range msgs {
		switch m.Role {
		case llm.RoleAssistant:
			calls += len(m.ToolCalls)
			for _, tc := range m.ToolCalls {
				if tc.Function.Name != "read" {
					t.Fatalf("unexpected in-flight tool call persisted: %s", tc.Function.Name)
				}
			}
		case llm.RoleTool:
			results++
		}
	}
	if calls != 1 {
		t.Fatalf("assistant tool calls = %d, want 1 (finished only)", calls)
	}
	if results != 1 {
		t.Fatalf("tool results = %d, want 1", results)
	}
}

func TestMessagesToConvMarksInterruptedTool(t *testing.T) {
	conv := &tui.Conversation{}
	msgs := []llm.Message{
		llm.NewMessage(llm.RoleUser, "hi"),
		{
			Role:    llm.RoleAssistant,
			Content: "working",
			ToolCalls: []llm.ToolCall{
				{Type: "function", ID: "call_1", Function: llm.Function{Name: "bash", Arguments: "{}"}},
			},
		},
	}

	MessagesToConv(conv, msgs)

	cur := conv.Current()
	if cur == nil || len(cur.Tools) != 1 {
		t.Fatalf("loaded tools = %+v, want 1", cur.Tools)
	}
	tc := cur.Tools[0]
	if tc.State != tui.ToolError {
		t.Fatalf("tool state = %v, want ToolError", tc.State)
	}
	if tc.Error == nil || tc.Error.Error() != "interrupted" {
		t.Fatalf("tool error = %v, want 'interrupted'", tc.Error)
	}
}
