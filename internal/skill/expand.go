package skill

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	skillRefRe = regexp.MustCompile(`(^|[^A-Za-z0-9_/:.-])(/[a-z0-9]+(?:-[a-z0-9]+)*)`)
)

func ExpandMessage(msg string) string {
	if !strings.Contains(msg, "/") {
		return msg
	}
	return skillRefRe.ReplaceAllStringFunc(msg, func(m string) string {
		parts := skillRefRe.FindStringSubmatch(m)
		if len(parts) != 3 {
			return m
		}
		prefix, token := parts[1], parts[2]
		s, ok := Get(strings.TrimPrefix(token, "/"))
		if !ok {
			return m
		}
		return prefix + skillBlock(s)
	})
}

func skillBlock(s *Skill) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n<skill:%s>\n", s.Name)
	if s.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", s.Description)
	}
	fmt.Fprintf(&b, "Path: %s\n", s.Path)
	if s.HasScripts {
		fmt.Fprintf(&b, "Scripts: %s\n", s.ScriptsPath)
	}
	if len(s.AllowedTools) > 0 {
		fmt.Fprintf(&b, "Allowed tools: %s\n", strings.Join(s.AllowedTools, ", "))
	}
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(s.Instructions))
	fmt.Fprintf(&b, "\n</skill:%s>\n", s.Name)
	return b.String()
}
