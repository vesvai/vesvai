package agent

import (
	"context"
	"fmt"

	"github.com/vesvai/vesvai/internal/llm"
)

type Pipeline struct {
	agents []Agent
	runner *Runner
}

func NewPipeline(runner *Runner, agents ...Agent) *Pipeline {
	return &Pipeline{
		agents: agents,
		runner: runner,
	}
}

func (p *Pipeline) Execute(ctx context.Context, input string) (*Response, error) {
	current := input
	var totalUsage llm.Usage
	totalSteps := 0

	for i, agent := range p.agents {
		resp, err := p.runner.Run(ctx, agent, current)
		if err != nil {
			return nil, fmt.Errorf("pipeline step %d failed: %w", i, err)
		}

		current = resp.Content
		totalUsage.PromptTokens += resp.Usage.PromptTokens
		totalUsage.CompletionTokens += resp.Usage.CompletionTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens
		totalSteps += resp.Steps
	}

	return &Response{
		Content: current,
		Usage:   totalUsage,
		Steps:   totalSteps,
	}, nil
}

type Router struct {
	rules    []RouterRule
	default_ Agent
	runner   *Runner
}

type RouterRule struct {
	Condition func(ctx context.Context, input string) bool
	Agent     Agent
}

func NewRouter(runner *Runner, defaultAgent Agent, rules ...RouterRule) *Router {
	return &Router{
		rules:    rules,
		default_: defaultAgent,
		runner:   runner,
	}
}

func When(condition func(ctx context.Context, input string) bool, agent Agent) RouterRule {
	return RouterRule{
		Condition: condition,
		Agent:     agent,
	}
}

func (r *Router) Route(ctx context.Context, input string) (*Response, error) {
	for _, rule := range r.rules {
		if rule.Condition(ctx, input) {
			return r.runner.Run(ctx, rule.Agent, input)
		}
	}
	return r.runner.Run(ctx, r.default_, input)
}

type Orchestrator struct {
	agents []Agent
	runner *Runner
}

func NewOrchestrator(runner *Runner, agents ...Agent) *Orchestrator {
	return &Orchestrator{
		agents: agents,
		runner: runner,
	}
}

func (o *Orchestrator) RunAll(ctx context.Context, input string) ([]*Response, error) {
	results := make([]*Response, len(o.agents))
	errs := make([]error, len(o.agents))

	type result struct {
		index int
		resp  *Response
		err   error
	}

	ch := make(chan result, len(o.agents))

	for i, agent := range o.agents {
		go func(idx int, a Agent) {
			resp, err := o.runner.Run(ctx, a, input)
			ch <- result{index: idx, resp: resp, err: err}
		}(i, agent)
	}

	for range o.agents {
		r := <-ch
		results[r.index] = r.resp
		errs[r.index] = r.err
	}

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}
