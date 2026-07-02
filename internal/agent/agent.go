package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/llm"
)

type contextKey string

const agentIDKey contextKey = "agent_id"
const sessionIDKey contextKey = "session_id"
const stepKey contextKey = "step"

func WithAgentContext(ctx context.Context, agentID, sessionID string) context.Context {
	ctx = context.WithValue(ctx, agentIDKey, agentID)
	ctx = context.WithValue(ctx, sessionIDKey, sessionID)
	return ctx
}

func AgentIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(agentIDKey).(string); ok {
		return id
	}
	return ""
}

func SessionIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(sessionIDKey).(string); ok {
		return id
	}
	return ""
}

func StepFromContext(ctx context.Context) int {
	if step, ok := ctx.Value(stepKey).(int); ok {
		return step
	}
	return 0
}

type Agent interface {
	Instructions() string
	Tools() []Tool
	Config() AgentConfig
}

type Response struct {
	Content   string
	ToolCalls []ToolCallResult
	Usage     llm.Usage
	Steps     int
}

type ToolCallResult struct {
	ToolName string
	Args     map[string]any
	Result   string
	Error    error
}

type Runner struct {
	provider    llm.Provider
	middlewares *MiddlewareChain
	eventBus    event.EventBus
}

func NewRunner(provider llm.Provider, eventBus event.EventBus, middlewares ...Middleware) *Runner {
	return &Runner{
		provider:    provider,
		middlewares: NewMiddlewareChain(middlewares...),
		eventBus:    eventBus,
	}
}

func (r *Runner) Run(ctx context.Context, agent Agent, userMessage string) (*Response, error) {
	config := agent.Config()

	agentID := config.ID
	if agentID == "" {
		agentID = fmt.Sprintf("%p", agent)
	}

	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	ctx = WithAgentContext(ctx, agentID, sessionID)

	messages := r.buildMessages(agent, userMessage)
	llmTools := r.buildLLMTools(agent)

	var allToolCalls []ToolCallResult
	step := 0

	r.publishEvent(ctx, EventAgentStart, AgentInitEventData{
		AgentID:   agentID,
		AgentType: fmt.Sprintf("%T", agent),
		Config:    config,
	})

	for {
		step++
		ctx = context.WithValue(ctx, stepKey, step)

		if config.MaxSteps > 0 && step > config.MaxSteps {
			return nil, fmt.Errorf("agent exceeded maximum steps (%d)", config.MaxSteps)
		}

		req := llm.NewRequest(config.Model, messages).
			WithTools(llmTools).
			WithTemperature(config.Temperature).
			WithMaxTokens(config.MaxTokens)

		if config.ToolChoice != "" {
			req = req.WithToolChoice(config.ToolChoice)
		}

		r.publishEvent(ctx, EventAgentMessageReceived, MessageEventData{
			AgentID:   agentID,
			Role:      "user",
			Content:   userMessage,
			MessageID: fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), step),
		})

		start := time.Now()
		resp, err := r.provider.Chat(ctx, req)
		if err != nil {
			r.publishEvent(ctx, EventAgentError, AgentErrorEventData{
				AgentID: agentID,
				Error:   err,
			})
			return nil, fmt.Errorf("llm call failed: %w", err)
		}
		_ = time.Since(start)

		r.publishEvent(ctx, EventAgentMessageSent, MessageEventData{
			AgentID:   agentID,
			Role:      "assistant",
			Content:   resp.GetContent(),
			MessageID: fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), step),
		})

		toolCalls := resp.GetToolCalls()
		if len(toolCalls) == 0 {
			r.publishEvent(ctx, EventAgentComplete, TaskEventData{
				AgentID: agentID,
				Status:  "completed",
			})
			return &Response{
				Content:   resp.GetContent(),
				ToolCalls: allToolCalls,
				Usage:     resp.Usage,
				Steps:     step,
			}, nil
		}

		assistantMsg := llm.Message{
			Role:      llm.RoleAssistant,
			Content:   resp.GetContent(),
			ToolCalls: toolCalls,
		}
		messages = append(messages, assistantMsg)

		for _, tc := range toolCalls {
			r.publishEvent(ctx, EventAgentToolCall, ToolEventData{
				AgentID:  agentID,
				ToolName: tc.Function.Name,
				Args:     parseToolArgsSafe(tc.Function.Arguments),
			})

			toolStart := time.Now()
			result, err := r.executeTool(ctx, agent, tc)
			duration := time.Since(toolStart).Milliseconds()

			r.publishEvent(ctx, EventAgentToolResult, ToolEventData{
				AgentID:  agentID,
				ToolName: tc.Function.Name,
				Result:   result,
				Error:    err,
				Duration: duration,
			})

			allToolCalls = append(allToolCalls, ToolCallResult{
				ToolName: tc.Function.Name,
				Args:     parseToolArgsSafe(tc.Function.Arguments),
				Result:   result,
				Error:    err,
			})

			toolResult := result
			if err != nil {
				toolResult = fmt.Sprintf("Error: %s", err.Error())
			}
			messages = append(messages, llm.ToolMessage(toolResult, tc.ID))
		}
	}
}

func (r *Runner) buildMessages(agent Agent, userMessage string) []llm.Message {
	var messages []llm.Message

	config := agent.Config()
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = agent.Instructions()
	}
	if systemPrompt != "" {
		messages = append(messages, llm.SystemMessage(systemPrompt))
	}

	messages = append(messages, llm.UserMessage(userMessage))

	return messages
}

func (r *Runner) buildLLMTools(agent Agent) []llm.Tool {
	agentTools := agent.Tools()
	llmTools := make([]llm.Tool, len(agentTools))
	for i, t := range agentTools {
		llmTools[i] = ToLLMTool(t)
	}
	return llmTools
}

func (r *Runner) executeTool(ctx context.Context, agent Agent, tc llm.ToolCall) (string, error) {
	tool, ok := findTool(agent.Tools(), tc.Function.Name)
	if !ok {
		return "", fmt.Errorf("tool not found: %s", tc.Function.Name)
	}

	params, err := ParseToolArgs(tc.Function.Arguments)
	if err != nil {
		return "", err
	}

	var result string
	var execErr error

	err = r.middlewares.Execute(ctx, agent, func(ctx context.Context) error {
		result, execErr = tool.Handle(ctx, params)
		return execErr
	})

	if err != nil {
		return "", err
	}

	return result, nil
}

func (r *Runner) publishEvent(ctx context.Context, eventType AgentEventType, data any) {
	if r.eventBus == nil {
		return
	}
	if m, ok := data.(interface {
		SetMetadata(agentID, sessionID string, step int)
	}); ok {
		m.SetMetadata(AgentIDFromContext(ctx), SessionIDFromContext(ctx), StepFromContext(ctx))
	}
	r.eventBus.Publish(ctx, NewAgentEvent(eventType, data))
}

type StreamCallback func(chunk StreamChunk) error

type StreamChunk struct {
	Content      string
	Reasoning    string
	ToolCalls    []ToolCallResult
	ToolCall     *ToolCallInfo
	ToolResult   *ToolResultInfo
	FinishReason string
	IsDone       bool
	Usage        *llm.Usage
}

type ToolCallInfo struct {
	ToolName string
	Args     map[string]any
}

type ToolResultInfo struct {
	ToolName string
	Result   string
	Error    error
	Duration int64
}

func (r *Runner) RunStream(ctx context.Context, agent Agent, userMessage string, callback StreamCallback) (*Response, error) {
	config := agent.Config()

	agentID := config.ID
	if agentID == "" {
		agentID = fmt.Sprintf("%p", agent)
	}

	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	ctx = WithAgentContext(ctx, agentID, sessionID)

	messages := r.buildMessages(agent, userMessage)
	llmTools := r.buildLLMTools(agent)

	var allToolCalls []ToolCallResult
	step := 0

	r.publishEvent(ctx, EventAgentStart, AgentInitEventData{
		AgentID:   agentID,
		AgentType: fmt.Sprintf("%T", agent),
		Config:    config,
	})

	for {
		step++
		ctx = context.WithValue(ctx, stepKey, step)

		if config.MaxSteps > 0 && step > config.MaxSteps {
			return nil, fmt.Errorf("agent exceeded maximum steps (%d)", config.MaxSteps)
		}

		req := llm.NewRequest(config.Model, messages).
			WithTools(llmTools).
			WithTemperature(config.Temperature).
			WithMaxTokens(config.MaxTokens)

		if config.ToolChoice != "" {
			req = req.WithToolChoice(config.ToolChoice)
		}

		r.publishEvent(ctx, EventAgentMessageReceived, MessageEventData{
			AgentID:   agentID,
			Role:      "user",
			Content:   userMessage,
			MessageID: fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), step),
		})

		var fullContent string
		var fullReasoning string
		var finalUsage *llm.Usage
		var toolCallIndex = make(map[int]*llm.ToolCall)
		var finishReason llm.FinishReason

		err := r.provider.ChatStream(ctx, req, func(chunk llm.StreamChunk) error {
			if chunk.Content != "" {
				fullContent += chunk.Content
				if err := callback(StreamChunk{
					Content: chunk.Content,
					IsDone:  false,
				}); err != nil {
					return err
				}
			}

			if chunk.Reasoning != "" {
				fullReasoning += chunk.Reasoning
				if err := callback(StreamChunk{
					Reasoning: chunk.Reasoning,
					IsDone:    false,
				}); err != nil {
					return err
				}
			}

			for _, tc := range chunk.ToolCalls {
				idx := tc.Index
				existing, ok := toolCallIndex[idx]
				if !ok {
					existing = &llm.ToolCall{
						Type: "function",
					}
					toolCallIndex[idx] = existing
				}
				if tc.ID != "" {
					existing.ID = tc.ID
				}
				if tc.Function.Name != "" {
					existing.Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					existing.Function.Arguments += tc.Function.Arguments
				}
			}

			if chunk.IsDone {
				finishReason = chunk.FinishReason
				if chunk.Usage != nil {
					finalUsage = chunk.Usage
				}
			}

			return nil
		})

		if err != nil {
			r.publishEvent(ctx, EventAgentError, AgentErrorEventData{
				AgentID: agentID,
				Error:   err,
			})
			return nil, fmt.Errorf("llm stream failed: %w", err)
		}

		if err := callback(StreamChunk{
			Content:      "",
			Reasoning:    "",
			FinishReason: string(finishReason),
			IsDone:       true,
			Usage:        finalUsage,
		}); err != nil {
			return nil, err
		}

		var finalToolCalls []llm.ToolCall
		for i := 0; i < len(toolCallIndex); i++ {
			if tc, ok := toolCallIndex[i]; ok && tc.Function.Name != "" {
				finalToolCalls = append(finalToolCalls, *tc)
			}
		}

		r.publishEvent(ctx, EventAgentMessageSent, MessageEventData{
			AgentID:   agentID,
			Role:      "assistant",
			Content:   fullContent,
			MessageID: fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), step),
		})

		if len(finalToolCalls) == 0 {
			r.publishEvent(ctx, EventAgentComplete, TaskEventData{
				AgentID: agentID,
				Status:  "completed",
			})
			return &Response{
				Content:   fullContent,
				ToolCalls: allToolCalls,
				Usage:     getUsageOrDefault(finalUsage),
				Steps:     step,
			}, nil
		}

		assistantMsg := llm.Message{
			Role:      llm.RoleAssistant,
			Content:   fullContent,
			ToolCalls: finalToolCalls,
		}
		messages = append(messages, assistantMsg)

		for _, tc := range finalToolCalls {
			args := parseToolArgsSafe(tc.Function.Arguments)

			callback(StreamChunk{
				ToolCall: &ToolCallInfo{
					ToolName: tc.Function.Name,
					Args:     args,
				},
			})

			r.publishEvent(ctx, EventAgentToolCall, ToolEventData{
				AgentID:  agentID,
				ToolName: tc.Function.Name,
				Args:     args,
			})

			toolStart := time.Now()
			result, err := r.executeTool(ctx, agent, tc)
			duration := time.Since(toolStart).Milliseconds()

			callback(StreamChunk{
				ToolResult: &ToolResultInfo{
					ToolName: tc.Function.Name,
					Result:   result,
					Error:    err,
					Duration: duration,
				},
			})

			r.publishEvent(ctx, EventAgentToolResult, ToolEventData{
				AgentID:  agentID,
				ToolName: tc.Function.Name,
				Result:   result,
				Error:    err,
				Duration: duration,
			})

			allToolCalls = append(allToolCalls, ToolCallResult{
				ToolName: tc.Function.Name,
				Args:     args,
				Result:   result,
				Error:    err,
			})

			toolResult := result
			if err != nil {
				toolResult = fmt.Sprintf("Error: %s", err.Error())
			}
			messages = append(messages, llm.ToolMessage(toolResult, tc.ID))
		}
	}
}

func getUsageOrDefault(usage *llm.Usage) llm.Usage {
	if usage != nil {
		return *usage
	}
	return llm.Usage{}
}

func findTool(tools []Tool, name string) (Tool, bool) {
	for _, t := range tools {
		if t.Name() == name {
			return t, true
		}
	}
	return nil, false
}

func parseToolArgsSafe(argsJSON string) map[string]any {
	params, _ := ParseToolArgs(argsJSON)
	return params
}
