package memory

import (
	"context"
	"os"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
)

func TestMemoryTools_InitializeMemory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)
	tools := NewMemoryTools(mem)

	var initTool agent.Tool
	for _, tool := range tools {
		if tool.Name() == "initialize-memory" {
			initTool = tool
			break
		}
	}

	if initTool == nil {
		t.Fatal("initialize-memory tool not found")
	}

	result, err := initTool.Handle(context.Background(), map[string]any{
		"workspace_path": "/home/obuntu/Projects/vesvai",
	})
	if err != nil {
		t.Fatalf("Failed to initialize memory: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	t.Logf("Initialize result:\n%s", result)
}

func TestMemoryTools_AddAndGetFact(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)
	tools := NewMemoryTools(mem)

	mem.memory = &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}

	var addFactTool agent.Tool
	for _, tool := range tools {
		if tool.Name() == "add-fact" {
			addFactTool = tool
			break
		}
	}

	if addFactTool == nil {
		t.Fatal("add-fact tool not found")
	}

	result, err := addFactTool.Handle(context.Background(), map[string]any{
		"type":       "architecture",
		"title":      "Test Architecture",
		"content":    "This is a test architecture fact",
		"confidence": 0.9,
		"tags":       []interface{}{"test", "architecture"},
		"source":     "test",
	})
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	t.Logf("Add fact result:\n%s", result)

	var factID string
	for _, tool := range tools {
		if tool.Name() == "list-facts" {
			listResult, err := tool.Handle(context.Background(), map[string]any{})
			if err != nil {
				t.Fatalf("Failed to list facts: %v", err)
			}
			t.Logf("List facts result:\n%s", listResult)

			lines := splitLines(listResult)
			for _, line := range lines {
				if len(line) > 6 && line[:6] == "  ID: " {
					factID = line[6:]
					break
				}
			}
			break
		}
	}

	if factID == "" {
		t.Fatal("Could not extract fact ID from list")
	}

	var getFactTool agent.Tool
	for _, tool := range tools {
		if tool.Name() == "get-fact" {
			getFactTool = tool
			break
		}
	}

	getResult, err := getFactTool.Handle(context.Background(), map[string]any{
		"id": factID,
	})
	if err != nil {
		t.Fatalf("Failed to get fact: %v", err)
	}

	t.Logf("Get fact result:\n%s", getResult)
}

func TestMemoryTools_SearchFacts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)
	tools := NewMemoryTools(mem)

	mem.memory = &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}

	mem.AddFact(FactTypeArchitecture, "Go Module", "Go project", 0.9, []string{"go"}, "")
	mem.AddFact(FactTypeConvention, "Code Style", "Uses golangci-lint", 0.8, []string{"lint"}, "")

	var searchTool agent.Tool
	for _, tool := range tools {
		if tool.Name() == "search-facts" {
			searchTool = tool
			break
		}
	}

	if searchTool == nil {
		t.Fatal("search-facts tool not found")
	}

	result, err := searchTool.Handle(context.Background(), map[string]any{
		"query": "Module",
	})
	if err != nil {
		t.Fatalf("Failed to search facts: %v", err)
	}

	t.Logf("Search result:\n%s", result)
}

func TestMemoryTools_GetStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStoreWithPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mem := New(store)
	tools := NewMemoryTools(mem)

	mem.memory = &WorkspaceMemory{
		Facts: make([]Fact, 0),
		Notes: make([]Note, 0),
	}

	mem.AddFact(FactTypeArchitecture, "Fact 1", "Content 1", 0.9, nil, "")
	mem.AddNote("Note 1", "Note content", nil)

	var statsTool agent.Tool
	for _, tool := range tools {
		if tool.Name() == "get-stats" {
			statsTool = tool
			break
		}
	}

	if statsTool == nil {
		t.Fatal("get-stats tool not found")
	}

	result, err := statsTool.Handle(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	t.Logf("Stats result:\n%s", result)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
