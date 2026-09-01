package file

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func readTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"read",
		"Read a file from the workspace. Returns the file content with line numbers, along with metadata (path, hash, size, line count). Supports partial reads: use 'offset' (1-indexed starting line) and 'limit' (max lines to return) to read a specific range. If offset and limit are omitted, the entire file is returned. The file hash can be used to detect external changes before editing with the 'edit' tool. Respects .gitignore/.vesvaignore rules. Returns LSP diagnostics (errors, warnings) if available. Use this tool to examine file contents before making changes.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "Virtual path to the file within the workspace, relative to workspace root. Use '/' as separator. Examples: 'src/main.go', 'README.md', 'internal/config/config.go'.",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "Starting line number (1-indexed) for partial read. Set to 1 to read from the beginning. Omit or set to 0 to read from the start.",
					"minimum":     0,
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of lines to return. Useful for reading a specific section of a large file. Omit or set to 0 to read all lines to the end of the file.",
					"minimum":     0,
				},
			},
			"required": []string{"filePath"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				FilePath string `json:"filePath"`
				Offset   int    `json:"offset"`
				Limit    int    `json:"limit"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("read: invalid arguments: %w", err)
			}
			if params.FilePath == "" {
				return "", fmt.Errorf("read: filePath is required")
			}

			var result string
			var err error
			if params.Offset > 0 || params.Limit > 0 {
				result, err = fs.ReadRange(params.FilePath, params.Offset, params.Limit)
			} else {
				result, err = fs.Read(params.FilePath)
			}
			if err != nil {
				return "", fmt.Errorf("read: %w", err)
			}

			return result, nil
		},
	)
}