package file

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func grepTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"grep",
		"Search file contents using a regular expression pattern. Supports three output modes: 'content' (default) — returns matching lines with line numbers, 'files_with_matches' — returns only file paths (deduplicated), 'count' — returns match count per file. Use 'include' to restrict search to specific file patterns (e.g. ['*.go', '*.ts']). Use 'headLimit' to cap the number of results. Respects .gitignore/.vesvaignore rules. Skips binary files. Use this tool to find all references to a function, variable, or string across the workspace.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "Regular expression pattern to search for in file contents. Uses Go regexp syntax. Examples: 'func\\s+\\w+', 'TODO|FIXME', 'error.*handl'.",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Directory to search from, relative to workspace root. Omit or set to empty string to search the entire workspace. Example: 'internal', 'src/components'.",
				},
				"include": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Glob patterns to filter files by name. Only files matching at least one include pattern will be searched. If empty, all non-ignored files are searched. Examples: ['*.go', '*.ts', '*.{js,jsx}'].",
				},
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"content", "files_with_matches", "count"},
					"description": "Output mode: 'content' (default) returns matching lines with line numbers, 'files_with_matches' returns only file paths, 'count' returns match count per file.",
				},
				"headLimit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return. 0 means unlimited. Useful to avoid overwhelming output from a broad search.",
					"minimum":     0,
				},
			},
			"required": []string{"pattern"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				Pattern   string   `json:"pattern"`
				Path      string   `json:"path"`
				Include   []string `json:"include"`
				Mode      string   `json:"mode"`
				HeadLimit int      `json:"headLimit"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("grep: invalid arguments: %w", err)
			}
			if params.Pattern == "" {
				return "", fmt.Errorf("grep: pattern is required")
			}

			mode := vfs.GrepMode(params.Mode)
			if mode == "" {
				mode = vfs.GrepModeContent
			}

			results, err := fs.Grep(params.Pattern, params.Path, params.Include, mode, params.HeadLimit)
			if err != nil {
				return "", fmt.Errorf("grep: %w", err)
			}

			if len(results) == 0 {
				return "No matches found.\n", nil
			}

			return formatGrepResults(results, mode), nil
		},
	)
}

func formatGrepResults(results []vfs.GrepResult, mode vfs.GrepMode) string {
	var b strings.Builder
	limited := false
	if mode == vfs.GrepModeFilesWithMatches {
		fmt.Fprintf(&b, "Matches: %d files\n", len(results))
		limited = len(results) >= 100
		for _, r := range results {
			fmt.Fprintf(&b, "  %s\n", r.Path)
		}
	} else if mode == vfs.GrepModeCount {
		total := 0
		for _, r := range results {
			total += r.Count
		}
		fmt.Fprintf(&b, "Matches: %d files, %d total matches\n", len(results), total)
		for _, r := range results {
			fmt.Fprintf(&b, "  %5d  %s\n", r.Count, r.Path)
		}
	} else {
		fmt.Fprintf(&b, "Matches: %d\n", len(results))
		limited = len(results) >= 100
		for _, r := range results {
			fmt.Fprintf(&b, "  %s:%d: %s\n", r.Path, r.Line, r.Text)
		}
	}
	if limited {
		fmt.Fprintf(&b, "... results truncated (limit reached)\n")
	}
	return b.String()
}
