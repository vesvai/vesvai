package tui

import (
	"context"

	"github.com/vesvai/vesvai/internal/permission"
)

type ApprovalRequest struct {
	ToolName string
	Args     map[string]any
	Reason   string
	Result   chan permission.Decision
}

type TUIApprover struct {
	requests chan ApprovalRequest
}

func NewTUIApprover() *TUIApprover {
	return &TUIApprover{
		requests: make(chan ApprovalRequest, 8),
	}
}

func (a *TUIApprover) RequestApproval(ctx context.Context, toolName string, params map[string]any, reason string) (permission.Decision, error) {
	req := ApprovalRequest{
		ToolName: toolName,
		Args:     params,
		Reason:   reason,
		Result:   make(chan permission.Decision, 1),
	}

	select {
	case a.requests <- req:
	case <-ctx.Done():
		return permission.DecisionDeny, ctx.Err()
	}

	select {
	case d := <-req.Result:
		return d, nil
	case <-ctx.Done():
		return permission.DecisionDeny, ctx.Err()
	}
}

func (a *TUIApprover) Requests() <-chan ApprovalRequest {
	return a.requests
}
