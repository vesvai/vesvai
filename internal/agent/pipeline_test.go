package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/llm"
)

func TestPipeline_Execute_SingleAgent(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "step 1 output"}},
		},
		Usage: llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	})

	runner := NewRunner(provider, bus)
	agent := newMockAgent("a1")

	pipeline := NewPipeline(runner, agent)
	resp, err := pipeline.Execute(context.Background(), "input")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Content != "step 1 output" {
		t.Errorf("Content = %q", resp.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestPipeline_Execute_MultipleAgents(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "processed by A"}},
			},
			Usage: llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "processed by B"}},
			},
			Usage: llm.Usage{PromptTokens: 20, CompletionTokens: 10, TotalTokens: 30},
		},
	)

	runner := NewRunner(provider, bus)
	a1 := newMockAgent("a1")
	a2 := newMockAgent("a2")

	pipeline := NewPipeline(runner, a1, a2)
	resp, err := pipeline.Execute(context.Background(), "initial")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Content != "processed by B" {
		t.Errorf("Content = %q, want processed by B", resp.Content)
	}
	if resp.Usage.TotalTokens != 45 {
		t.Errorf("TotalTokens = %d, want 45", resp.Usage.TotalTokens)
	}
	if resp.Steps != 2 {
		t.Errorf("Steps = %d, want 2", resp.Steps)
	}
}

func TestPipeline_Execute_AgentError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider._chatErr = errors.New("agent failed")

	runner := NewRunner(provider, bus)
	agent := newMockAgent("fail")

	pipeline := NewPipeline(runner, agent)
	_, err := pipeline.Execute(context.Background(), "input")
	if err == nil {
		t.Fatal("Execute() should error")
	}
	if !errors.Is(err, err) {
		t.Errorf("error = %v", err)
	}
}

func TestPipeline_Execute_Empty(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)

	pipeline := NewPipeline(runner)
	resp, err := pipeline.Execute(context.Background(), "input")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Content != "input" {
		t.Errorf("Content = %q, want input", resp.Content)
	}
}

func TestRouter_Route_FirstRuleMatch(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "code response"}},
			},
		},
	)

	runner := NewRunner(provider, bus)
	codeAgent := newMockAgent("code")
	defaultAgent := newMockAgent("default")

	router := NewRouter(runner, defaultAgent,
		When(func(ctx context.Context, input string) bool {
			return len(input) > 0 && input[0] == '/'
		}, codeAgent),
	)

	resp, err := router.Route(context.Background(), "/help")
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if resp.Content != "code response" {
		t.Errorf("Content = %q, want code response", resp.Content)
	}
}

func TestRouter_Route_DefaultAgent(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "default response"}},
		},
	})

	runner := NewRunner(provider, bus)
	codeAgent := newMockAgent("code")
	defaultAgent := newMockAgent("default")

	router := NewRouter(runner, defaultAgent,
		When(func(ctx context.Context, input string) bool {
			return input == "special"
		}, codeAgent),
	)

	resp, err := router.Route(context.Background(), "normal input")
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if resp.Content != "default response" {
		t.Errorf("Content = %q, want default response", resp.Content)
	}
}

func TestRouter_Route_NoRules(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "default"}},
		},
	})

	runner := NewRunner(provider, bus)
	defaultAgent := newMockAgent("default")

	router := NewRouter(runner, defaultAgent)
	resp, err := router.Route(context.Background(), "any input")
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if resp.Content != "default" {
		t.Errorf("Content = %q", resp.Content)
	}
}

func TestRouter_Route_MultipleRules_FirstMatchWins(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "first match"}},
		},
	})

	runner := NewRunner(provider, bus)
	agent1 := newMockAgent("agent1")
	agent2 := newMockAgent("agent2")
	defaultAgent := newMockAgent("default")

	router := NewRouter(runner, defaultAgent,
		When(func(ctx context.Context, input string) bool { return true }, agent1),
		When(func(ctx context.Context, input string) bool { return true }, agent2),
	)

	resp, err := router.Route(context.Background(), "anything")
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if resp.Content != "first match" {
		t.Errorf("Content = %q, want first match", resp.Content)
	}
}

func TestOrchestrator_RunAll(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "agent1 output"}},
			},
			Usage: llm.Usage{TotalTokens: 10},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "agent2 output"}},
			},
			Usage: llm.Usage{TotalTokens: 20},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "agent3 output"}},
			},
			Usage: llm.Usage{TotalTokens: 30},
		},
	)

	runner := NewRunner(provider, bus)
	agents := []Agent{
		newMockAgent("a1"),
		newMockAgent("a2"),
		newMockAgent("a3"),
	}

	orch := NewOrchestrator(runner, agents...)
	responses, err := orch.RunAll(context.Background(), "shared input")
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if len(responses) != 3 {
		t.Fatalf("responses len = %d, want 3", len(responses))
	}

	contents := make(map[string]bool)
	for _, r := range responses {
		contents[r.Content] = true
	}
	if !contents["agent1 output"] || !contents["agent2 output"] || !contents["agent3 output"] {
		t.Errorf("missing expected outputs: %v", contents)
	}
}

func TestOrchestrator_RunAll_Error(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "ok"}},
			},
		},
	)
	provider._chatErr = errors.New("third agent fails")

	runner := NewRunner(provider, bus)
	agents := []Agent{
		newMockAgent("a1"),
		newMockAgent("a2"),
		newMockAgent("a3"),
	}

	orch := NewOrchestrator(runner, agents...)
	_, err := orch.RunAll(context.Background(), "input")
	if err == nil {
		t.Fatal("RunAll() should error")
	}
}

func TestOrchestrator_RunAll_Concurrent(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()

	provider.setResponses(
		&llm.Response{Choices: []llm.Choice{{Message: &llm.Message{Content: "1"}}}},
		&llm.Response{Choices: []llm.Choice{{Message: &llm.Message{Content: "2"}}}},
		&llm.Response{Choices: []llm.Choice{{Message: &llm.Message{Content: "3"}}}},
		&llm.Response{Choices: []llm.Choice{{Message: &llm.Message{Content: "4"}}}},
		&llm.Response{Choices: []llm.Choice{{Message: &llm.Message{Content: "5"}}}},
	)

	runner := NewRunner(provider, bus)
	agents := make([]Agent, 5)
	for i := range agents {
		agents[i] = newMockAgent("a")
	}

	orch := NewOrchestrator(runner, agents...)
	start := time.Now()
	_, err := orch.RunAll(context.Background(), "concurrent")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}

	if elapsed > 2*time.Second {
		t.Errorf("RunAll took %v, expected concurrent execution", elapsed)
	}
}

func TestNewPipeline(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)

	agents := []Agent{newMockAgent("a1"), newMockAgent("a2")}
	pipeline := NewPipeline(runner, agents...)

	if pipeline.runner != runner {
		t.Error("runner not set")
	}
	if len(pipeline.agents) != 2 {
		t.Errorf("agents len = %d, want 2", len(pipeline.agents))
	}
}

func TestNewRouter(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	defaultAgent := newMockAgent("default")

	rules := []RouterRule{
		{Condition: func(ctx context.Context, input string) bool { return true }, Agent: newMockAgent("r1")},
	}

	router := NewRouter(runner, defaultAgent, rules...)
	if router.runner != runner {
		t.Error("runner not set")
	}
	if router.default_ != defaultAgent {
		t.Error("default_ not set")
	}
	if len(router.rules) != 1 {
		t.Errorf("rules len = %d, want 1", len(router.rules))
	}
}

func TestNewOrchestrator(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)

	agents := []Agent{newMockAgent("a1")}
	orch := NewOrchestrator(runner, agents...)

	if orch.runner != runner {
		t.Error("runner not set")
	}
	if len(orch.agents) != 1 {
		t.Errorf("agents len = %d, want 1", len(orch.agents))
	}
}

func TestWhen(t *testing.T) {
	cond := func(ctx context.Context, input string) bool {
		return input == "match"
	}
	agent := newMockAgent("test")

	rule := When(cond, agent)
	if rule.Condition(context.Background(), "match") != true {
		t.Error("Condition should match")
	}
	if rule.Condition(context.Background(), "no") != false {
		t.Error("Condition should not match")
	}
	if rule.Agent != agent {
		t.Error("Agent not set")
	}
}

func TestRouter_Route_SecondRuleMatch(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "second match"}},
		},
	})

	runner := NewRunner(provider, bus)
	defaultAgent := newMockAgent("default")
	agent1 := newMockAgent("a1")
	agent2 := newMockAgent("a2")

	router := NewRouter(runner, defaultAgent,
		When(func(ctx context.Context, input string) bool { return false }, agent1),
		When(func(ctx context.Context, input string) bool { return true }, agent2),
	)

	resp, err := router.Route(context.Background(), "anything")
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if resp.Content != "second match" {
		t.Errorf("Content = %q, want second match", resp.Content)
	}
}
