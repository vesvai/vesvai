package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/skill"
)

func NewSkillTools(manager *skill.Manager) []agent.Tool {
	return []agent.Tool{
		newListSkillsTool(manager),
		newReadSkillTool(manager),
		newCreateSkillTool(manager),
	}
}

func newListSkillsTool(manager *skill.Manager) agent.Tool {
	return agent.NewFuncTool(
		"list-skills",
		"List all available skills from both global (~/.config/vesvai/skills/) and project (.vesvai/skills/) locations. Project skills override global skills with the same name. Returns skill names, descriptions, and locations.",
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			skills, err := manager.List()
			if err != nil {
				return "", fmt.Errorf("failed to list skills: %w", err)
			}

			if len(skills) == 0 {
				return "No skills found. Create skills in ~/.config/vesvai/skills/ (global) or .vesvai/skills/ (project).\n", nil
			}

			var sb strings.Builder
			sb.WriteString("Available Skills:\n\n")

			for _, s := range skills {
				sb.WriteString(fmt.Sprintf("• %s (%s)\n", s.Name, s.Location))
				if s.Description != "" {
					sb.WriteString(fmt.Sprintf("  %s\n", s.Description))
				}
			}

			sb.WriteString(fmt.Sprintf("\n(%d skills total)\n", len(skills)))
			return sb.String(), nil
		},
	)
}

func newReadSkillTool(manager *skill.Manager) agent.Tool {
	return agent.NewFuncTool(
		"read-skill",
		"Read a skill's content by name. Project skills take precedence over global skills with the same name. Returns the full markdown content of the skill file.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The skill name to read (without .md extension)",
				},
			},
			"required": []string{"name"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			name, _ := params["name"].(string)
			if name == "" {
				return "", fmt.Errorf("skill name is required")
			}

			skill, err := manager.Read(name)
			if err != nil {
				return "", err
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Skill: %s\n", skill.Name))
			sb.WriteString(fmt.Sprintf("Location: %s\n", skill.Location))
			sb.WriteString(fmt.Sprintf("Path: %s\n\n", skill.Path))
			sb.WriteString("---\n\n")
			sb.WriteString(skill.Content)

			return sb.String(), nil
		},
	)
}

func newCreateSkillTool(manager *skill.Manager) agent.Tool {
	return agent.NewFuncTool(
		"create-skill",
		"Create a new skill file. Agents can use this to create their own skills for future use. Skills are markdown files with optional frontmatter. Project skills are stored in .vesvai/skills/, global skills in ~/.config/vesvai/skills/.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Skill name (will be sanitized: lowercase, hyphens for spaces)",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The skill content in markdown format. Can include frontmatter with description, triggers, etc.",
				},
				"location": map[string]any{
					"type":        "string",
					"description": "Where to save the skill: 'project' for .vesvai/skills/, 'global' for ~/.config/vesvai/skills/",
					"enum":        []string{"project", "global"},
				},
			},
			"required": []string{"name", "content"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			name, _ := params["name"].(string)
			if name == "" {
				return "", fmt.Errorf("skill name is required")
			}

			content, _ := params["content"].(string)
			if content == "" {
				return "", fmt.Errorf("skill content is required")
			}

			locationStr, _ := params["location"].(string)
			var location skill.SkillLocation
			switch locationStr {
			case "global":
				location = skill.LocationGlobal
			case "project", "":
				location = skill.LocationProject
			default:
				return "", fmt.Errorf("invalid location: %s (use 'project' or 'global')", locationStr)
			}

			skill, err := manager.Create(name, content, location)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Successfully created skill '%s' at %s (%s)\n", skill.Name, skill.Path, skill.Location), nil
		},
	)
}
