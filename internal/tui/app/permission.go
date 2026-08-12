package app

import (
	"context"

	"github.com/vesvai/vesvai/internal/permission"
)

type permissionRequest struct {
	toolName string
	params   map[string]any
	reason   string
	resp     chan permission.Decision
}

type TUIApprover struct {
	reqs chan permissionRequest
}

func NewTUIApprover() *TUIApprover {
	return &TUIApprover{
		reqs: make(chan permissionRequest, 4),
	}
}

func (a *TUIApprover) RequestApproval(ctx context.Context, toolName string, params map[string]any, reason string) (permission.Decision, error) {
	req := permissionRequest{
		toolName: toolName,
		params:   params,
		reason:   reason,
		resp:     make(chan permission.Decision, 1),
	}

	select {
	case a.reqs <- req:
	case <-ctx.Done():
		return permission.DecisionDeny, ctx.Err()
	}

	select {
	case d := <-req.resp:
		return d, nil
	case <-ctx.Done():
		return permission.DecisionDeny, ctx.Err()
	}
}
