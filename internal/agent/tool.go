package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/vesvai/vesvai/internal/llm"
)

type Tool interface {
	Name() string

	Description() string

	Schema() map[string]any

	Handle(ctx context.Context, params map[string]any) (string, error)
}

type FuncTool struct {
	toolName        string
	toolDescription string
	toolSchema      map[string]any
	handler         func(ctx context.Context, params map[string]any) (string, error)
}

func NewFuncTool(name, description string, schema map[string]any, handler func(ctx context.Context, params map[string]any) (string, error)) *FuncTool {
	return &FuncTool{
		toolName:        name,
		toolDescription: description,
		toolSchema:      schema,
		handler:         handler,
	}
}

func (t *FuncTool) Name() string           { return t.toolName }
func (t *FuncTool) Description() string    { return t.toolDescription }
func (t *FuncTool) Schema() map[string]any { return t.toolSchema }

func (t *FuncTool) Handle(ctx context.Context, params map[string]any) (string, error) {
	return t.handler(ctx, params)
}

func ToLLMTool(t Tool) llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		},
	}
}

func ParseToolArgs(argsJSON string) (map[string]any, error) {
	var params map[string]any
	if argsJSON == "" || argsJSON == "{}" {
		return params, nil
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}
	return params, nil
}

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
	order []string // preserve insertion order
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := t.Name()
	if _, exists := r.tools[name]; exists {
		panic(fmt.Sprintf("tool already registered: %s", name))
	}
	r.tools[name] = t
	r.order = append(r.order, name)
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

func (r *ToolRegistry) All() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.tools[name])
	}
	return result
}

func (r *ToolRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

func (r *ToolRegistry) ToLLMTools() []llm.Tool {
	tools := r.All()
	llmTools := make([]llm.Tool, len(tools))
	for i, t := range tools {
		llmTools[i] = ToLLMTool(t)
	}
	return llmTools
}

func (r *ToolRegistry) Merge(other *ToolRegistry) {
	for _, t := range other.All() {
		r.Register(t)
	}
}
