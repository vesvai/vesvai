package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vesvai/vesvai/internal/filesystem"
)

func asInt(params map[string]any, key string) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case json.Number:
			i, _ := n.Int64()
			return int(i)
		}
	}
	return 0
}

func asString(params map[string]any, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func asBool(params map[string]any, key string, fallback bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return fallback
}

func asFloat64(params map[string]any, key string, fallback float64) float64 {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case json.Number:
			f, _ := n.Float64()
			return f
		}
	}
	return fallback
}

func absoluteToRelative(fs *filesystem.FileSystem, absPath string) (string, error) {
	absPath = filepath.Clean(absPath)

	if strings.HasPrefix(absPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home dir: %w", err)
		}
		absPath = filepath.Join(home, absPath[2:])
	} else if absPath == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home dir: %w", err)
		}
		absPath = home
	}

	root := fs.Root()

	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, absPath)
	}
	absPath = filepath.Clean(absPath)

	if absPath == root {
		return ".", nil
	}

	if !strings.HasPrefix(absPath, root+string(filepath.Separator)) && absPath != root {
		return "", fmt.Errorf("path %s is outside project root %s", absPath, root)
	}

	relPath, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(relPath), nil
}

func expandHome(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home dir: %w", err)
		}
		return filepath.Join(home, p[2:]), nil
	}
	if p == "~" {
		return os.UserHomeDir()
	}
	return filepath.Abs(p)
}
