package prompt

import (
	"strings"
)

type SkillInfo struct {
	Name        string
	Description string
}

type skillsList struct {
	base
	items []SkillInfo
}

func SkillsList(items []SkillInfo) Part {
	return &skillsList{items: items}
}

func (s *skillsList) renderMarkdown(_ *renderCtx) (string, error) {
	if len(s.items) == 0 {
		return "", nil
	}
	var sb strings.Builder
	for i, it := range s.items {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("- **" + it.Name + "**")
		if it.Description != "" {
			sb.WriteString(": " + it.Description)
		}
	}
	return sb.String(), nil
}

func (s *skillsList) renderXML(_ *renderCtx) (string, error) {
	var sb strings.Builder
	sb.WriteString("<skills>")
	if len(s.items) > 0 {
		sb.WriteString("\n")
	}
	for i, it := range s.items {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(`  <skill name="` + escapeXML(it.Name) + `" description="` + escapeXML(it.Description) + `"/>`)
	}
	if len(s.items) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("</skills>")
	return sb.String(), nil
}

func (s *skillsList) renderJSON(_ *renderCtx) (any, error) {
	items := make([]map[string]any, 0, len(s.items))
	for _, it := range s.items {
		items = append(items, map[string]any{"name": it.Name, "description": it.Description})
	}
	return map[string]any{"type": "skills", "items": items}, nil
}
