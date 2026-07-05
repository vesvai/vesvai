package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vesvai/vesvai/internal/config"
)

const (
	memoryFileName = "workspace.json"
)

// FileStore implements MemoryStore using JSON files on disk
type FileStore struct {
	mu       sync.RWMutex
	basePath string
	closed   bool
}

// NewFileStore creates a new FileStore using the global memory directory
func NewFileStore() (*FileStore, error) {
	memoryPath, err := config.GetMemoryDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory dir: %w", err)
	}

	if err := os.MkdirAll(memoryPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	return &FileStore{
		basePath: memoryPath,
	}, nil
}

// NewFileStoreWithPath creates a FileStore with a custom path (for project-level memory)
func NewFileStoreWithPath(basePath string) (*FileStore, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	return &FileStore{
		basePath: basePath,
	}, nil
}

func (s *FileStore) Save(memory *WorkspaceMemory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStoreClosed
	}

	memory.UpdatedAt = time.Now()
	memory.Version++

	filePath := filepath.Join(s.basePath, memoryFileName)

	if err := writeAtomically(filePath, memory); err != nil {
		return fmt.Errorf("failed to write memory: %w", err)
	}

	return nil
}

func (s *FileStore) Load() (*WorkspaceMemory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, ErrStoreClosed
	}

	filePath := filepath.Join(s.basePath, memoryFileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &WorkspaceMemory{
				Facts:   make([]Fact, 0),
				Notes:   make([]Note, 0),
				Version: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to read memory file: %w", err)
	}

	var memory WorkspaceMemory
	if err := json.Unmarshal(data, &memory); err != nil {
		return nil, fmt.Errorf("failed to parse memory file: %w", err)
	}

	return &memory, nil
}

func (s *FileStore) Exists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.basePath, memoryFileName)
	_, err := os.Stat(filePath)
	return err == nil
}

func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	return nil
}

// writeAtomically writes data to a file atomically using tmp+rename pattern
func writeAtomically(path string, data interface{}) error {
	tmpPath := path + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create tmp file: %w", err)
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode data: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close tmp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename tmp file: %w", err)
	}

	return nil
}
