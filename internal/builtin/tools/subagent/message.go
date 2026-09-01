package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
)

type HistoryReader interface {
	Messages(sessionID string) ([]session.Message, error)
}

func subAgentMessageTool(reader HistoryReader) tool.Tool {
	return tool.NewSpec(
		"subagent-message",
		"Send a follow-up message to an existing subagent. The subagent continues its previous conversation: it is resumed with its full history plus the new message. Use this when a problem arises with a subagent's earlier work and you need it to fix, extend, or reconsider its task. The subagent must have finished (completed, failed, or interrupted).",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Name of the subagent to message (as given when it was spawned).",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "Follow-up instructions: what changed, what to fix, or what to do next.",
				},
				"background": map[string]any{
					"type":        "boolean",
					"description": "If true, resume the subagent in the background and return immediately. If false (default), wait for it to finish and return its output.",
				},
			},
			"required": []string{"name", "message"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				Name       string `json:"name"`
				Message    string `json:"message"`
				Background bool   `json:"background"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("subagent-message: invalid arguments: %w", err)
			}
			if strings.TrimSpace(params.Name) == "" {
				return "", fmt.Errorf("subagent-message: name is required")
			}
			if strings.TrimSpace(params.Message) == "" {
				return "", fmt.Errorf("subagent-message: message is required")
			}
			if reader == nil {
				return "", fmt.Errorf("subagent-message: session history unavailable")
			}

			parent := agent.FromContext(ctx)
			if parent == nil {
				return "", fmt.Errorf("subagent-message: no parent agent in context")
			}
			if parent.Provider == nil {
				return "", fmt.Errorf("subagent-message: parent agent has no provider configured")
			}

			record, ok := store.get(params.Name)
			if !ok {
				return "", fmt.Errorf("subagent-message: no subagent named %q", params.Name)
			}
			if record.Status == StatusPending || record.Status == StatusRunning {
				return "", fmt.Errorf("subagent-message: %q is still running", params.Name)
			}
			if record.SessionID == "" {
				return "", fmt.Errorf("subagent-message: %q has no recorded session to resume", params.Name)
			}

			sessMsgs, err := reader.Messages(record.SessionID)
			if err != nil {
				return "", fmt.Errorf("subagent-message: read session %q: %w", record.SessionID, err)
			}

			sa, err := store.resume(params.Name)
			if err != nil {
				return "", err
			}

			if parent.Bus != nil {
				store.subscribe(parent.Bus)
			}

			run := func(runCtx context.Context) {
				sub, err := agents.New(record.AgentType)
				if err != nil {
					store.finish(sa, "", err)
					return
				}
				sub.Provider = parent.Provider
				sub.Model = parent.Model

				history := make([]llm.Message, 0, len(sessMsgs)+1)
				if sub.SystemPrompt != "" {
					history = append(history, llm.SystemMessage(sub.SystemPrompt))
				}
				history = append(history, session.MessagesToLLM(sessMsgs)...)

				if parent.Bus != nil {
					sub.Bus = parent.Bus
					store.setAgentID(sa.Name, sub.ID)
					parent.Bus.Publish(session.TopicSessionResume, session.SessionResume{
						AgentID:   sub.ID,
						SessionID: record.SessionID,
					})
					res, runErr := sub.Resume(runCtx, params.Message, history)
					store.finish(sa, agentOutput(res, runErr), runErr)
					return
				}
				store.start(sa)
				res, runErr := sub.Resume(runCtx, params.Message, history)
				store.finish(sa, agentOutput(res, runErr), runErr)
			}

			if params.Background {
				go run(context.Background())
				return fmt.Sprintf("Subagent %q resumed in background with new message.\n", sa.Name), nil
			}

			run(ctx)
			done, _ := store.get(sa.Name)
			if done.Status == StatusFailed {
				return fmt.Sprintf("Subagent %q failed: %s\n", done.Name, done.Err), nil
			}
			return fmt.Sprintf("Subagent %q finished.\nResponse:\n%s\n", done.Name, done.Output), nil
		},
	)
}
