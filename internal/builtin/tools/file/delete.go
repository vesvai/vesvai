package file

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func deleteTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"delete",
		"Delete a file from the workspace. The file must have been read first with the 'read' tool (a snapshot is required for safety). Returns an error if the file does not exist. Respects .gitignore/.vesvaignore rules. Use this tool to permanently remove files from the workspace.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "Virtual path to the file within the workspace (e.g. 'src/main.go', 'README.md').",
				},
			},
			"required": []string{"filePath"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				FilePath string `json:"filePath"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("delete: invalid arguments: %w", err)
			}
			if params.FilePath == "" {
				return "", fmt.Errorf("delete: filePath is required")
			}

			if err := fs.Delete(params.FilePath); err != nil {
				return "", fmt.Errorf("delete: %w", err)
			}

			return "File deleted successfully.", nil
		},
	)
}
