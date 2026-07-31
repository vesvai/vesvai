package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func IsBinary(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	check := data
	if len(check) > 512 {
		check = check[:512]
	}

	contentType := http.DetectContentType(check)
	switch {
	case strings.HasPrefix(contentType, "application/octet-stream"):
		return true
	case strings.HasPrefix(contentType, "image/"):
		return true
	case strings.HasPrefix(contentType, "audio/"):
		return true
	case strings.HasPrefix(contentType, "video/"):
		return true
	}

	limit := len(data)
	if limit > 256 {
		limit = 256
	}
	for i := 0; i < limit; i++ {
		if data[i] == 0 {
			return true
		}
	}

	return false
}

func isDotPath(relPath string) bool {
	parts := strings.Split(relPath, string(filepath.Separator))
	for _, p := range parts {
		if strings.HasPrefix(p, ".") && p != "." && p != ".." {
			return true
		}
	}
	return false
}

func cleanPath(relPath string) string {
	relPath = filepath.Clean(relPath)
	relPath = filepath.ToSlash(relPath)
	relPath = strings.TrimPrefix(relPath, "./")
	relPath = strings.TrimPrefix(relPath, "/")
	return relPath
}

func ExpandHome(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return home, nil
	}

	if len(path) > 1 && path[1] == '/' {
		return home + path[1:], nil
	}

	return path, nil
}

func hashFileContent(absPath string) (string, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

func toRegexPattern(pattern string) string {
	pattern = strings.TrimPrefix(pattern, "/")

	var result strings.Builder
	result.WriteString("(^|/)")

	i := 0
	for i < len(pattern) {
		c := pattern[i]

		if c == '*' && i+1 < len(pattern) && pattern[i+1] == '*' {
			if i+2 < len(pattern) && pattern[i+2] == '/' {
				result.WriteString("(.*/)?")
				i += 3
			} else if i+2 >= len(pattern) {
				result.WriteString("(.+)?")
				i += 2
			} else {
				result.WriteString("(.*/)?")
				i += 2
			}
		} else if c == '*' {
			result.WriteString("[^/]*")
			i += 1
		} else if c == '?' {
			result.WriteString("[^/]")
			i += 1
		} else if c == '[' {
			end := strings.IndexByte(pattern[i:], ']')
			if end != -1 {
				result.WriteString(pattern[i : i+end+1])
				i += end + 1
			} else {
				result.WriteString("\\[")
				i += 1
			}
		} else if c == '.' || c == '(' || c == ')' || c == '+' || c == '^' || c == '$' || c == '{' || c == '}' || c == '|' || c == '\\' {
			result.WriteByte('\\')
			result.WriteByte(c)
			i += 1
		} else {
			result.WriteByte(c)
			i += 1
		}
	}

	return result.String()
}
