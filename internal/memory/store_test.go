package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	memory := &WorkspaceMemory{
		Facts: []Fact{
			{
				ID:         "test-fact-1",
				Type:       FactTypeArchitecture,
				Title:      "Test Architecture",
				Content:    "This is a test architecture fact",
				Confidence: 0.9,
				Tags:       []string{"test", "architecture"},
				Source:     "test",
			},
		},
		Notes: []Note{
			{
				ID:      "test-note-1",
				Title:   "Test Note",
				Content: "This is a test note",
				Tags:    []string{"test"},
			},
		},
	}

	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	if len(loaded.Facts) != 1 {
		t.Errorf("Expected 1 fact, got %d", len(loaded.Facts))
	}
	if len(loaded.Notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(loaded.Notes))
	}
	if loaded.Facts[0].Title != "Test Architecture" {
		t.Errorf("Expected title 'Test Architecture', got '%s'", loaded.Facts[0].Title)
	}
	if loaded.Notes[0].Title != "Test Note" {
		t.Errorf("Expected title 'Test Note', got '%s'", loaded.Notes[0].Title)
	}
}

func TestFileStore_Exists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	if store.Exists() {
		t.Error("Expected store to not exist initially")
	}

	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}

	if !store.Exists() {
		t.Error("Expected store to exist after save")
	}
}

func TestFileStore_LoadNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	memory, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load non-existent memory: %v", err)
	}

	if len(memory.Facts) != 0 {
		t.Errorf("Expected 0 facts, got %d", len(memory.Facts))
	}
	if len(memory.Notes) != 0 {
		t.Errorf("Expected 0 notes, got %d", len(memory.Notes))
	}
}

func TestFileStore_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed, got %v", err)
	}

	if _, err := store.Load(); err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed, got %v", err)
	}
}

func TestFileStore_AtomicWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.json")
	testData := map[string]string{"key": "value"}

	if err := writeAtomically(filePath, testData); err != nil {
		t.Fatalf("Failed to write atomically: %v", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected file to exist after atomic write")
	}

	tmpFilePath := filePath + ".tmp"
	if _, err := os.Stat(tmpFilePath); !os.IsNotExist(err) {
		t.Error("Expected tmp file to be removed after atomic write")
	}
}

func TestFileStore_MultipleSaves(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	for i := 0; i < 5; i++ {
		memory, err := store.Load()
		if err != nil {
			t.Fatalf("Failed to load memory on iteration %d: %v", i, err)
		}
		memory.Facts = []Fact{
			{
				ID:    "fact-" + string(rune('A'+i)),
				Title: "Fact " + string(rune('A'+i)),
			},
		}
		if err := store.Save(memory); err != nil {
			t.Fatalf("Failed to save memory on iteration %d: %v", i, err)
		}
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	if len(loaded.Facts) != 1 {
		t.Errorf("Expected 1 fact, got %d", len(loaded.Facts))
	}

	if loaded.Version != 5 {
		t.Errorf("Expected version 5, got %d", loaded.Version)
	}
}
