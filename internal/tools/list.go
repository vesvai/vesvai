package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/filesystem"
)

func newListTool(fs *filesystem.FileSystem) agent.Tool {
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

			relPath, err := absoluteToRelative(fs, dirPath)
			if err != nil {
				return "", err
			}

			entries, err := fs.List(ctx, relPath)
			if err != nil {
				return "", fmt.Errorf("failed to read directory: %w", err)
			}

			var sb strings.Builder
			for _, entry := range entries {
				if entry.IsDir {
					sb.WriteString(fmt.Sprintf("%s/\n", entry.Name))
				} else if entry.Mode&os.ModeSymlink != 0 {
					sb.WriteString(fmt.Sprintf("%s (symlink, %d bytes)\n", entry.Name, entry.Size))
				} else {
					lineStr := ""
					if lines := countFileLines(filepath.Join(fs.Root(), filepath.FromSlash(entry.Path)), entry.Size); lines > 0 {
						lineStr = fmt.Sprintf(", %d lines", lines)
					}
					sb.WriteString(fmt.Sprintf("%s (%d bytes%s)\n", entry.Name, entry.Size, lineStr))
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

func countFileLines(absPath string, size int64) int {
	if size <= 0 || size > 2<<20 {
		return 0
	}
	f, err := os.Open(absPath)
	if err != nil {
		return 0
	}
	defer f.Close()

	reader := bufio.NewReaderSize(f, 64*1024)
	count := 0
	endsWithNewline := false
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			endsWithNewline = buf[n-1] == '\n'
			for _, b := range buf[:n] {
				if b == '\n' {
					count++
				}
			}
		}
		if err != nil {
			break
		}
	}
	if !endsWithNewline {
		count++
	}
	return count
}
