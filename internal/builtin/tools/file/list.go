package file

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func listTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"list",
		"List files and directories in a workspace directory. Returns a sorted table of entries with name, path, size, and type (file/directory). Also returns summary counts (files, directories, total size). If 'path' is omitted, lists the workspace root. Respects .gitignore/.vesvaignore rules (ignored files/directories are not shown). Use this tool to explore the workspace structure and find files.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Virtual path to the directory within the workspace. Omit or set to empty string to list the workspace root. Examples: 'src', 'internal/config', 'docs'.",
				},
			},
			"required": []string{},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("list: invalid arguments: %w", err)
			}

			dir := params.Path
			if dir == "" {
				dir = "."
			}

			result, err := fs.ListCtx(ctx, dir)
			if err != nil {
				return "", fmt.Errorf("list: %w", err)
			}

			return formatListResult(result), nil
		},
	).SetPermissionError(isScopeError)
}

func formatListResult(r vfs.ListResult) string {
	dir := r.Path
	if dir == "" {
		dir = "."
	}
	out := fmt.Sprintf("Directory: %s\n", dir)
	out += fmt.Sprintf("Files: %d | Directories: %d | Total size: %d bytes\n", r.FileCount, r.DirCount, r.TotalSize)
	if len(r.Entries) == 0 {
		out += "(empty)\n"
		return out
	}
	out += "---\n"
	for _, e := range r.Entries {
		typeStr := "     "
		if e.IsDir {
			typeStr = "dir  "
		}
		out += fmt.Sprintf("%s %s %s\n", typeStr, formatSize(e.Size), e.Path)
	}
	return out
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%5d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%5.1fKB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%5.1fMB", float64(size)/(1024*1024))
	}
}
