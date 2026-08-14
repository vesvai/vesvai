package prompt

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/skill"
)

var registry = &promptRegistry{}

type promptRegistry struct {
	mu    sync.RWMutex
	hooks *hook.Hooks
}

func SetHooks(hooks *hook.Hooks) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.hooks = hooks
}

func currentHooks() *hook.Hooks {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	return registry.hooks
}

func (b *Builder) Tools(items ...string) *Builder {
	hs := currentHooks()
	if len(items) > 0 {
		return b.Section("tools", RenderToolsXMLByName(items, collectTools(hs)), OrderTools)
	}
	if hs == nil {
		return b
	}
	tools := collectTools(hs)
	if len(tools) == 0 {
		return b
	}
	return b.Section("tools", RenderToolsXML(tools), OrderTools)
}

func collectTools(hs *hook.Hooks) []agent.Tool {
	if hs == nil {
		return nil
	}
	value := hs.ApplyFilter(context.Background(), hook.HookToolsCollect, []agent.Tool{})
	tools, _ := value.([]agent.Tool)
	return tools
}

func (b *Builder) Skills(items ...string) *Builder {
	hs := currentHooks()
	if len(items) > 0 {
		return b.Section("skills", RenderSkillsXMLByName(items, collectSkills(hs)), OrderSkills)
	}
	if hs == nil {
		return b
	}
	skills := collectSkills(hs)
	if len(skills) == 0 {
		return b
	}
	return b.Section("skills", RenderSkillsXML(skills), OrderSkills)
}

func collectSkills(hs *hook.Hooks) []skill.Skill {
	if hs == nil {
		return nil
	}
	value := hs.ApplyFilter(context.Background(), hook.HookSkillsCollect, []skill.Skill{})
	skills, _ := value.([]skill.Skill)
	return skills
}

type paramInfo struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

func schemaParams(schema map[string]any) []paramInfo {
	if len(schema) == 0 {
		return nil
	}
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return nil
	}
	required := map[string]bool{}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)

	params := make([]paramInfo, 0, len(names))
	for _, name := range names {
		p, _ := props[name].(map[string]any)
		ptype, _ := p["type"].(string)
		desc, _ := p["description"].(string)
		params = append(params, paramInfo{
			Name:        name,
			Type:        ptype,
			Description: desc,
			Required:    required[name],
		})
	}
	return params
}

func RenderToolsXML(tools []agent.Tool) string {
	if len(tools) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<tools>")
	for _, t := range tools {
		writeToolXML(&sb, t)
	}
	sb.WriteString("\n</tools>")
	return sb.String()
}

func RenderToolsXMLByName(names []string, tools []agent.Tool) string {
	if len(names) == 0 {
		return ""
	}
	byName := make(map[string]agent.Tool, len(tools))
	for _, t := range tools {
		byName[t.Name()] = t
	}
	var sb strings.Builder
	sb.WriteString("<tools>")
	for _, name := range names {
		if t, ok := byName[name]; ok {
			writeToolXML(&sb, t)
		} else {
			sb.WriteString(fmt.Sprintf("\n  <tool name=\"%s\" />", xmlEscape(name)))
		}
	}
	sb.WriteString("\n</tools>")
	return sb.String()
}

func writeToolXML(sb *strings.Builder, t agent.Tool) {
	sb.WriteString(fmt.Sprintf("\n  <tool name=\"%s\">", xmlEscape(t.Name())))
	if d := t.Description(); d != "" {
		sb.WriteString("\n    <description>")
		sb.WriteString(xmlEscape(d))
		sb.WriteString("</description>")
	}
	if params := schemaParams(t.Schema()); len(params) > 0 {
		sb.WriteString("\n    <parameters>")
		for _, p := range params {
			sb.WriteString(fmt.Sprintf("\n      <parameter name=\"%s\" type=\"%s\"",
				xmlEscape(p.Name), xmlEscape(p.Type)))
			if p.Required {
				sb.WriteString(" required=\"true\"")
			}
			if p.Description != "" {
				sb.WriteString(fmt.Sprintf(" description=\"%s\"", xmlEscape(p.Description)))
			}
			sb.WriteString(" />")
		}
		sb.WriteString("\n    </parameters>")
	}
	sb.WriteString("\n  </tool>")
}

func RenderSkillsXML(skills []skill.Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<skills>")
	for _, s := range skills {
		writeSkillXML(&sb, s)
	}
	sb.WriteString("\n</skills>")
	return sb.String()
}

func RenderSkillsXMLByName(names []string, skills []skill.Skill) string {
	if len(names) == 0 {
		return ""
	}
	byName := make(map[string]skill.Skill, len(skills))
	for _, s := range skills {
		byName[s.Name] = s
	}
	var sb strings.Builder
	sb.WriteString("<skills>")
	for _, name := range names {
		if s, ok := byName[name]; ok {
			writeSkillXML(&sb, s)
		} else {
			sb.WriteString(fmt.Sprintf("\n  <skill name=\"%s\" />", xmlEscape(name)))
		}
	}
	sb.WriteString("\n</skills>")
	return sb.String()
}

func writeSkillXML(sb *strings.Builder, s skill.Skill) {
	sb.WriteString(fmt.Sprintf("\n  <skill name=\"%s\">", xmlEscape(s.Name)))
	if s.Description != "" {
		sb.WriteString("\n    <description>")
		sb.WriteString(xmlEscape(s.Description))
		sb.WriteString("</description>")
	}
	sb.WriteString("\n  </skill>")
}

func xmlEscape(s string) string {
	var sb strings.Builder
	_ = xml.EscapeText(&sb, []byte(s))
	return sb.String()
}
