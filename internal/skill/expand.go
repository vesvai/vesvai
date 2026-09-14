package skill

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	skillRefRe = regexp.MustCompile(`(^|[^A-Za-z0-9_/:.-])(/[a-z0-9]+(?:-[a-z0-9]+)*)`)
	argRefRe   = regexp.MustCompile(`\$([a-zA-Z][a-zA-Z0-9_]*)`)
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
	if s.WhenToUse != "" {
		fmt.Fprintf(&b, "When to use: %s\n", s.WhenToUse)
	}
	if s.ArgumentHint != "" {
		fmt.Fprintf(&b, "Arguments: %s\n", s.ArgumentHint)
	}
	if len(s.Arguments) > 0 {
		fmt.Fprintf(&b, "Parameters: %s\n", strings.Join(s.Arguments, ", "))
	}
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(s.Instructions))
	fmt.Fprintf(&b, "\n</skill:%s>\n", s.Name)
	return b.String()
}

func ExpandWithArgs(s *Skill, args map[string]string) string {
	if s == nil {
		return ""
	}
	content := s.Instructions
	if len(args) > 0 {
		content = argRefRe.ReplaceAllStringFunc(content, func(m string) string {
			parts := argRefRe.FindStringSubmatch(m)
			if len(parts) != 2 {
				return m
			}
			if val, ok := args[parts[1]]; ok {
				return val
			}
			return m
		})
	}
	return content
}
