package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExists   = errors.New("session already exists")
	ErrStoreClosed     = errors.New("store is closed")
)

type FileStore struct {
	mu       sync.RWMutex
	basePath string
	index    *sessionIndex
	closed   bool
}

type sessionIndex struct {
	mu       sync.RWMutex
	sessions []SessionMetadataIndex
	dirty    bool
}

func NewFileStore(basePath string) (*FileStore, error) {
	sessionsPath := filepath.Join(basePath, "sessions")
	if err := os.MkdirAll(sessionsPath, 0755); err != nil {
		return nil, fmt.Errorf("create sessions directory: %w", err)
	}

	store := &FileStore{
		basePath: sessionsPath,
		index: &sessionIndex{
			sessions: make([]SessionMetadataIndex, 0, 64),
		},
	}

	if err := store.loadIndex(); err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}

	return store, nil
}

func (s *FileStore) Save(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStoreClosed
	}

	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now

	session.Metadata.MessageCount = len(session.Messages)

	sessionPath := filepath.Join(s.basePath, session.ID+".json")
	if err := writeAtomically(sessionPath, session); err != nil {
		return fmt.Errorf("write session: %w", err)
	}

	s.index.mu.Lock()
	defer s.index.mu.Unlock()

	meta := SessionMetadataIndex{
		ID:           session.ID,
		Title:        session.Title,
		MessageCount: session.Metadata.MessageCount,
		Model:        session.Metadata.Model,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}

	found := false
	for i, m := range s.index.sessions {
		if m.ID == session.ID {
			s.index.sessions[i] = meta
			found = true
			break
		}
	}
	if !found {
		s.index.sessions = append(s.index.sessions, meta)
	}
	s.index.dirty = true

	if len(s.index.sessions)%10 == 0 {
		return s.persistIndexLocked()
	}

	return nil
}

func (s *FileStore) Load(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, ErrStoreClosed
	}

	sessionPath := filepath.Join(s.basePath, id+".json")

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("read session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}

	return &session, nil
}

func (s *FileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStoreClosed
	}

	sessionPath := filepath.Join(s.basePath, id+".json")
	if err := os.Remove(sessionPath); err != nil {
		if os.IsNotExist(err) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("remove session: %w", err)
	}

	s.index.mu.Lock()
	defer s.index.mu.Unlock()

	for i, m := range s.index.sessions {
		if m.ID == id {
			s.index.sessions = append(s.index.sessions[:i], s.index.sessions[i+1:]...)
			s.index.dirty = true
			break
		}
	}

	return s.persistIndexLocked()
}

func (s *FileStore) List(opts ListOptions) (*ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, ErrStoreClosed
	}

	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 20
	}
	if opts.PageSize > 100 {
		opts.PageSize = 100
	}

	s.index.mu.RLock()
	sessions := make([]SessionMetadataIndex, len(s.index.sessions))
	copy(sessions, s.index.sessions)
	s.index.mu.RUnlock()

	sort.Slice(sessions, func(i, j int) bool {
		if opts.Reverse {
			if opts.SortBy == "created_at" {
				return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
			}
			return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
		}
		if opts.SortBy == "created_at" {
			return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
		}
		return sessions[i].UpdatedAt.Before(sessions[j].UpdatedAt)
	})

	total := len(sessions)
	totalPages := (total + opts.PageSize - 1) / opts.PageSize

	start := (opts.Page - 1) * opts.PageSize
	if start >= total {
		return &ListResult{
			Sessions:   []SessionMetadataIndex{},
			Total:      total,
			Page:       opts.Page,
			PageSize:   opts.PageSize,
			TotalPages: totalPages,
		}, nil
	}

	end := start + opts.PageSize
	if end > total {
		end = total
	}

	return &ListResult{
		Sessions:   sessions[start:end],
		Total:      total,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *FileStore) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return false
	}

	sessionPath := filepath.Join(s.basePath, id+".json")
	_, err := os.Stat(sessionPath)
	return err == nil
}

func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.index.mu.Lock()
	defer s.index.mu.Unlock()

	if err := s.persistIndexLocked(); err != nil {
		return err
	}

	s.closed = true
	return nil
}

func (s *FileStore) loadIndex() error {
	indexPath := filepath.Join(s.basePath, "index.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	s.index.mu.Lock()
	defer s.index.mu.Unlock()

	return json.Unmarshal(data, &s.index.sessions)
}

func (s *FileStore) persistIndexLocked() error {
	if !s.index.dirty {
		return nil
	}

	indexPath := filepath.Join(s.basePath, "index.json")
	if err := writeAtomically(indexPath, s.index.sessions); err != nil {
		return err
	}

	s.index.dirty = false
	return nil
}

func writeAtomically(path string, data interface{}) error {
	tmpPath := path + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "")

	if err := encoder.Encode(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, path)
}
