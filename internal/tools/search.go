package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/filesystem"
)

func newGlobTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"glob",
		"Find files matching a glob pattern. Returns matching file paths sorted by modification time (most recent first). Use this to find files by name pattern, like all Go files, all TypeScript files, or files matching a specific naming convention. Supports standard glob syntax: * matches any characters, ** matches directories recursively.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "Glob pattern to match. Examples: '*.go' for Go files, '**/*.ts' for all TypeScript files, 'src/**/*.test.*' for test files.",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Directory to search in. Defaults to current working directory.",
				},
			},
			"required": []string{"pattern"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			pattern := asString(params, "pattern")
			if pattern == "" {
				return "", fmt.Errorf("pattern is required")
			}

			searchPath := asString(params, "path")
			if searchPath != "" {
				rel, err := absoluteToRelative(fs, searchPath)
				if err != nil {
					return "", err
				}
				if strings.Contains(pattern, "/") {
					pattern = rel + "/" + pattern
				} else {
					pattern = rel + "/**/" + pattern
				}
			}

			matches, err := fs.Glob(ctx, pattern)
			if err != nil {
				return "", fmt.Errorf("invalid glob pattern: %w", err)
			}

			if len(matches) == 0 {
				return "No files found matching pattern\n", nil
			}

			var sb strings.Builder
			for _, item := range matches {
				sb.WriteString(fmt.Sprintf("%s\n", item.Path))
			}

			sb.WriteString(fmt.Sprintf("\n(%d files found)\n", len(matches)))
			return sb.String(), nil
		},
	)
}

func newGrepTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"grep",
		"Search file contents using regex patterns. Use this to find where a function is defined, where a variable is used, or locate specific code patterns. Supports filtering by file extension. Output modes: 'content' shows matching lines with file and line numbers, 'files_with_matches' shows only file paths, 'count' shows match counts per file. Respects .gitignore and .vesvaignore.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "Regex pattern to search for (Go regexp syntax). Examples: 'func\\s+main' for main functions, 'TODO|FIXME' for task markers.",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Directory to search in. Defaults to current working directory.",
				},
				"include": map[string]any{
					"type":        "string",
					"description": "File pattern to include (e.g. '*.go', '*.ts', '*.{ts,tsx}'). Only matching files are searched.",
				},
				"output_mode": map[string]any{
					"type":        "string",
					"description": "Output format. 'content' (default) shows file:line: match. 'files_with_matches' shows unique file paths. 'count' shows matches per file.",
					"enum":        []string{"content", "files_with_matches", "count"},
				},
				"head_limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return. 0 means no limit. Use for large codebases to avoid overwhelming output.",
				},
			},
			"required": []string{"pattern"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			pattern := asString(params, "pattern")
			if pattern == "" {
				return "", fmt.Errorf("pattern is required")
			}

			searchPath := asString(params, "path")
			if searchPath == "" {
				searchPath = "."
			} else {
				rel, err := absoluteToRelative(fs, searchPath)
				if err != nil {
					return "", err
				}
				searchPath = rel
			}

			outputMode := asString(params, "output_mode")
			if outputMode == "" {
				outputMode = "content"
			}

			headLimit := asInt(params, "head_limit")

			include := asString(params, "include")

			results, err := fs.Grep(ctx, pattern, searchPath)
			if err != nil {
				return "", fmt.Errorf("grep failed: %w", err)
			}

			if include != "" {
				includeRegex := strings.ReplaceAll(include, ".", "\\.")
				includeRegex = strings.ReplaceAll(includeRegex, "*", ".*")
				includeRegex = strings.ReplaceAll(includeRegex, "?", ".")
				includePattern, err := regexp.Compile(includeRegex + "$")
				if err != nil {
					return "", fmt.Errorf("invalid include pattern: %w", err)
				}
				var filtered []filesystem.GrepResult
				for _, r := range results {
					if includePattern.MatchString(r.Path) {
						filtered = append(filtered, r)
					}
				}
				results = filtered
			}

			if headLimit > 0 && len(results) > headLimit {
				results = results[:headLimit]
			}

			var sb strings.Builder
			switch outputMode {
			case "files_with_matches":
				seen := make(map[string]bool)
				for _, m := range results {
					if !seen[m.Path] {
						sb.WriteString(m.Path + "\n")
						seen[m.Path] = true
					}
				}
				if len(seen) == 0 {
					sb.WriteString("No matches found\n")
				} else {
					sb.WriteString(fmt.Sprintf("\n(%d files with matches)\n", len(seen)))
				}

			case "count":
				fileMatches := make(map[string]int)
				for _, m := range results {
					fileMatches[m.Path]++
				}
				total := 0
				for file, count := range fileMatches {
					sb.WriteString(fmt.Sprintf("%s: %d\n", file, count))
					total += count
				}
				if total == 0 {
					sb.WriteString("No matches found\n")
				} else {
					sb.WriteString(fmt.Sprintf("\n(%d matches in %d files)\n", total, len(fileMatches)))
				}

			default:
				for _, m := range results {
					sb.WriteString(fmt.Sprintf("%s:%d: %s\n", m.Path, m.Line, m.Content))
				}
				if len(results) == 0 {
					sb.WriteString("No matches found\n")
				} else {
					sb.WriteString(fmt.Sprintf("\n(%d matches)\n", len(results)))
				}
			}

			return sb.String(), nil
		},
	)
}
