package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vesvai/vesvai/internal/agent"
)

func newGlobTool() agent.Tool {
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
			if searchPath == "" {
				var err error
				searchPath, err = os.Getwd()
				if err != nil {
					return "", fmt.Errorf("failed to get working directory: %w", err)
				}
			}

			abs, err := resolvePath(searchPath)
			if err != nil {
				return "", err
			}

			fullPattern := pattern
			if !filepath.IsAbs(pattern) {
				fullPattern = filepath.Join(abs, pattern)
			}

			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				return "", fmt.Errorf("invalid glob pattern: %w", err)
			}

			if len(matches) == 0 {
				return "No files found matching pattern\n", nil
			}

			type matchInfo struct {
				path    string
				modTime time.Time
			}
			var items []matchInfo
			for _, m := range matches {
				info, err := os.Stat(m)
				if err != nil {
					continue
				}
				items = append(items, matchInfo{path: m, modTime: info.ModTime()})
			}

			sort.SliceStable(items, func(i, j int) bool {
				return items[i].modTime.After(items[j].modTime)
			})

			var sb strings.Builder
			for _, item := range items {
				rel, err := filepath.Rel(abs, item.path)
				if err != nil {
					rel = item.path
				}
				sb.WriteString(fmt.Sprintf("%s\n", rel))
			}

			sb.WriteString(fmt.Sprintf("\n(%d files found)\n", len(items)))
			return sb.String(), nil
		},
	)
}

func newGrepTool() agent.Tool {
	return agent.NewFuncTool(
		"grep",
		"Search file contents using regex patterns. Use this to find where a function is defined, where a variable is used, or locate specific code patterns. Supports filtering by file extension. Output modes: 'content' shows matching lines with file and line numbers, 'files_with_matches' shows only file paths, 'count' shows match counts per file. Skips hidden directories, node_modules, vendor, and .git by default.",
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

			re, err := regexp.Compile(pattern)
			if err != nil {
				return "", fmt.Errorf("invalid regex pattern: %w", err)
			}

			searchPath := asString(params, "path")
			if searchPath == "" {
				searchPath, _ = os.Getwd()
			}

			abs, err := resolvePath(searchPath)
			if err != nil {
				return "", err
			}

			outputMode := asString(params, "output_mode")
			if outputMode == "" {
				outputMode = "content"
			}

			headLimit := asInt(params, "head_limit")

			include := asString(params, "include")

			var includePattern *regexp.Regexp
			if include != "" {
				includeRegex := strings.ReplaceAll(include, ".", "\\.")
				includeRegex = strings.ReplaceAll(includeRegex, "*", ".*")
				includeRegex = strings.ReplaceAll(includeRegex, "?", ".")
				includePattern, err = regexp.Compile(includeRegex + "$")
				if err != nil {
					return "", fmt.Errorf("invalid include pattern: %w", err)
				}
			}

			var files []string
			err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					name := info.Name()
					if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == ".git" {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.HasPrefix(info.Name(), ".") {
					return nil
				}
				if includePattern != nil && !includePattern.MatchString(path) {
					return nil
				}
				files = append(files, path)
				return nil
			})
			if err != nil {
				return "", fmt.Errorf("failed to walk directory: %w", err)
			}

			if len(files) == 0 {
				return "No files found to search\n", nil
			}

			type match struct {
				file    string
				line    int
				content string
			}
			var matches []match
			fileMatches := make(map[string]int)

			for _, filePath := range files {
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}

				scanner := bufio.NewScanner(strings.NewReader(string(data)))
				lineNum := 0
				for scanner.Scan() {
					lineNum++
					line := scanner.Text()
					if re.MatchString(line) {
						rel, _ := filepath.Rel(abs, filePath)
						if outputMode == "content" {
							matches = append(matches, match{file: rel, line: lineNum, content: line})
						}
						fileMatches[rel]++
					}
				}

				if headLimit > 0 && len(matches) >= headLimit {
					break
				}
			}

			var sb strings.Builder
			switch outputMode {
			case "files_with_matches":
				seen := make(map[string]bool)
				for _, m := range matches {
					if !seen[m.file] {
						sb.WriteString(m.file + "\n")
						seen[m.file] = true
					}
				}
				if len(seen) == 0 {
					sb.WriteString("No matches found\n")
				} else {
					sb.WriteString(fmt.Sprintf("\n(%d files with matches)\n", len(seen)))
				}

			case "count":
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
				for _, m := range matches {
					sb.WriteString(fmt.Sprintf("%s:%d: %s\n", m.file, m.line, m.content))
				}
				if len(matches) == 0 {
					sb.WriteString("No matches found\n")
				} else {
					sb.WriteString(fmt.Sprintf("\n(%d matches)\n", len(matches)))
				}
			}

			return sb.String(), nil
		},
	)
}
