package plan

import (
	"context"
	"fmt"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/prompt"
	"github.com/vesvai/vesvai/internal/agent/reminder"
	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/vfs"
)

func PlanTools(fs *vfs.VFS) {
	tools.Register(enterplanmodeTool(fs))
	tools.Register(exitplanmodeTool(fs))
}

func enterplanmodeToolPrompt() (string, error) {
	sys, err := enterplanmodeToolPromptBuilder().
		Build(prompt.FormatMarkdown)
	if err != nil {
		return "", err
	}
	return sys, nil
}

func enterplanmodeTool(fs *vfs.VFS) tool.Tool {
	prompt, err := enterplanmodeToolPrompt()
	if err != nil {
		panic(fmt.Sprintf("failed to generate enterplanmode tool prompt: %v", err))
	}

	return tool.NewSpec(
		"enterplanmode",
		prompt,
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		},
		func(ctx context.Context, args string) (string, error) {
			parent := agent.FromContext(ctx)
			if parent == nil {
				return "", fmt.Errorf("enterplanmode: no parent agent in context")
			}
			parent.AttachReminder(PlanModeReminder())
			return "Plan mode enabled. You are now in READ-ONLY planning mode and must not make any edits or run any non-readonly tools until the user approves a plan. Call exitplanmode when the plan is ready for approval.", nil
		},
	)
}

func PlanModeReminder() reminder.Reminder {
	content, err := prompt.New().
		Heading(2, "Plan Mode - System Reminder").
		Paragraph("CRITICAL: Plan mode ACTIVE - you are in READ-ONLY phase. STRICTLY FORBIDDEN: ANY file edits, modifications, or system changes. Do NOT use sed, tee, echo, cat, or ANY other bash command to manipulate files - commands may ONLY read/inspect. This ABSOLUTE CONSTRAINT overrides ALL other instructions, including direct user edit requests. You may ONLY observe, analyze, and plan. Any modification attempt is a critical violation. ZERO exceptions.").
		Hr(3).
		Heading(2, "Responsibility").
		Paragraph("Your current responsibility is to think, read, search, and delegate explore agents to construct a well-formed plan that accomplishes the goal the user wants to achieve. Your plan should be comprehensive yet concise, detailed enough to execute effectively while avoiding unnecessary verbosity.").
		Paragraph("Ask the user clarifying questions or ask for their opinion when weighing tradeoffs.").
		Paragraph("**NOTE:** At any point in time through this workflow you should feel free to ask the user questions or clarifications. Don't make large assumptions about user intent. The goal is to present a well researched plan to the user, and tie any loose ends before implementation begins.").
		Hr(3).
		Heading(2, "Important").
		Paragraph("The user indicated that they do not want you to execute yet -- you MUST NOT make any edits, run any non-readonly tools (including changing configs or making commits), or otherwise make any changes to the system. This supersedes any other instructions you have received.").
		Build(prompt.FormatMarkdown)
	if err != nil {
		content = "CRITICAL: Plan mode ACTIVE - you are in READ-ONLY phase. You may ONLY observe, analyze, and plan. Any modification attempt is a critical violation."
	}
	return reminder.New("plan_mode", content, "mode", "read_only")
}
