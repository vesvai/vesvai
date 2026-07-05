package memory

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/lifecycle"
)

type Manager struct {
	mu     sync.RWMutex
	store  *FileStore
	memory *Memory
	lc     *lifecycle.Lifecycle
}

func NewManager(lc *lifecycle.Lifecycle) *Manager {
	return &Manager{
		lc: lc,
	}
}

func (m *Manager) RegisterHooks() {
	m.lc.On(lifecycle.HookCreate).Priority(70).Do(m.onCreate)
	m.lc.On(lifecycle.HookMount).Priority(70).Do(m.onMount)
	m.lc.On(lifecycle.HookUnmount).Priority(70).Do(m.onUnmount)
	m.lc.On(lifecycle.HookDelete).Priority(70).Do(m.onDelete)
}

func (m *Manager) onCreate(ctx context.Context, args ...interface{}) error {
	memoryPath, err := config.GetMemoryDir()
	if err != nil {
		return fmt.Errorf("failed to get memory dir: %w", err)
	}

	if err := os.MkdirAll(memoryPath, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	store, err := NewFileStore()
	if err != nil {
		return fmt.Errorf("failed to create file store: %w", err)
	}

	m.mu.Lock()
	m.store = store
	m.mu.Unlock()

	m.lc.SetComponentPhase("memory", lifecycle.PhaseCreated)
	return nil
}

func (m *Manager) onMount(ctx context.Context, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.store == nil {
		return fmt.Errorf("memory store not initialized")
	}

	mem := New(m.store)
	if err := mem.Load(); err != nil {
		return fmt.Errorf("failed to load memory: %w", err)
	}

	m.memory = mem
	m.lc.SetComponentPhase("memory", lifecycle.PhaseMounted)
	return nil
}

func (m *Manager) onUnmount(ctx context.Context, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.memory != nil {
		if err := m.memory.Save(); err != nil {
			fmt.Printf("Memory: failed to save on unmount: %v\n", err)
		}
	}

	if m.store != nil {
		m.store.Close()
	}

	m.lc.SetComponentPhase("memory", lifecycle.PhaseUnmounted)
	return nil
}

func (m *Manager) onDelete(ctx context.Context, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store = nil
	m.memory = nil
	m.lc.SetComponentPhase("memory", lifecycle.PhaseDeleted)
	return nil
}

func (m *Manager) Memory() *Memory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.memory
}

func (m *Manager) Store() *FileStore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store
}
