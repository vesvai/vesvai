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
const toolNameKey contextKey = "tool_name"
const toolParamsKey contextKey = "tool_params"
const modelKey contextKey = "model"

const historyKey contextKey = "history"

const extraSystemKey contextKey = "extra_system"

func WithExtraSystemContext(ctx context.Context, extra []llm.Message) context.Context {
	if len(extra) == 0 {
		return ctx
	}
	return context.WithValue(ctx, extraSystemKey, extra)
}

func ExtraSystemFromContext(ctx context.Context) []llm.Message {
	if e, ok := ctx.Value(extraSystemKey).([]llm.Message); ok {
		return e
	}
	return nil
}

func WithHistoryContext(ctx context.Context, history []llm.Message) context.Context {
	if len(history) == 0 {
		return ctx
	}
	return context.WithValue(ctx, historyKey, history)
}

func HistoryFromContext(ctx context.Context) []llm.Message {
	if h, ok := ctx.Value(historyKey).([]llm.Message); ok {
		return h
	}
	return nil
}

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

func ToolNameFromContext(ctx context.Context) string {
	if name, ok := ctx.Value(toolNameKey).(string); ok {
		return name
	}
	return ""
}

func ToolParamsFromContext(ctx context.Context) map[string]any {
	if params, ok := ctx.Value(toolParamsKey).(map[string]any); ok {
		return params
	}
	return nil
}

func WithToolContext(ctx context.Context, toolName string, params map[string]any) context.Context {
	if toolName != "" {
		ctx = context.WithValue(ctx, toolNameKey, toolName)
	}
	if params != nil {
		ctx = context.WithValue(ctx, toolParamsKey, params)
	}
	return ctx
}

func WithModelContext(ctx context.Context, model string) context.Context {
	if model == "" {
		return ctx
	}
	return context.WithValue(ctx, modelKey, model)
}

func ModelFromContext(ctx context.Context) string {
	if m, ok := ctx.Value(modelKey).(string); ok {
		return m
	}
	return ""
}

type Agent interface {
	Instructions() string
	ToolNames() []string
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
	provider     llm.Provider
	middlewares  *MiddlewareChain
	eventBus     event.EventBus
	toolRegistry *ToolRegistry
	subagents    *SubagentManager
	mailbox      *Mailbox
}

func NewRunner(provider llm.Provider, eventBus event.EventBus, toolRegistry *ToolRegistry, middlewares ...Middleware) *Runner {
	return &Runner{
		provider:     provider,
		middlewares:  NewMiddlewareChain(middlewares...),
		eventBus:     eventBus,
		toolRegistry: toolRegistry,
		subagents:    NewSubagentManager(),
		mailbox:      NewMailbox(),
	}
}

func (r *Runner) Subagents() *SubagentManager {
	return r.subagents
}

func (r *Runner) Mailbox() *Mailbox {
	return r.mailbox
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
	ctx = WithModelContext(ctx, config.Model)

	messages := r.buildMessages(agent, userMessage, ctx)
	llmTools := r.buildLLMTools(agent)

	var allToolCalls []ToolCallResult
	var fullContent string
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

		req := llm.NewRequest(resolveModel(ctx, config), messages).
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

		stepContent := resp.GetContent()
		if stepContent != "" {
			if fullContent != "" {
				fullContent += "\n\n"
			}
			fullContent += stepContent
		}

		r.publishEvent(ctx, EventAgentMessageSent, MessageEventData{
			AgentID:   agentID,
			Role:      "assistant",
			Content:   stepContent,
			MessageID: fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), step),
		})

		toolCalls := resp.GetToolCalls()
		if len(toolCalls) == 0 {
			r.publishEvent(ctx, EventAgentComplete, TaskEventData{
				AgentID: agentID,
				Status:  "completed",
			})
			return &Response{
				Content:   fullContent,
				ToolCalls: allToolCalls,
				Usage:     resp.Usage,
				Steps:     step,
			}, nil
		}

		assistantMsg := llm.Message{
			Role:      llm.RoleAssistant,
			Content:   stepContent,
			ToolCalls: toolCalls,
		}
		messages = append(messages, assistantMsg)

		for _, tc := range toolCalls {
			r.publishEvent(ctx, EventAgentToolCall, &ToolEventData{
				AgentID:  agentID,
				ToolName: tc.Function.Name,
				Args:     parseToolArgsSafe(tc.Function.Arguments),
			})

			toolStart := time.Now()
			result, err := r.executeTool(ctx, agent, tc, nil)
			duration := time.Since(toolStart).Milliseconds()

			r.publishEvent(ctx, EventAgentToolResult, &ToolEventData{
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

func (r *Runner) buildMessages(agent Agent, userContent any, ctx context.Context) []llm.Message {
	var messages []llm.Message

	config := agent.Config()
	systemPrompt := r.resolvePrompt(config)
	if systemPrompt == "" {
		systemPrompt = agent.Instructions()
	}
	if systemPrompt != "" {
		messages = append(messages, llm.SystemMessage(systemPrompt))
	}

	if extra := ExtraSystemFromContext(ctx); len(extra) > 0 {
		messages = append(messages, extra...)
	}

	if history := HistoryFromContext(ctx); len(history) > 0 {
		messages = append(messages, history...)
	}

	messages = append(messages, llm.UserMessage(userContent))

	return messages
}

func (r *Runner) resolvePrompt(config AgentConfig) string {
	if len(config.PromptVariants) == 0 {
		return config.SystemPrompt
	}

	matcher := NewPromptMatcher(config.SystemPrompt, config.PromptVariants)
	return matcher.Match(config.Model)
}

func resolveModel(ctx context.Context, config AgentConfig) string {
	if config.Model != "" {
		return config.Model
	}
	return ModelFromContext(ctx)
}

func (r *Runner) buildLLMTools(agent Agent) []llm.Tool {
	agentTools := r.resolveTools(agent)
	llmTools := make([]llm.Tool, len(agentTools))
	for i, t := range agentTools {
		llmTools[i] = ToLLMTool(t)
	}
	return llmTools
}

func (r *Runner) executeTool(ctx context.Context, agent Agent, tc llm.ToolCall, callback StreamCallback) (string, error) {
	tool, ok := r.resolveTool(agent, tc.Function.Name)
	if !ok {
		return "", fmt.Errorf("tool not found: %s", tc.Function.Name)
	}

	params, err := ParseToolArgs(tc.Function.Arguments)
	if err != nil {
		return "", err
	}

	ctx = context.WithValue(ctx, toolNameKey, tc.Function.Name)
	ctx = context.WithValue(ctx, toolParamsKey, params)

	var result string
	var execErr error

	err = r.middlewares.Execute(ctx, agent, func(ctx context.Context) error {
		if st, ok := tool.(interface {
			HandleStream(ctx context.Context, params map[string]any, callback StreamCallback) (string, error)
		}); ok && callback != nil {
			result, execErr = st.HandleStream(ctx, params, callback)
		} else {
			result, execErr = tool.Handle(ctx, params)
		}
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

type SubagentStartInfo struct {
	Name   string
	Prompt string
}

type SubagentDoneInfo struct {
	Name     string
	Result   string
	Error    error
	Duration time.Duration
}

type StreamChunk struct {
	Content      string
	Reasoning    string
	ToolCalls    []ToolCallResult
	ToolCall     *ToolCallInfo
	ToolResult   *ToolResultInfo
	FinishReason string
	IsDone       bool
	Final        bool
	Usage        *llm.Usage

	SubagentID    string
	SubagentStart *SubagentStartInfo
	SubagentDone  *SubagentDoneInfo
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
	return r.RunStreamContent(ctx, agent, userMessage, callback)
}

func (r *Runner) RunStreamContent(ctx context.Context, agent Agent, userContent any, callback StreamCallback) (*Response, error) {
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
	ctx = WithModelContext(ctx, config.Model)

	messages := r.buildMessages(agent, userContent, ctx)
	llmTools := r.buildLLMTools(agent)

	var allToolCalls []ToolCallResult
	var fullContent string
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

		req := llm.NewRequest(resolveModel(ctx, config), messages).
			WithTools(llmTools).
			WithTemperature(config.Temperature).
			WithMaxTokens(config.MaxTokens)

		if config.ToolChoice != "" {
			req = req.WithToolChoice(config.ToolChoice)
		}

		r.publishEvent(ctx, EventAgentMessageReceived, MessageEventData{
			AgentID:   agentID,
			Role:      "user",
			Content:   contentText(userContent),
			MessageID: fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), step),
		})

		var stepContent string
		var finalUsage *llm.Usage
		var toolCallIndex = make(map[int]*llm.ToolCall)
		var finishReason llm.FinishReason

		err := r.provider.ChatStream(ctx, req, func(chunk llm.StreamChunk) error {
			if chunk.Content != "" {
				stepContent += chunk.Content
				if err := callback(StreamChunk{
					Content: chunk.Content,
					IsDone:  false,
				}); err != nil {
					return err
				}
			}

			if chunk.Reasoning != "" {
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

		if stepContent != "" {
			if fullContent != "" {
				fullContent += "\n\n"
			}
			fullContent += stepContent
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
			Content:   stepContent,
			MessageID: fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), step),
		})

		if len(finalToolCalls) == 0 {
			if err := callback(StreamChunk{
				Content:      "",
				Reasoning:    "",
				FinishReason: string(finishReason),
				IsDone:       true,
				Final:        true,
				Usage:        finalUsage,
			}); err != nil {
				return nil, err
			}
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
			Content:   stepContent,
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

			r.publishEvent(ctx, EventAgentToolCall, &ToolEventData{
				AgentID:  agentID,
				ToolName: tc.Function.Name,
				Args:     args,
			})

			toolStart := time.Now()
			result, err := r.executeTool(ctx, agent, tc, callback)
			duration := time.Since(toolStart).Milliseconds()

			callback(StreamChunk{
				ToolResult: &ToolResultInfo{
					ToolName: tc.Function.Name,
					Result:   result,
					Error:    err,
					Duration: duration,
				},
			})

			r.publishEvent(ctx, EventAgentToolResult, &ToolEventData{
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

		if err := callback(StreamChunk{
			Content:      "",
			Reasoning:    "",
			FinishReason: string(finishReason),
			IsDone:       true,
			Usage:        finalUsage,
		}); err != nil {
			return nil, err
		}
	}
}

func getUsageOrDefault(usage *llm.Usage) llm.Usage {
	if usage != nil {
		return *usage
	}
	return llm.Usage{}
}

func (r *Runner) resolveTools(a Agent) []Tool {
	names := a.ToolNames()
	if r.toolRegistry == nil {
		return nil
	}
	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		if t, ok := r.toolRegistry.Get(name); ok {
			tools = append(tools, t)
		}
	}
	return tools
}

func (r *Runner) resolveTool(a Agent, name string) (Tool, bool) {
	if r.toolRegistry == nil {
		return nil, false
	}
	t, ok := r.toolRegistry.Get(name)
	return t, ok
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

func contentText(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case llm.Content:
		return c.Text
	case *llm.Content:
		if c == nil {
			return ""
		}
		return c.Text
	}
	return fmt.Sprintf("%v", content)
}
