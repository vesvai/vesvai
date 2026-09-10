package file

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"
	"strings"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func globTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"glob",
		"Find files matching a glob pattern. Supports '**' (recursive), '*' (single segment), and '?' (single character) wildcards. Results are sorted by modification time (newest first). If 'path' is omitted, searches from the workspace root. Respects .gitignore/.vesvaignore rules. Use this tool to discover files matching a pattern, such as finding all Go source files ('**/*.go') or all test files ('**/*_test.go').",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "Glob pattern to match files. Use '**' for recursive matching, '*' for single-segment matching, '?' for single character. Examples: '**/*.go', 'src/**/*.ts', 'internal/**/*_test.go', '*.md'.",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Directory to search from, relative to workspace root. Omit or set to empty string to search from the workspace root. Example: 'internal', 'src/components'.",
				},
			},
			"required": []string{"pattern"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				Pattern string `json:"pattern"`
				Path    string `json:"path"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("glob: invalid arguments: %w", err)
			}
			if params.Pattern == "" {
				return "", fmt.Errorf("glob: pattern is required")
			}

			results, err := fs.GlobCtx(ctx, params.Pattern, params.Path)
			if err != nil {
				return "", fmt.Errorf("glob: %w", err)
			}

			if len(results) == 0 {
				return "No files matched the pattern.\n", nil
			}

			var b strings.Builder
			fmt.Fprintf(&b, "Pattern: %s | Matches: %d\n", params.Pattern, len(results))
			for _, r := range results {
				fmt.Fprintf(&b, "  %s\n", r)
			}
			return b.String(), nil
		},
	).SetPermissionError(isScopeError)
}
