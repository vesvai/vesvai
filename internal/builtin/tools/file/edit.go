package file

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func editTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"edit",
		"Apply a text replacement to a file. The file must have been read first with the 'read' tool (a snapshot is required). Replaces the FIRST occurrence of 'oldString' by default. Set 'replaceAll' to true to replace ALL occurrences. Returns an error if the file has changed externally since the last read (hash mismatch) — in that case re-read the file first. Returns an error if 'oldString' is not found in the file. Use this tool for surgical edits instead of rewriting the entire file with 'write'. Preserves file permissions.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "Virtual path to the file within the workspace (e.g. 'src/main.go').",
				},
				"oldString": map[string]any{
					"type":        "string",
					"description": "Exact text to search for in the file. Must match exactly including whitespace and indentation. The first occurrence is replaced by default.",
				},
				"newString": map[string]any{
					"type":        "string",
					"description": "Replacement text that will replace 'oldString' in the file.",
				},
				"replaceAll": map[string]any{
					"type":        "boolean",
					"description": "If true, replace ALL occurrences of 'oldString'. If false (default), replace only the first occurrence.",
				},
			},
			"required": []string{"filePath", "oldString", "newString"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				FilePath   string `json:"filePath"`
				OldString  string `json:"oldString"`
				NewString  string `json:"newString"`
				ReplaceAll bool   `json:"replaceAll"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("edit: invalid arguments: %w", err)
			}
			if params.FilePath == "" {
				return "", fmt.Errorf("edit: filePath is required")
			}
			if params.OldString == "" {
				return "", fmt.Errorf("edit: oldString is required")
			}

			result, err := fs.Edit(params.FilePath, params.OldString, params.NewString, params.ReplaceAll)
			if err != nil {
				return "", fmt.Errorf("edit: %w", err)
			}

			return result, nil
		},
	)
}