package file

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func writeTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"write",
		"Write content to a file, creating or overwriting it. Creates parent directories automatically if they don't exist. Uses atomic writes (writes to a temp file first, then renames) to prevent data corruption. Returns the file path, size, and hash. Use this tool to create new files or to completely replace the contents of an existing file. For surgical edits (replacing specific text), prefer the 'edit' tool instead. Respects .gitignore/.vesvaignore rules. Returns LSP diagnostics if available.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "Virtual path to the file within the workspace (e.g. 'src/main.go', 'README.md').",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Full content to write to the file. This completely replaces any existing content. Use the 'edit' tool instead if you only need to replace specific text.",
				},
			},
			"required": []string{"filePath", "content"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				FilePath string `json:"filePath"`
				Content  string `json:"content"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("write: invalid arguments: %w", err)
			}
			if params.FilePath == "" {
				return "", fmt.Errorf("write: filePath is required")
			}

			result, err := fs.WriteCtx(ctx, params.FilePath, []byte(params.Content))
			if err != nil {
				return "", fmt.Errorf("write: %w", err)
			}

			return result, nil
		},
	).SetPermissionError(isScopeError)
}
