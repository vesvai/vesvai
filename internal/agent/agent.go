package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vesvai/vesvai/internal/agent/middleware"
	"github.com/vesvai/vesvai/internal/agent/middlewares"
	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/core/logger"
	"github.com/vesvai/vesvai/internal/llm"
)

const DefaultMaxIterations = 150

type Option func(*Agent)

type Agent struct {
	ID              string
	Name            string
	Description     string
	Model           llm.Model
	Provider        llm.Provider
	SystemPrompt    string
	Temperature     float64
	TopP            float64
	MaxTokens       int
	MaxIterations   int
	Attachments     []llm.Attachment
	Tools           *tool.Registry
	ToolNames       []string
	ReasoningEffort string

	structuredName   string
	structuredSchema any

	chain           *middleware.Chain
	middlewareNames []string
	namesResolved   bool
	Bus             event.Bus
	log             *logger.Logger
}

func New(name string, opts ...Option) *Agent {
	if name == "" {
		name = "agent"
	}
	a := &Agent{
		ID:            uuid.NewString(),
		Name:          name,
		Tools:         tool.NewRegistry(),
		MaxIterations: DefaultMaxIterations,
		chain:         middleware.NewChain(),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func WithDescription(desc string) Option {
	return func(a *Agent) { a.Description = desc }
}

func WithModel(model llm.Model) Option {
	return func(a *Agent) { a.Model = model }
}

func WithProvider(p llm.Provider) Option {
	return func(a *Agent) { a.Provider = p }
}

func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) { a.SystemPrompt = prompt }
}

func WithTemperature(temp float64) Option {
	return func(a *Agent) { a.Temperature = temp }
}

func WithTopP(topP float64) Option {
	return func(a *Agent) { a.TopP = topP }
}

func WithMaxTokens(maxTokens int) Option {
	return func(a *Agent) { a.MaxTokens = maxTokens }
}

func WithAttachments(atts ...llm.Attachment) Option {
	return func(a *Agent) {
		if len(atts) > 0 {
			a.Attachments = append(a.Attachments, atts...)
		}
	}
}

func WithMaxIterations(maxIterations int) Option {
	return func(a *Agent) {
		if maxIterations > 0 {
			a.MaxIterations = maxIterations
		}
	}
}

func WithTool(t tool.Tool) Option {
	return func(a *Agent) {
		if t != nil {
			_ = a.Tools.Register(t)
		}
	}
}

func WithTools(ts ...tool.Tool) Option {
	return func(a *Agent) {
		for _, t := range ts {
			if t != nil {
				_ = a.Tools.Register(t)
			}
		}
	}
}

func WithToolNames(names ...string) Option {
	return func(a *Agent) {
		for _, n := range names {
			if n != "" {
				a.ToolNames = append(a.ToolNames, n)
			}
		}
	}
}

func WithMiddleware(mws ...middleware.Middleware) Option {
	return func(a *Agent) { a.chain.Append(mws...) }
}

func WithMiddlewareNames(names ...string) Option {
	return func(a *Agent) {
		for _, n := range names {
			if n != "" {
				a.middlewareNames = append(a.middlewareNames, n)
			}
		}
	}
}

func WithBus(b event.Bus) Option {
	return func(a *Agent) { a.Bus = b }
}

func WithLogger(l *logger.Logger) Option {
	return func(a *Agent) { a.log = l }
}

func WithReasoningEffort(effort string) Option {
	return func(a *Agent) { a.ReasoningEffort = effort }
}

// WithStructuredOutput makes the agent request a JSON response matching the
// given JSON schema from the LLM provider.
func WithStructuredOutput(name string, schema any) Option {
	return func(a *Agent) {
		a.structuredName = name
		a.structuredSchema = schema
	}
}

func (a *Agent) Run(ctx context.Context, input string) (*RunResult, error) {
	return a.run(ctx, input, nil)
}

func (a *Agent) Resume(ctx context.Context, input string, history []llm.Message) (*RunResult, error) {
	return a.resume(ctx, input, history, nil)
}

func (a *Agent) ResumeStream(ctx context.Context, input string, history []llm.Message, handler StreamHandler) (*RunResult, error) {
	return a.resume(ctx, input, history, handler)
}

func (a *Agent) RunStream(ctx context.Context, input string, handler StreamHandler) (*RunResult, error) {
	return a.run(ctx, input, handler)
}

func (a *Agent) publish(topic string, payload any) {
	if a.Bus == nil {
		return
	}
	a.Bus.Publish(topic, payload)
}

func (a *Agent) debugf(format string, args ...any) {
	if a.log == nil {
		return
	}
	a.log.Fdebug(format, args...)
}

func (a *Agent) resolveToolNames() error {
	for _, name := range a.ToolNames {
		if _, ok := a.Tools.Get(name); ok {
			continue
		}
		t, ok := tools.Get(name)
		if !ok {
			return fmt.Errorf("%w: %q", tool.ErrNotFound, name)
		}
		_ = a.Tools.Register(t)
	}
	return nil
}

func (a *Agent) resolveMiddlewareNames() error {
	if a.namesResolved {
		return nil
	}
	for _, name := range a.middlewareNames {
		m, ok := middlewares.Get(name)
		if !ok {
			return fmt.Errorf("%w: %q", middleware.ErrNotFound, name)
		}
		a.chain.Append(m)
	}
	a.namesResolved = true
	return nil
}
