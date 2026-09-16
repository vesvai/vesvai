package prompt

import (
	"strings"
)

type SkillInfo struct {
	Name         string
	Description  string
	WhenToUse    string
	ArgumentHint string
	Arguments    []string
	Context      string
}

type hr struct {
	base
	count int
}

func Hr(count int) Part {
	if count < 1 {
		count = 1
	}
	return &hr{count: count}
}

func (h *hr) renderMarkdown(_ *renderCtx) (string, error) {
	return strings.Repeat("-", h.count), nil
}

func (h *hr) renderXML(_ *renderCtx) (string, error) {
	return strings.Repeat("-", h.count), nil
}

func (h *hr) renderJSON(_ *renderCtx) (any, error) {
	return map[string]any{"type": "hr", "count": h.count}, nil
}

func (p *Prompt) Hr(count int) *Prompt { return p.Add(Hr(count)) }

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
		if it.WhenToUse != "" {
			sb.WriteString(" (when: " + it.WhenToUse + ")")
		}
		if len(it.Arguments) > 0 {
			sb.WriteString(" [args: " + strings.Join(it.Arguments, ", ") + "]")
		}
		if it.ArgumentHint != "" {
			sb.WriteString(" " + it.ArgumentHint)
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
		sb.WriteString(`  <skill name="` + escapeXML(it.Name) + `" description="` + escapeXML(it.Description) + `"`)
		if it.WhenToUse != "" {
			sb.WriteString(` when_to_use="` + escapeXML(it.WhenToUse) + `"`)
		}
		if len(it.Arguments) > 0 {
			sb.WriteString(` arguments="` + escapeXML(strings.Join(it.Arguments, ",")) + `"`)
		}
		if it.Context != "" {
			sb.WriteString(` context="` + escapeXML(it.Context) + `"`)
		}
		sb.WriteString("/>")
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
		item := map[string]any{"name": it.Name, "description": it.Description}
		if it.WhenToUse != "" {
			item["when_to_use"] = it.WhenToUse
		}
		if len(it.Arguments) > 0 {
			item["arguments"] = it.Arguments
		}
		if it.ArgumentHint != "" {
			item["argument_hint"] = it.ArgumentHint
		}
		if it.Context != "" {
			item["context"] = it.Context
		}
		items = append(items, item)
	}
	return map[string]any{"type": "skills", "items": items}, nil
}
