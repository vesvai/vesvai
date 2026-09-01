package middlewares

import (
	"context"

	agentmw "github.com/vesvai/vesvai/internal/agent/middleware"
	"github.com/vesvai/vesvai/internal/llm"
)

type Redaction struct {
	agentmw.BaseMiddleware
	red           *redactor
	outputEnabled bool
}

func NewRedaction(opts ...RedactionOption) *Redaction {
	red := newRedactor(opts...)
	return &Redaction{red: red, outputEnabled: true}
}

func (r *Redaction) RedactString(s string) string {
	return r.red.Redact(s)
}

func (r *Redaction) BeforeLLM(_ context.Context, req *llm.Request) error {
	if req == nil {
		return nil
	}
	for i := range req.Messages {
		r.red.redactMessage(&req.Messages[i], true)
	}
	return nil
}

func (r *Redaction) AfterLLM(_ context.Context, _ *llm.Request, resp *llm.Response) error {
	if resp == nil || !r.outputEnabled {
		return nil
	}
	for i := range resp.Choices {
		if resp.Choices[i].Message != nil {
			r.red.redactMessage(resp.Choices[i].Message, false)
		}
	}
	return nil
}

func (r *Redaction) AfterRun(_ context.Context, _ string, result *agentmw.Result, _ error) error {
	if result == nil || !r.outputEnabled {
		return nil
	}
	result.Output = r.red.Redact(result.Output)
	for i := range result.History {
		r.red.redactMessage(&result.History[i], true)
	}
	return nil
}
