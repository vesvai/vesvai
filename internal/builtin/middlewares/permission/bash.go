package permission

import (
	"strings"

	json "github.com/goccy/go-json"
)

var bashWhitelist = map[string]bool{
	"go":   true,
	"npm":  true,
	"ls":   true,
	"pwd":  true,
	"node": true,
	"git":  true,
}

func bashAllowed(args string) bool {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return false
	}
	cmd := strings.TrimSpace(params.Command)
	if cmd == "" {
		return false
	}
	if hasShellMetachar(cmd) {
		return false
	}
	first := strings.Fields(cmd)[0]
	return bashWhitelist[first]
}

func hasShellMetachar(s string) bool {
	return strings.ContainsAny(s, ";|&><$`\n\r") || strings.Contains(s, "&&") || strings.Contains(s, "||")
}
