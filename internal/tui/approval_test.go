package tui

import (
	"context"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/permission"
)

func TestTUIApprover_RequestAndResolve(t *testing.T) {
	approver := NewTUIApprover()

	done := make(chan permission.Decision, 1)
	go func() {
		d, err := approver.RequestApproval(context.Background(), "bash", map[string]any{"command": "ls"}, "test reason")
		if err != nil {
			done <- permission.DecisionDeny
			return
		}
		done <- d
	}()

	select {
	case req := <-approver.Requests():
		if req.ToolName != "bash" {
			t.Errorf("tool name = %q, want bash", req.ToolName)
		}
		req.Result <- permission.DecisionAllow
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for approval request")
	}

	select {
	case d := <-done:
		if d != permission.DecisionAllow {
			t.Errorf("decision = %v, want allow", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for decision")
	}
}

func TestTUIApprover_CancelledContext(t *testing.T) {
	approver := NewTUIApprover()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := approver.RequestApproval(ctx, "bash", nil, "reason")
	if err == nil {
		t.Error("cancelled context should return an error")
	}
}

func TestTUIApprover_QueuedRequests(t *testing.T) {
	approver := NewTUIApprover()

	type result struct {
		idx      int
		decision permission.Decision
	}
	results := make(chan result, 2)

	for i := 0; i < 2; i++ {
		go func(idx int) {
			d, err := approver.RequestApproval(context.Background(), "tool", nil, "reason")
			if err != nil {
				results <- result{idx, permission.DecisionDeny}
				return
			}
			results <- result{idx, d}
		}(i)
	}

	var got []permission.Decision
	for i := 0; i < 2; i++ {
		select {
		case req := <-approver.Requests():
			req.Result <- permission.DecisionAllow
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for request %d", i)
		}
	}

	for i := 0; i < 2; i++ {
		select {
		case r := <-results:
			got = append(got, r.decision)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for result")
		}
	}

	for _, d := range got {
		if d != permission.DecisionAllow {
			t.Errorf("decision = %v, want allow for all", got)
			break
		}
	}
}

func TestTUIApprover_AllowAlways(t *testing.T) {
	approver := NewTUIApprover()

	done := make(chan permission.Decision, 1)
	go func() {
		d, _ := approver.RequestApproval(context.Background(), "bash", map[string]any{"command": "go test"}, "r")
		done <- d
	}()

	select {
	case req := <-approver.Requests():
		req.Result <- permission.DecisionAllowAlways
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	select {
	case d := <-done:
		if d != permission.DecisionAllowAlways {
			t.Errorf("decision = %v, want allow-always", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for decision")
	}
}
