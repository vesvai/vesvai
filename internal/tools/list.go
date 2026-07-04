package tools

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
)

func newListTool() agent.Tool {
	return agent.NewFuncTool(
		"list",
		"List files and directories at a given path. Returns entries sorted with directories first, then files alphabetically. Each entry shows its type (file, directory, symlink) and size. Use this to explore project structure, find files, or understand directory layout before reading or editing.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Absolute path to the directory to list. Supports ~ for home directory.",
				},
			},
			"required": []string{"path"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			dirPath := asString(params, "path")
			if dirPath == "" {
				return "", fmt.Errorf("path is required")
			}

			abs, err := resolvePath(dirPath)
			if err != nil {
				return "", err
			}

			entries, err := os.ReadDir(abs)
			if err != nil {
				return "", fmt.Errorf("failed to read directory: %w", err)
			}

			sort.SliceStable(entries, func(i, j int) bool {
				if entries[i].IsDir() != entries[j].IsDir() {
					return entries[i].IsDir()
				}
				return entries[i].Name() < entries[j].Name()
			})

			var sb strings.Builder
			for _, entry := range entries {
				info, err := entry.Info()
				if err != nil {
					if entry.IsDir() {
						sb.WriteString(fmt.Sprintf("%s/\n", entry.Name()))
					} else {
						sb.WriteString(fmt.Sprintf("%s\n", entry.Name()))
					}
					continue
				}

				if entry.IsDir() {
					sb.WriteString(fmt.Sprintf("%s/\n", entry.Name()))
				} else if entry.Type()&os.ModeSymlink != 0 {
					sb.WriteString(fmt.Sprintf("%s (symlink, %d bytes)\n", entry.Name(), info.Size()))
				} else {
					sb.WriteString(fmt.Sprintf("%s (%d bytes)\n", entry.Name(), info.Size()))
				}
			}

			if len(entries) == 0 {
				sb.WriteString("(empty directory)\n")
			} else {
				sb.WriteString(fmt.Sprintf("\n(%d entries)\n", len(entries)))
			}

			return sb.String(), nil
		},
	)
}
