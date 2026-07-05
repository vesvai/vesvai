package session

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/llm"
)

func newTestStore(t *testing.T) (*FileStore, func()) {
	t.Helper()
	store, err := NewFileStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store, func() {
		store.Close()
	}
}

func newTestSession(id string) *Session {
	return &Session{
		ID:    id,
		Title: "Test Session",
		Messages: []llm.Message{
			llm.UserMessage("Hello"),
			llm.AssistantMessage("Hi there!"),
		},
		Metadata: SessionMetadata{
			Model:         "gpt-4",
			WorkspacePath: "/test/workspace",
		},
	}
}

func TestSaveAndLoad(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	session := newTestSession("test-1")
	if err := store.Save(session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	loaded, err := store.Load("test-1")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, loaded.ID)
	}
	if loaded.Title != session.Title {
		t.Errorf("expected Title %s, got %s", session.Title, loaded.Title)
	}
	if len(loaded.Messages) != len(session.Messages) {
		t.Errorf("expected %d messages, got %d", len(session.Messages), len(loaded.Messages))
	}
	if loaded.Metadata.Model != session.Metadata.Model {
		t.Errorf("expected Model %s, got %s", session.Metadata.Model, loaded.Metadata.Model)
	}
}

func TestSaveAutoGeneratesID(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	session := &Session{
		Messages: []llm.Message{
			llm.UserMessage("Test"),
		},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	if session.ID == "" {
		t.Error("expected auto-generated ID")
	}

	loaded, err := store.Load(session.ID)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, loaded.ID)
	}
}

func TestSaveUpdatesTimestamps(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	session := newTestSession("test-1")
	before := time.Now()
	if err := store.Save(session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}
	after := time.Now()

	if session.CreatedAt.Before(before) || session.CreatedAt.After(after) {
		t.Error("CreatedAt not set correctly")
	}
	if session.UpdatedAt.Before(before) || session.UpdatedAt.After(after) {
		t.Error("UpdatedAt not set correctly")
	}
}

func TestSaveUpdatesMetadataCount(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	session := newTestSession("test-1")
	if err := store.Save(session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	if session.Metadata.MessageCount != 2 {
		t.Errorf("expected MessageCount 2, got %d", session.Metadata.MessageCount)
	}
}

func TestLoadNotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	_, err := store.Load("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	session := newTestSession("test-1")
	if err := store.Save(session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	if err := store.Delete("test-1"); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	_, err := store.Load("test-1")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	err := store.Delete("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestExists(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	session := newTestSession("test-1")
	if err := store.Save(session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	if !store.Exists("test-1") {
		t.Error("expected Exists to return true for existing session")
	}

	if store.Exists("nonexistent") {
		t.Error("expected Exists to return false for nonexistent session")
	}
}

func TestListPagination(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Create 50 sessions
	for i := 0; i < 50; i++ {
		session := &Session{
			ID:    fmt.Sprintf("session-%03d", i),
			Title: fmt.Sprintf("Session %d", i),
			Messages: []llm.Message{
				llm.UserMessage("Message"),
			},
		}
		if err := store.Save(session); err != nil {
			t.Fatalf("failed to save session %d: %v", i, err)
		}
	}

	// Test first page
	result, err := store.List(ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if result.Total != 50 {
		t.Errorf("expected total 50, got %d", result.Total)
	}
	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("expected page size 10, got %d", result.PageSize)
	}
	if result.TotalPages != 5 {
		t.Errorf("expected total pages 5, got %d", result.TotalPages)
	}
	if len(result.Sessions) != 10 {
		t.Errorf("expected 10 sessions, got %d", len(result.Sessions))
	}

	// Test second page
	result2, err := store.List(ListOptions{Page: 2, PageSize: 10})
	if err != nil {
		t.Fatalf("failed to list sessions page 2: %v", err)
	}

	if len(result2.Sessions) != 10 {
		t.Errorf("expected 10 sessions on page 2, got %d", len(result2.Sessions))
	}

	// Test last page
	resultLast, err := store.List(ListOptions{Page: 5, PageSize: 10})
	if err != nil {
		t.Fatalf("failed to list sessions last page: %v", err)
	}

	if len(resultLast.Sessions) != 10 {
		t.Errorf("expected 10 sessions on last page, got %d", len(resultLast.Sessions))
	}

	// Test beyond last page
	resultBeyond, err := store.List(ListOptions{Page: 6, PageSize: 10})
	if err != nil {
		t.Fatalf("failed to list sessions beyond last page: %v", err)
	}

	if len(resultBeyond.Sessions) != 0 {
		t.Errorf("expected 0 sessions beyond last page, got %d", len(resultBeyond.Sessions))
	}
}

func TestListDefaults(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Test negative page
	result, err := store.List(ListOptions{Page: -1, PageSize: -1})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
	if result.PageSize != 20 {
		t.Errorf("expected page size 20, got %d", result.PageSize)
	}

	// Test page size > 100
	result2, err := store.List(ListOptions{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if result2.PageSize != 100 {
		t.Errorf("expected page size 100, got %d", result2.PageSize)
	}
}

func TestListSortingByCreatedAt(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Create sessions with different timestamps
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:    "session-" + string(rune('a'+i)),
			Title: "Session",
			Messages: []llm.Message{
				llm.UserMessage("Message"),
			},
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := store.Save(session); err != nil {
			t.Fatalf("failed to save session: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Test ascending order
	result, err := store.List(ListOptions{Page: 1, PageSize: 5, SortBy: "created_at", Reverse: false})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	for i := 1; i < len(result.Sessions); i++ {
		if result.Sessions[i].CreatedAt.Before(result.Sessions[i-1].CreatedAt) {
			t.Error("sessions not sorted by created_at ascending")
			break
		}
	}

	// Test descending order
	resultDesc, err := store.List(ListOptions{Page: 1, PageSize: 5, SortBy: "created_at", Reverse: true})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	for i := 1; i < len(resultDesc.Sessions); i++ {
		if resultDesc.Sessions[i].CreatedAt.After(resultDesc.Sessions[i-1].CreatedAt) {
			t.Error("sessions not sorted by created_at descending")
			break
		}
	}
}

func TestListSortingByUpdatedAt(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Create sessions with different timestamps
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:    "session-" + string(rune('a'+i)),
			Title: "Session",
			Messages: []llm.Message{
				llm.UserMessage("Message"),
			},
			UpdatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := store.Save(session); err != nil {
			t.Fatalf("failed to save session: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Test ascending order
	result, err := store.List(ListOptions{Page: 1, PageSize: 5, SortBy: "updated_at", Reverse: false})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	for i := 1; i < len(result.Sessions); i++ {
		if result.Sessions[i].UpdatedAt.Before(result.Sessions[i-1].UpdatedAt) {
			t.Error("sessions not sorted by updated_at ascending")
			break
		}
	}

	// Test descending order
	resultDesc, err := store.List(ListOptions{Page: 1, PageSize: 5, SortBy: "updated_at", Reverse: true})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	for i := 1; i < len(resultDesc.Sessions); i++ {
		if resultDesc.Sessions[i].UpdatedAt.After(resultDesc.Sessions[i-1].UpdatedAt) {
			t.Error("sessions not sorted by updated_at descending")
			break
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	var wg sync.WaitGroup
	const goroutines = 10
	const operations = 100

	// Concurrent saves
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				session := &Session{
					ID:    "session-" + string(rune('a'+g)),
					Title: "Session",
					Messages: []llm.Message{
						llm.UserMessage("Message"),
					},
				}
				if err := store.Save(session); err != nil {
					t.Errorf("failed to save session in goroutine %d: %v", g, err)
					return
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				store.List(ListOptions{Page: 1, PageSize: 10})
				store.Exists("session-a")
			}
		}(i)
	}

	wg.Wait()
}

func TestStoreClosedErrors(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	if err := store.Close(); err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Test Save on closed store
	err := store.Save(newTestSession("test"))
	if err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on Save, got %v", err)
	}

	// Test Load on closed store
	_, err = store.Load("test")
	if err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on Load, got %v", err)
	}

	// Test Delete on closed store
	err = store.Delete("test")
	if err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on Delete, got %v", err)
	}

	// Test List on closed store
	_, err = store.List(ListOptions{})
	if err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on List, got %v", err)
	}

	// Test Exists on closed store
	if store.Exists("test") {
		t.Error("expected Exists to return false on closed store")
	}
}

func TestCloseIdempotent(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	if err := store.Close(); err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Second close should not error
	if err := store.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestDeleteRemovesFromIndex(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:    "session-" + string(rune('a'+i)),
			Title: "Session",
			Messages: []llm.Message{
				llm.UserMessage("Message"),
			},
		}
		if err := store.Save(session); err != nil {
			t.Fatalf("failed to save session: %v", err)
		}
	}

	// Delete one
	if err := store.Delete("session-c"); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// List should have 4 sessions
	result, err := store.List(ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if result.Total != 4 {
		t.Errorf("expected 4 sessions after delete, got %d", result.Total)
	}

	// Verify deleted session not in list
	for _, s := range result.Sessions {
		if s.ID == "session-c" {
			t.Error("deleted session still in list")
		}
	}
}
