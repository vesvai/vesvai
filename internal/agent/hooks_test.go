package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/llm"
)

const hookMarker = "%%skill-test%%"

func registerMarkerHook(t *testing.T) {
	t.Helper()
	OnMessageInput(func(s string) string {
		if strings.Contains(s, hookMarker) {
			return s + " [skill loaded]"
		}
		return s
	})
}

func TestOnMessageInputAppliedOnRun(t *testing.T) {
	registerMarkerHook(t)
	a, prov := newTestAgent(t)
	prov.responses = []mockResponse{{content: "done"}}

	if _, err := a.Run(context.Background(), "use skill "+hookMarker); err != nil {
		t.Fatal(err)
	}

	prov.mu.Lock()
	defer prov.mu.Unlock()
	last := prov.lastReq.Messages[len(prov.lastReq.Messages)-1]
	content, _ := last.Content.(string)
	if last.Role != llm.RoleUser || !strings.Contains(content, "[skill loaded]") {
		t.Fatalf("user message not expanded: %+v", last)
	}
}

func TestOnMessageInputAppliedOnResume(t *testing.T) {
	registerMarkerHook(t)
	a, prov := newTestAgent(t)
	prov.responses = []mockResponse{{content: "done"}}

	history := []llm.Message{llm.UserMessage("previous")}
	if _, err := a.Resume(context.Background(), "follow up "+hookMarker, history); err != nil {
		t.Fatal(err)
	}

	prov.mu.Lock()
	defer prov.mu.Unlock()
	last := prov.lastReq.Messages[len(prov.lastReq.Messages)-1]
	content, _ := last.Content.(string)
	if last.Role != llm.RoleUser || !strings.Contains(content, "[skill loaded]") {
		t.Fatalf("user message not expanded: %+v", last)
	}
}
