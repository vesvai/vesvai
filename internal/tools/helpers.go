package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

func readLines(filePath string, offset, limit int) ([]string, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum < offset {
			continue
		}
		if limit > 0 && len(lines) >= limit {
			break
		}
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	totalLines := lineNum
	return lines, totalLines, nil
}

func resolvePath(p string) (string, error) {
	expanded := p
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home dir: %w", err)
		}
		expanded = filepath.Join(home, p[1:])
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	return abs, nil
}
