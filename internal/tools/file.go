package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/filesystem"
)

func newReadTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"read",
		"Read a file from the filesystem. Returns content with line numbers for easy reference. Use this to examine source code, configs, logs, or any text file. Supports pagination via offset/limit for large files. Always use absolute paths. The tool returns line numbers so you can reference specific lines in follow-up edits.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "Absolute path to the file to read. Supports ~ for home directory.",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "Line number to start reading from (1-indexed). Use this to skip to a specific section. Defaults to 1.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of lines to return. Defaults to 2000. Use smaller values for quick checks, larger for full file reads.",
				},
			},
			"required": []string{"filePath"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			filePath := asString(params, "filePath")
			if filePath == "" {
				return "", fmt.Errorf("filePath is required")
			}

			relPath, err := absoluteToRelative(fs, filePath)
			if err != nil {
				return "", err
			}

			offset := asInt(params, "offset")
			if offset < 1 {
				offset = 1
			}
			limit := asInt(params, "limit")
			if limit <= 0 {
				limit = 2000
			}

			content, err := fs.Read(ctx, relPath)
			if err != nil {
				return "", fmt.Errorf("failed to read file: %w", err)
			}

			lines := strings.Split(content, "\n")
			totalLines := len(lines)

			if offset > totalLines {
				return "", fmt.Errorf("offset %d exceeds file length %d", offset, totalLines)
			}

			end := offset + limit - 1
			if end > totalLines {
				end = totalLines
			}

			var sb strings.Builder
			for i := offset - 1; i < end; i++ {
				sb.WriteString(fmt.Sprintf("%d: %s\n", i+1, lines[i]))
			}

			if end < totalLines {
				sb.WriteString(fmt.Sprintf("\n(Showing lines %d-%d of %d total lines. Use offset=%d to continue reading.)\n",
					offset, end, totalLines, end+1))
			} else if offset > 1 {
				sb.WriteString(fmt.Sprintf("\n(Showing lines %d-%d of %d total lines.)\n",
					offset, end, totalLines))
			} else if totalLines > 0 {
				sb.WriteString(fmt.Sprintf("\n(%d lines total)\n", totalLines))
			}

			return sb.String(), nil
		},
	)
}

func newEditTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"edit",
		"Perform exact string replacement in a file. Use this to modify existing code or text. The oldString must match exactly including whitespace and indentation. Always read the file first to get the exact text. Provide enough surrounding context in oldString to make the match unique. If multiple matches exist, the tool will error unless replaceAll is true. After editing, the file always ends with a newline.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "Absolute path to the file to edit",
				},
				"oldString": map[string]any{
					"type":        "string",
					"description": "The exact text to find and replace. Must match exactly including indentation. Include enough context to make it unique.",
				},
				"newString": map[string]any{
					"type":        "string",
					"description": "The replacement text. Must be different from oldString.",
				},
				"replaceAll": map[string]any{
					"type":        "boolean",
					"description": "Replace all occurrences instead of just the first. Default false. Use when you want to rename a symbol across a file.",
				},
			},
			"required": []string{"filePath", "oldString", "newString"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			filePath := asString(params, "filePath")
			if filePath == "" {
				return "", fmt.Errorf("filePath is required")
			}
			oldString := asString(params, "oldString")
			if oldString == "" {
				return "", fmt.Errorf("oldString is required")
			}
			newString := asString(params, "newString")

			relPath, err := absoluteToRelative(fs, filePath)
			if err != nil {
				return "", err
			}

			content, err := fs.Read(ctx, relPath)
			if err != nil {
				return "", fmt.Errorf("failed to read file: %w", err)
			}

			if !strings.Contains(content, oldString) {
				return "", fmt.Errorf("oldString not found in file. Read the file first to get the exact text.")
			}

			if oldString == newString {
				return "", fmt.Errorf("oldString and newString are identical")
			}

			replaceAll := asBool(params, "replaceAll", false)

			var newContent string
			var replacements int
			if replaceAll {
				newContent = strings.ReplaceAll(content, oldString, newString)
				replacements = strings.Count(content, oldString)
			} else {
				count := strings.Count(content, oldString)
				if count > 1 {
					return "", fmt.Errorf("found multiple matches for oldString (found %d). Provide more surrounding context to make the match unique, or use replaceAll=true", count)
				}
				newContent = strings.Replace(content, oldString, newString, 1)
				replacements = 1
			}

			if len(newContent) > 0 && newContent[len(newContent)-1] != '\n' {
				newContent += "\n"
			}

			if err := fs.Write(ctx, relPath, []byte(newContent)); err != nil {
				return "", fmt.Errorf("failed to write file: %w", err)
			}

			return fmt.Sprintf("Successfully replaced %d occurrence(s) in %s", replacements, filePath), nil
		},
	)
}

func newWriteTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"write",
		"Write content to a file, creating it and any parent directories if they don't exist. Overwrites existing content entirely. Use this for creating new files or completely replacing file contents. The file will always end with a newline after writing.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "Absolute path to the file to write. Parent directories will be created if needed.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The complete content to write to the file.",
				},
			},
			"required": []string{"filePath", "content"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			filePath := asString(params, "filePath")
			if filePath == "" {
				return "", fmt.Errorf("filePath is required")
			}
			content := asString(params, "content")

			relPath, err := absoluteToRelative(fs, filePath)
			if err != nil {
				return "", err
			}

			if len(content) > 0 && content[len(content)-1] != '\n' {
				content += "\n"
			}

			if err := fs.Write(ctx, relPath, []byte(content)); err != nil {
				return "", fmt.Errorf("failed to write file: %w", err)
			}

			lines := strings.Count(content, "\n")
			return fmt.Sprintf("Successfully wrote %d bytes (%d lines) to %s", len(content), lines, filePath), nil
		},
	)
}
