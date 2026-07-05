package session

import (
	"fmt"
	"testing"

	"github.com/vesvai/vesvai/internal/llm"
)

func setupBenchmarkStore(b *testing.B, sessionCount int) *FileStore {
	b.Helper()
	store, err := NewFileStore()
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	for i := 0; i < sessionCount; i++ {
		session := &Session{
			ID:    fmt.Sprintf("session-%d", i),
			Title: fmt.Sprintf("Session %d", i),
			Messages: []llm.Message{
				llm.UserMessage(fmt.Sprintf("Message %d", i)),
				llm.AssistantMessage(fmt.Sprintf("Response %d", i)),
			},
			Metadata: SessionMetadata{
				Model:         "gpt-4",
				WorkspacePath: "/test/workspace",
			},
		}
		if err := store.Save(session); err != nil {
			b.Fatalf("failed to save session: %v", err)
		}
	}

	return store
}

func BenchmarkSave(b *testing.B) {
	store := setupBenchmarkStore(b, 0)
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := &Session{
			ID:    fmt.Sprintf("bench-session-%d", i),
			Title: "Benchmark Session",
			Messages: []llm.Message{
				llm.UserMessage("Hello"),
				llm.AssistantMessage("Hi there!"),
			},
			Metadata: SessionMetadata{
				Model:         "gpt-4",
				WorkspacePath: "/test/workspace",
			},
		}
		if err := store.Save(session); err != nil {
			b.Fatalf("failed to save session: %v", err)
		}
	}
}

func BenchmarkLoad(b *testing.B) {
	store := setupBenchmarkStore(b, 1000)
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("session-%d", i%1000)
		if _, err := store.Load(id); err != nil {
			b.Fatalf("failed to load session: %v", err)
		}
	}
}

func BenchmarkListPage1(b *testing.B) {
	store := setupBenchmarkStore(b, 1000)
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.List(ListOptions{Page: 1, PageSize: 20}); err != nil {
			b.Fatalf("failed to list sessions: %v", err)
		}
	}
}

func BenchmarkListPage10(b *testing.B) {
	store := setupBenchmarkStore(b, 1000)
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.List(ListOptions{Page: 10, PageSize: 20}); err != nil {
			b.Fatalf("failed to list sessions: %v", err)
		}
	}
}

func BenchmarkListPage100(b *testing.B) {
	store := setupBenchmarkStore(b, 1000)
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.List(ListOptions{Page: 100, PageSize: 20}); err != nil {
			b.Fatalf("failed to list sessions: %v", err)
		}
	}
}

func BenchmarkConcurrentLoad(b *testing.B) {
	store := setupBenchmarkStore(b, 1000)
	defer store.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := fmt.Sprintf("session-%d", i%1000)
			if _, err := store.Load(id); err != nil {
				b.Fatalf("failed to load session: %v", err)
			}
			i++
		}
	})
}

func BenchmarkEncodeSession(b *testing.B) {
	session := &Session{
		ID:    "test-session",
		Title: "Test Session",
		Messages: []llm.Message{
			llm.UserMessage("Hello"),
			llm.AssistantMessage("Hi there!"),
			llm.UserMessage("How are you?"),
			llm.AssistantMessage("I'm doing well, thanks for asking!"),
		},
		Metadata: SessionMetadata{
			Model:         "gpt-4",
			MessageCount:  4,
			WorkspacePath: "/test/workspace",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := EncodeSession(session); err != nil {
			b.Fatalf("failed to encode session: %v", err)
		}
	}
}

func BenchmarkDecodeSession(b *testing.B) {
	session := &Session{
		ID:    "test-session",
		Title: "Test Session",
		Messages: []llm.Message{
			llm.UserMessage("Hello"),
			llm.AssistantMessage("Hi there!"),
			llm.UserMessage("How are you?"),
			llm.AssistantMessage("I'm doing well, thanks for asking!"),
		},
		Metadata: SessionMetadata{
			Model:         "gpt-4",
			MessageCount:  4,
			WorkspacePath: "/test/workspace",
		},
	}

	data, err := EncodeSession(session)
	if err != nil {
		b.Fatalf("failed to encode session: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DecodeSession(data); err != nil {
			b.Fatalf("failed to decode session: %v", err)
		}
	}
}
