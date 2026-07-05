package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
)

func NewMemoryTools(mem *Memory) []agent.Tool {
	return []agent.Tool{
		newInitializeMemoryTool(mem),
		newListFactsTool(mem),
		newGetFactTool(mem),
		newAddFactTool(mem),
		newUpdateFactTool(mem),
		newDeleteFactTool(mem),
		newSearchFactsTool(mem),
		newAddNoteTool(mem),
		newGetNoteTool(mem),
		newUpdateNoteTool(mem),
		newDeleteNoteTool(mem),
		newGetStatsTool(mem),
		newMergeFactTool(mem),
	}
}

func newInitializeMemoryTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"initialize-memory",
		"Initialize workspace memory by analyzing the codebase. This extracts architecture, conventions, patterns, and dependencies from the project. Only runs once - subsequent calls load existing memory.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspace_path": map[string]any{
					"type":        "string",
					"description": "Path to the workspace root directory to analyze",
				},
			},
			"required": []string{"workspace_path"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			workspacePath, _ := params["workspace_path"].(string)
			if workspacePath == "" {
				return "", fmt.Errorf("workspace_path is required")
			}

			if err := mem.Initialize(workspacePath); err != nil {
				return "", fmt.Errorf("failed to initialize memory: %w", err)
			}

			stats := mem.GetStats()
			totalFacts := stats["total_facts"].(int)
			totalNotes := stats["total_notes"].(int)

			return fmt.Sprintf("Memory initialized successfully.\nFacts: %d\nNotes: %d\n", totalFacts, totalNotes), nil
		},
	)
}

func newListFactsTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"list-facts",
		"List all facts in memory. Optionally filter by type. Returns paginated results with fact details.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{
					"type":        "string",
					"description": "Filter by fact type",
					"enum": []string{
						"architecture", "convention", "decision", "warning",
						"module", "relationship", "dependency", "pattern",
					},
				},
				"page": map[string]any{
					"type":        "integer",
					"description": "Page number (default: 1)",
				},
				"page_size": map[string]any{
					"type":        "integer",
					"description": "Items per page (default: 50)",
				},
			},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			opts := &ListOptions{}

			if typeStr, ok := params["type"].(string); ok && typeStr != "" {
				opts.Type = FactType(typeStr)
			}
			if page, ok := params["page"].(float64); ok && page > 0 {
				opts.Page = int(page)
			}
			if pageSize, ok := params["page_size"].(float64); ok && pageSize > 0 {
				opts.PageSize = int(pageSize)
			}

			result, err := mem.ListFacts(opts)
			if err != nil {
				return "", fmt.Errorf("failed to list facts: %w", err)
			}

			if len(result.Facts) == 0 {
				return "No facts found.\n", nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Facts (page %d/%d, %d total):\n\n", result.Page, result.TotalPages, result.Total))

			for _, fact := range result.Facts {
				sb.WriteString(fmt.Sprintf("[%s] %s\n", fact.Type, fact.Title))
				sb.WriteString(fmt.Sprintf("  ID: %s\n", fact.ID))
				sb.WriteString(fmt.Sprintf("  Confidence: %.2f\n", fact.Confidence))
				if len(fact.Tags) > 0 {
					sb.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(fact.Tags, ", ")))
				}
				sb.WriteString("\n")
			}

			return sb.String(), nil
		},
	)
}

func newGetFactTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"get-fact",
		"Retrieve a specific fact by its ID. Returns full fact details including type, title, content, confidence, and tags.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The fact ID to retrieve",
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("fact id is required")
			}

			fact, err := mem.GetFact(id)
			if err != nil {
				return "", err
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Fact: %s\n", fact.Title))
			sb.WriteString(fmt.Sprintf("ID: %s\n", fact.ID))
			sb.WriteString(fmt.Sprintf("Type: %s\n", fact.Type))
			sb.WriteString(fmt.Sprintf("Confidence: %.2f\n", fact.Confidence))
			if len(fact.Tags) > 0 {
				sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(fact.Tags, ", ")))
			}
			if fact.Source != "" {
				sb.WriteString(fmt.Sprintf("Source: %s\n", fact.Source))
			}
			sb.WriteString(fmt.Sprintf("Created: %s\n", fact.CreatedAt.Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("Updated: %s\n\n", fact.UpdatedAt.Format("2006-01-02 15:04:05")))
			sb.WriteString("---\n\n")
			sb.WriteString(fact.Content)

			return sb.String(), nil
		},
	)
}

func newAddFactTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"add-fact",
		"Add a new fact to workspace memory. Facts represent knowledge about the codebase architecture, conventions, patterns, or decisions.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{
					"type":        "string",
					"description": "The fact type category",
					"enum": []string{
						"architecture", "convention", "decision", "warning",
						"module", "relationship", "dependency", "pattern",
					},
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Short descriptive title for the fact",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Detailed content describing the fact",
				},
				"confidence": map[string]any{
					"type":        "number",
					"description": "Confidence score from 0.0 to 1.0 (default: 0.8)",
				},
				"tags": map[string]any{
					"type":        "array",
					"description": "Optional tags for categorization",
					"items": map[string]any{
						"type": "string",
					},
				},
				"source": map[string]any{
					"type":        "string",
					"description": "Source file or description of where this fact was learned",
				},
			},
			"required": []string{"type", "title", "content"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			typeStr, _ := params["type"].(string)
			title, _ := params["title"].(string)
			content, _ := params["content"].(string)

			if typeStr == "" || title == "" || content == "" {
				return "", fmt.Errorf("type, title, and content are required")
			}

			factType := FactType(typeStr)
			confidence := 0.8
			if c, ok := params["confidence"].(float64); ok {
				confidence = c
			}

			var tags []string
			if t, ok := params["tags"].([]interface{}); ok {
				for _, tag := range t {
					if s, ok := tag.(string); ok {
						tags = append(tags, s)
					}
				}
			}

			source, _ := params["source"].(string)

			fact, err := mem.AddFact(factType, title, content, confidence, tags, source)
			if err != nil {
				return "", fmt.Errorf("failed to add fact: %w", err)
			}

			return fmt.Sprintf("Fact added successfully.\nID: %s\nTitle: %s\nType: %s\n", fact.ID, fact.Title, fact.Type), nil
		},
	)
}

func newUpdateFactTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"update-fact",
		"Update an existing fact. Only specified fields will be updated.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The fact ID to update",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "New title (optional)",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "New content (optional)",
				},
				"confidence": map[string]any{
					"type":        "number",
					"description": "New confidence score (optional)",
				},
				"tags": map[string]any{
					"type":        "array",
					"description": "New tags (optional)",
					"items": map[string]any{
						"type": "string",
					},
				},
				"source": map[string]any{
					"type":        "string",
					"description": "New source (optional)",
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("fact id is required")
			}

			updates := make(map[string]interface{})
			if title, ok := params["title"].(string); ok && title != "" {
				updates["title"] = title
			}
			if content, ok := params["content"].(string); ok && content != "" {
				updates["content"] = content
			}
			if confidence, ok := params["confidence"].(float64); ok {
				updates["confidence"] = confidence
			}
			if t, ok := params["tags"].([]interface{}); ok {
				tags := make([]string, 0)
				for _, tag := range t {
					if s, ok := tag.(string); ok {
						tags = append(tags, s)
					}
				}
				updates["tags"] = tags
			}
			if source, ok := params["source"].(string); ok {
				updates["source"] = source
			}

			if len(updates) == 0 {
				return "", fmt.Errorf("at least one field to update is required")
			}

			fact, err := mem.UpdateFact(id, updates)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Fact updated successfully.\nID: %s\nTitle: %s\n", fact.ID, fact.Title), nil
		},
	)
}

func newDeleteFactTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"delete-fact",
		"Delete a fact from memory by its ID.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The fact ID to delete",
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("fact id is required")
			}

			if err := mem.DeleteFact(id); err != nil {
				return "", err
			}

			return fmt.Sprintf("Fact deleted successfully.\nID: %s\n", id), nil
		},
	)
}

func newSearchFactsTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"search-facts",
		"Search facts by query string, type, tags, or confidence threshold. Returns matching facts.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query to match against title and content",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Filter by fact type",
					"enum": []string{
						"architecture", "convention", "decision", "warning",
						"module", "relationship", "dependency", "pattern",
					},
				},
				"min_confidence": map[string]any{
					"type":        "number",
					"description": "Minimum confidence threshold (0.0-1.0)",
				},
				"tags": map[string]any{
					"type":        "array",
					"description": "Filter by tags (AND logic)",
					"items": map[string]any{
						"type": "string",
					},
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results",
				},
			},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			query, _ := params["query"].(string)

			opts := &SearchOptions{}

			if typeStr, ok := params["type"].(string); ok && typeStr != "" {
				opts.Type = FactType(typeStr)
			}
			if minConf, ok := params["min_confidence"].(float64); ok {
				opts.MinConfidence = minConf
			}
			if t, ok := params["tags"].([]interface{}); ok {
				tags := make([]string, 0)
				for _, tag := range t {
					if s, ok := tag.(string); ok {
						tags = append(tags, s)
					}
				}
				opts.Tags = tags
			}
			if limit, ok := params["limit"].(float64); ok && limit > 0 {
				opts.Limit = int(limit)
			}

			results, err := mem.SearchFacts(query, opts)
			if err != nil {
				return "", fmt.Errorf("failed to search facts: %w", err)
			}

			if len(results) == 0 {
				return "No facts found matching your search.\n", nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Found %d facts:\n\n", len(results)))

			for _, fact := range results {
				sb.WriteString(fmt.Sprintf("[%s] %s\n", fact.Type, fact.Title))
				sb.WriteString(fmt.Sprintf("  ID: %s\n", fact.ID))
				sb.WriteString(fmt.Sprintf("  Confidence: %.2f\n", fact.Confidence))
				if len(fact.Tags) > 0 {
					sb.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(fact.Tags, ", ")))
				}
				sb.WriteString("\n")
			}

			return sb.String(), nil
		},
	)
}

func newAddNoteTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"add-note",
		"Add a note to workspace memory. Notes are for observations, context, or temporary information.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "Note title",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Note content",
				},
				"tags": map[string]any{
					"type":        "array",
					"description": "Optional tags for categorization",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
			"required": []string{"title", "content"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			title, _ := params["title"].(string)
			content, _ := params["content"].(string)

			if title == "" || content == "" {
				return "", fmt.Errorf("title and content are required")
			}

			var tags []string
			if t, ok := params["tags"].([]interface{}); ok {
				for _, tag := range t {
					if s, ok := tag.(string); ok {
						tags = append(tags, s)
					}
				}
			}

			note, err := mem.AddNote(title, content, tags)
			if err != nil {
				return "", fmt.Errorf("failed to add note: %w", err)
			}

			return fmt.Sprintf("Note added successfully.\nID: %s\nTitle: %s\n", note.ID, note.Title), nil
		},
	)
}

func newGetNoteTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"get-note",
		"Retrieve a specific note by its ID.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The note ID to retrieve",
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("note id is required")
			}

			note, err := mem.GetNote(id)
			if err != nil {
				return "", err
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Note: %s\n", note.Title))
			sb.WriteString(fmt.Sprintf("ID: %s\n", note.ID))
			if len(note.Tags) > 0 {
				sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(note.Tags, ", ")))
			}
			sb.WriteString(fmt.Sprintf("Created: %s\n", note.CreatedAt.Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("Updated: %s\n\n", note.UpdatedAt.Format("2006-01-02 15:04:05")))
			sb.WriteString("---\n\n")
			sb.WriteString(note.Content)

			return sb.String(), nil
		},
	)
}

func newUpdateNoteTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"update-note",
		"Update an existing note. Only specified fields will be updated.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The note ID to update",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "New title (optional)",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "New content (optional)",
				},
				"tags": map[string]any{
					"type":        "array",
					"description": "New tags (optional)",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("note id is required")
			}

			updates := make(map[string]interface{})
			if title, ok := params["title"].(string); ok && title != "" {
				updates["title"] = title
			}
			if content, ok := params["content"].(string); ok && content != "" {
				updates["content"] = content
			}
			if t, ok := params["tags"].([]interface{}); ok {
				tags := make([]string, 0)
				for _, tag := range t {
					if s, ok := tag.(string); ok {
						tags = append(tags, s)
					}
				}
				updates["tags"] = tags
			}

			if len(updates) == 0 {
				return "", fmt.Errorf("at least one field to update is required")
			}

			note, err := mem.UpdateNote(id, updates)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Note updated successfully.\nID: %s\nTitle: %s\n", note.ID, note.Title), nil
		},
	)
}

func newDeleteNoteTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"delete-note",
		"Delete a note from memory by its ID.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The note ID to delete",
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("note id is required")
			}

			if err := mem.DeleteNote(id); err != nil {
				return "", err
			}

			return fmt.Sprintf("Note deleted successfully.\nID: %s\n", id), nil
		},
	)
}

func newGetStatsTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"get-stats",
		"Get statistics about workspace memory including fact counts, note counts, and breakdown by type.",
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			stats := mem.GetStats()
			if stats == nil {
				return "Memory not initialized.\n", nil
			}

			var sb strings.Builder
			sb.WriteString("Workspace Memory Statistics:\n\n")
			sb.WriteString(fmt.Sprintf("Total Facts: %d\n", stats["total_facts"]))
			sb.WriteString(fmt.Sprintf("Total Notes: %d\n", stats["total_notes"]))
			sb.WriteString(fmt.Sprintf("Version: %d\n", stats["version"]))

			if avgConf, ok := stats["average_confidence"].(float64); ok {
				sb.WriteString(fmt.Sprintf("Average Confidence: %.2f\n", avgConf))
			}

			if factsByType, ok := stats["facts_by_type"].(map[FactType]int); ok && len(factsByType) > 0 {
				sb.WriteString("\nFacts by Type:\n")
				for factType, count := range factsByType {
					sb.WriteString(fmt.Sprintf("  %s: %d\n", factType, count))
				}
			}

			return sb.String(), nil
		},
	)
}

func newMergeFactTool(mem *Memory) agent.Tool {
	return agent.NewFuncTool(
		"merge-fact",
		"Merge a fact with existing facts. If a fact with the same title and type exists, it will be updated with higher confidence. Otherwise, a new fact is created.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{
					"type":        "string",
					"description": "The fact type category",
					"enum": []string{
						"architecture", "convention", "decision", "warning",
						"module", "relationship", "dependency", "pattern",
					},
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Short descriptive title for the fact",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Detailed content describing the fact",
				},
				"confidence": map[string]any{
					"type":        "number",
					"description": "Confidence score from 0.0 to 1.0",
				},
				"tags": map[string]any{
					"type":        "array",
					"description": "Optional tags for categorization",
					"items": map[string]any{
						"type": "string",
					},
				},
				"source": map[string]any{
					"type":        "string",
					"description": "Source file or description",
				},
			},
			"required": []string{"type", "title", "content", "confidence"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			typeStr, _ := params["type"].(string)
			title, _ := params["title"].(string)
			content, _ := params["content"].(string)
			confidence, _ := params["confidence"].(float64)

			if typeStr == "" || title == "" || content == "" {
				return "", fmt.Errorf("type, title, and content are required")
			}

			var tags []string
			if t, ok := params["tags"].([]interface{}); ok {
				for _, tag := range t {
					if s, ok := tag.(string); ok {
						tags = append(tags, s)
					}
				}
			}

			source, _ := params["source"].(string)

			fact, err := mem.MergeFacts(Fact{
				Type:       FactType(typeStr),
				Title:      title,
				Content:    content,
				Confidence: confidence,
				Tags:       tags,
				Source:     source,
			})
			if err != nil {
				return "", fmt.Errorf("failed to merge fact: %w", err)
			}

			return fmt.Sprintf("Fact merged successfully.\nID: %s\nTitle: %s\nConfidence: %.2f\n", fact.ID, fact.Title, fact.Confidence), nil
		},
	)
}
