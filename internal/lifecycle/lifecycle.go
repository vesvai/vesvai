package lifecycle

import (
	"context"
	"fmt"
	"sync"

	"github.com/vesvai/vesvai/internal/hook"
)

const (
	HookCreate  = "lifecycle:create"
	HookMount   = "lifecycle:mount"
	HookUnmount = "lifecycle:unmount"
	HookDelete  = "lifecycle:delete"
)

type Phase int

const (
	PhaseNone Phase = iota
	PhaseCreated
	PhaseMounted
	PhaseUnmounted
	PhaseDeleted
)

func (p Phase) String() string {
	switch p {
	case PhaseNone:
		return "none"
	case PhaseCreated:
		return "created"
	case PhaseMounted:
		return "mounted"
	case PhaseUnmounted:
		return "unmounted"
	case PhaseDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

type Lifecycle struct {
	mu         sync.RWMutex
	hooks      *hook.Hooks
	phase      Phase
	registered map[string]Phase
}

func New(hooks *hook.Hooks) *Lifecycle {
	return &Lifecycle{
		hooks:      hooks,
		phase:      PhaseNone,
		registered: make(map[string]Phase),
	}
}

func (l *Lifecycle) Phase() Phase {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.phase
}

func (l *Lifecycle) RegisterComponent(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.registered[name] = PhaseNone
}

func (l *Lifecycle) SetPhase(ctx context.Context, phase Phase) error {
	l.mu.Lock()
	l.phase = phase
	l.mu.Unlock()

	var hookName string
	switch phase {
	case PhaseCreated:
		hookName = HookCreate
	case PhaseMounted:
		hookName = HookMount
	case PhaseUnmounted:
		hookName = HookUnmount
	case PhaseDeleted:
		hookName = HookDelete
	default:
		return fmt.Errorf("invalid phase: %v", phase)
	}

	l.hooks.DoAction(ctx, hookName)
	return nil
}

func (l *Lifecycle) On(hookName string) *hook.HookBuilder {
	return l.hooks.On(hookName)
}

func (l *Lifecycle) Create(ctx context.Context) error {
	return l.SetPhase(ctx, PhaseCreated)
}

func (l *Lifecycle) Mount(ctx context.Context) error {
	return l.SetPhase(ctx, PhaseMounted)
}

func (l *Lifecycle) Unmount(ctx context.Context) error {
	return l.SetPhase(ctx, PhaseUnmounted)
}

func (l *Lifecycle) Delete(ctx context.Context) error {
	return l.SetPhase(ctx, PhaseDeleted)
}

func (l *Lifecycle) ComponentPhase(name string) Phase {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.registered[name]
}

func (l *Lifecycle) SetComponentPhase(name string, phase Phase) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.registered[name] = phase
}

func (l *Lifecycle) Components() map[string]Phase {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make(map[string]Phase, len(l.registered))
	for k, v := range l.registered {
		result[k] = v
	}
	return result
}

func (l *Lifecycle) Hooks() *hook.Hooks {
	return l.hooks
}
