package permission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/llm"
)

var (
	errJudgeUnconfigured = errors.New("permission: judge provider/model not configured")
	errJudgeModelMissing = errors.New("permission: judge model not found")
)

func (m *Middleware) resolveJudge() (llm.Provider, llm.Model, bool) {
	if m.llm == nil {
		return nil, llm.Model{}, false
	}
	var providerName, modelID string
	if m.cfg != nil {
		providerName, modelID = m.cfg.JudgeProvider, m.cfg.JudgeModel
	}

	if providerName != "" && modelID != "" {
		prov, err := m.llm.Provider(providerName)
		if err != nil {
			return nil, llm.Model{}, false
		}
		res := m.llm.Select(llm.SelectRequest{
			Mode:     llm.SelectModeExact,
			Provider: providerName,
			Model:    modelID,
		})
		if res.Err != nil || res.Model.ID == "" {
			return nil, llm.Model{}, false
		}
		return prov, res.Model, true
	}

	res := m.llm.Select(llm.SelectRequest{Mode: llm.SelectModePreferred})
	if res.Err != nil || res.Model.ID == "" {
		return nil, llm.Model{}, false
	}
	prov, err := m.llm.Provider(res.Provider)
	if err != nil {
		return nil, llm.Model{}, false
	}
	return prov, res.Model, true
}

const judgeSystemPrompt = `You are a permission judge for an AI coding assistant. A tool call is proposed. Decide whether it should be allowed.

Consider:
- Whether the operation is reasonable for a coding assistant to perform.
- If a sandbox denial is included, the user has been asked to approve the operation; judge whether it is acceptable anyway.
- Read-only and low-risk operations are usually allowed. Destructive, secret-exposing, or out-of-scope operations should be denied.

Respond with ONLY a JSON object, no other text:
{"allow": true|false, "reason": "short justification"}

The "reason" field is REQUIRED and must be non-empty when "allow" is false.`

type judgeVerdict struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason"`
}

func newJudgeAgent(prov llm.Provider, model llm.Model) *agent.Agent {
	return agent.New("judge",
		agent.WithProvider(prov),
		agent.WithModel(model),
		agent.WithSystemPrompt(judgeSystemPrompt),
		agent.WithMaxIterations(1),
	)
}

func buildJudgePrompt(call llm.ToolCall, permErr error) string {
	var b strings.Builder
	b.WriteString("A tool call requires permission approval.\n\n")
	fmt.Fprintf(&b, "Tool: %s\n", call.Function.Name)
	fmt.Fprintf(&b, "Arguments: %s\n", call.Function.Arguments)
	if permErr != nil {
		fmt.Fprintf(&b, "Sandbox denial: %s\n", permErr.Error())
	}
	b.WriteString("\nDecide whether this tool call should be allowed.")
	return b.String()
}

func (m *Middleware) askJudge(ctx context.Context, call llm.ToolCall, permErr error) (*decision, error) {
	judge := m.judge()
	if judge == nil {
		return nil, fmt.Errorf("judge provider unavailable")
	}
	res, err := judge.Run(ctx, buildJudgePrompt(call, permErr))
	if err != nil {
		return nil, fmt.Errorf("judge request failed: %w", err)
	}
	verdict, err := parseVerdict(res.Output)
	if err != nil {
		return nil, fmt.Errorf("judge returned an invalid response")
	}
	if verdict.Allow {
		return &decision{Allow: true}, nil
	}
	reason := strings.TrimSpace(verdict.Reason)
	if reason == "" {
		reason = "judge denied the tool call"
	}
	return &decision{Allow: false, Reason: reason}, nil
}

func parseVerdict(output string) (judgeVerdict, error) {
	var verdict judgeVerdict
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &verdict); err == nil {
		return verdict, nil
	}
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start < 0 || end <= start {
		return verdict, fmt.Errorf("no JSON object found")
	}
	if err := json.Unmarshal([]byte(output[start:end+1]), &verdict); err != nil {
		return verdict, err
	}
	return verdict, nil
}
