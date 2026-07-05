package memory

import (
	"os"
	"testing"
)

func TestMemory_AddAndGetFact(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)

	// Initialize memory
	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save initial memory: %v", err)
	}
	if err := mem.Load(); err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	// Add fact
	fact, err := mem.AddFact(FactTypeArchitecture, "Test Fact", "Test content", 0.9, []string{"test"}, "test")
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}

	// Get fact
	retrieved, err := mem.GetFact(fact.ID)
	if err != nil {
		t.Fatalf("Failed to get fact: %v", err)
	}

	if retrieved.Title != "Test Fact" {
		t.Errorf("Expected title 'Test Fact', got '%s'", retrieved.Title)
	}
	if retrieved.Content != "Test content" {
		t.Errorf("Expected content 'Test content', got '%s'", retrieved.Content)
	}
	if retrieved.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", retrieved.Confidence)
	}
}

func TestMemory_UpdateFact(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)

	// Initialize memory
	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save initial memory: %v", err)
	}
	if err := mem.Load(); err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	// Add fact
	fact, err := mem.AddFact(FactTypeArchitecture, "Original Title", "Original content", 0.8, nil, "")
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}

	// Update fact
	updated, err := mem.UpdateFact(fact.ID, map[string]interface{}{
		"title":      "Updated Title",
		"content":    "Updated content",
		"confidence": 0.95,
	})
	if err != nil {
		t.Fatalf("Failed to update fact: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", updated.Title)
	}
	if updated.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got '%s'", updated.Content)
	}
	if updated.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", updated.Confidence)
	}
}

func TestMemory_DeleteFact(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)

	// Initialize memory
	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save initial memory: %v", err)
	}
	if err := mem.Load(); err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	// Add fact
	fact, err := mem.AddFact(FactTypeArchitecture, "To Delete", "Delete me", 0.5, nil, "")
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}

	// Delete fact
	if err := mem.DeleteFact(fact.ID); err != nil {
		t.Fatalf("Failed to delete fact: %v", err)
	}

	// Try to get deleted fact
	if _, err := mem.GetFact(fact.ID); err != ErrFactNotFound {
		t.Errorf("Expected ErrFactNotFound, got %v", err)
	}
}

func TestMemory_SearchFacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)

	// Initialize memory
	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save initial memory: %v", err)
	}
	if err := mem.Load(); err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	// Add multiple facts
	mem.AddFact(FactTypeArchitecture, "Go Module", "Go project", 0.9, []string{"go"}, "")
	mem.AddFact(FactTypeConvention, "Code Style", "Uses golangci-lint", 0.8, []string{"lint"}, "")
	mem.AddFact(FactTypeArchitecture, "REST API", "HTTP endpoints", 0.85, []string{"api"}, "")

	results, err := mem.SearchFacts("Module", nil)
	if err != nil {
		t.Fatalf("Failed to search facts: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Search by type
	results, err = mem.SearchFacts("", &SearchOptions{Type: FactTypeArchitecture})
	if err != nil {
		t.Fatalf("Failed to search facts: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Search by tags
	results, err = mem.SearchFacts("", &SearchOptions{Tags: []string{"go"}})
	if err != nil {
		t.Fatalf("Failed to search facts: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestMemory_MergeFacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)

	// Initialize memory
	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save initial memory: %v", err)
	}
	if err := mem.Load(); err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	// Add initial fact
	mem.AddFact(FactTypeArchitecture, "Go Module", "Go project", 0.7, []string{"go"}, "")

	// Merge with higher confidence
	merged, err := mem.MergeFacts(Fact{
		Type:       FactTypeArchitecture,
		Title:      "Go Module",
		Content:    "Go module project with multiple packages",
		Confidence: 0.9,
		Tags:       []string{"go", "modules"},
	})
	if err != nil {
		t.Fatalf("Failed to merge facts: %v", err)
	}

	if merged.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", merged.Confidence)
	}
	if merged.Content != "Go module project with multiple packages" {
		t.Errorf("Expected updated content, got '%s'", merged.Content)
	}

	// Verify only one fact exists
	facts, _ := mem.GetFactsByType(FactTypeArchitecture)
	if len(facts) != 1 {
		t.Errorf("Expected 1 fact, got %d", len(facts))
	}
}

func TestMemory_AddAndGetNote(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)

	// Initialize memory
	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save initial memory: %v", err)
	}
	if err := mem.Load(); err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	// Add note
	note, err := mem.AddNote("Test Note", "Test note content", []string{"test"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	// Get note
	retrieved, err := mem.GetNote(note.ID)
	if err != nil {
		t.Fatalf("Failed to get note: %v", err)
	}

	if retrieved.Title != "Test Note" {
		t.Errorf("Expected title 'Test Note', got '%s'", retrieved.Title)
	}
	if retrieved.Content != "Test note content" {
		t.Errorf("Expected content 'Test note content', got '%s'", retrieved.Content)
	}
}

func TestMemory_GetStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)

	// Initialize memory
	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save initial memory: %v", err)
	}
	if err := mem.Load(); err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	// Add some facts
	mem.AddFact(FactTypeArchitecture, "Fact 1", "Content 1", 0.9, nil, "")
	mem.AddFact(FactTypeConvention, "Fact 2", "Content 2", 0.8, nil, "")
	mem.AddNote("Note 1", "Note content", nil)

	// Get stats
	stats := mem.GetStats()
	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	totalFacts, ok := stats["total_facts"].(int)
	if !ok || totalFacts != 2 {
		t.Errorf("Expected 2 total facts, got %v", stats["total_facts"])
	}

	totalNotes, ok := stats["total_notes"].(int)
	if !ok || totalNotes != 1 {
		t.Errorf("Expected 1 total note, got %v", stats["total_notes"])
	}
}

func TestMemory_PersistenceAcrossLoads(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// First instance
	mem1 := New(store)
	memory := &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}
	if err := store.Save(memory); err != nil {
		t.Fatalf("Failed to save initial memory: %v", err)
	}
	if err := mem1.Load(); err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	// Add facts
	mem1.AddFact(FactTypeArchitecture, "Persistent Fact", "Should persist", 0.9, nil, "")
	mem1.AddNote("Persistent Note", "Should persist", nil)

	// Save
	if err := mem1.Save(); err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}

	// Second instance
	mem2 := New(store)
	if err := mem2.Load(); err != nil {
		t.Fatalf("Failed to load memory in second instance: %v", err)
	}

	// Verify persistence
	facts, _ := mem2.GetFactsByType(FactTypeArchitecture)
	if len(facts) != 1 {
		t.Errorf("Expected 1 fact in second instance, got %d", len(facts))
	}

	notes := mem2.memory.Notes
	if len(notes) != 1 {
		t.Errorf("Expected 1 note in second instance, got %d", len(notes))
	}
}
