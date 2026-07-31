package permission

import (
	"context"
	"fmt"

	"github.com/vesvai/vesvai/internal/agent"
)

type Decision int

const (
	DecisionDeny Decision = iota
	DecisionAllow
	DecisionAllowAlways
)

type Approver interface {
	RequestApproval(ctx context.Context, toolName string, params map[string]any, reason string) (Decision, error)
}

type ApproverFunc func(ctx context.Context, toolName string, params map[string]any, reason string) (Decision, error)

func (f ApproverFunc) RequestApproval(ctx context.Context, toolName string, params map[string]any, reason string) (Decision, error) {
	return f(ctx, toolName, params, reason)
}

var NopApprover ApproverFunc = func(ctx context.Context, _ string, _ map[string]any, _ string) (Decision, error) {
	return DecisionDeny, nil
}

func PermissionMiddleware(mgr *Manager, approver Approver) agent.Middleware {
	return func(ctx context.Context, agnt agent.Agent, next agent.MiddlewareFunc) error {
		toolName := agent.ToolNameFromContext(ctx)
		params := agent.ToolParamsFromContext(ctx)

		if toolName == "" {
			return next(ctx)
		}

		resolution := mgr.Check(toolName, params)

		switch resolution.Action {
		case ActionAllow:
			return next(ctx)

		case ActionDeny:
			return fmt.Errorf("permission denied: %s", resolution.Reason)

		case ActionPrompt:
			if approver == nil {
				return fmt.Errorf("permission denied (no approver available): %s", resolution.Reason)
			}

			decision, err := approver.RequestApproval(ctx, toolName, params, resolution.Reason)
			if err != nil {
				return fmt.Errorf("approval request failed: %w", err)
			}

			switch decision {
			case DecisionAllow:
				return next(ctx)

			case DecisionAllowAlways:
				if saveErr := mgr.AddAllowRule(toolName, params); saveErr != nil {
					_ = saveErr
				}
				return next(ctx)

			default:
				return fmt.Errorf("permission denied by user")
			}
		}

		return next(ctx)
	}
}

type TerminalApprover struct{}

func (TerminalApprover) RequestApproval(ctx context.Context, toolName string, params map[string]any, reason string) (Decision, error) {
	_ = reason
	fmt.Printf("[permission] tool=%s params=%v — [a]llow / [A]llow-always / [d]eny: ", toolName, params)
	var resp string
	fmt.Scanln(&resp)
	switch resp {
	case "a", "A", "y", "yes":
		if resp == "A" {
			return DecisionAllowAlways, nil
		}
		return DecisionAllow, nil
	default:
		return DecisionDeny, nil
	}
}
