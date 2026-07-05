package plugin

import (
	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/hook"
)

type PluginState int

const (
	StateRegistered PluginState = iota
	StateInitialized
	StateRunning
	StateStopped
	StateError
)

func (s PluginState) String() string {
	switch s {
	case StateRegistered:
		return "registered"
	case StateInitialized:
		return "initialized"
	case StateRunning:
		return "running"
	case StateStopped:
		return "stopped"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

type PluginMeta struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Author       string   `json:"author,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type PluginContext struct {
	Hooks        *hook.Hooks
	EventBus     event.EventBus
	ToolRegistry *agent.ToolRegistry
	Config       map[string]any
}

type Plugin interface {
	Meta() PluginMeta
	Init(ctx PluginContext) error
	Start() error
	Stop() error
}

type PluginFactory func() Plugin

type ToolProvider interface {
	Tools() []agent.Tool
}

type AgentProvider interface {
	Agents() []agent.Agent
}

type PromptProvider interface {
	Prompts() map[string]string
}

type HookRegistrar interface {
	RegisterHooks(hooks *hook.Hooks)
}

type EventSubscriber interface {
	SubscribeEvents(bus event.EventBus)
}

type DiskPluginMeta struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Author       string   `json:"author,omitempty"`
	Entry        string   `json:"entry"`
	Dependencies []string `json:"dependencies,omitempty"`
}

func (d *DiskPluginMeta) ToPluginMeta() PluginMeta {
	return PluginMeta{
		Name:         d.Name,
		Version:      d.Version,
		Description:  d.Description,
		Author:       d.Author,
		Dependencies: d.Dependencies,
	}
}
