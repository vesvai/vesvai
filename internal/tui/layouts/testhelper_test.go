package layouts

import (
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/components"
)

var testSessions = []components.Session{
	{ID: "s1", Title: "Refactor the TUI layout", Updated: "2m ago"},
	{ID: "s2", Title: "Add shift selection to textarea", Updated: "18m ago"},
	{ID: "s3", Title: "Design the command palette", Updated: "1h ago"},
	{ID: "s4", Title: "Fix resize wipe in main layout", Updated: "3h ago"},
	{ID: "s5", Title: "Session persistence backend", Updated: "yesterday"},
}

var testModels = []tui.ModelInfo{
	{Name: "deepseek-v4", Provider: "DeepSeek", Effort: "max", ContextWindow: 1_000_000},
	{Name: "Opus 4.5", Provider: "Anthropic", Effort: "max", ContextWindow: 200_000},
	{Name: "claude-3.5-haiku", Provider: "Anthropic", Effort: "medium", ContextWindow: 200_000},
	{Name: "gemini-2.5-flash", Provider: "Google", Effort: "low", ContextWindow: 1_000_000},
	{Name: "gpt-4o-mini", Provider: "OpenAI", Effort: "medium", ContextWindow: 128_000},
}

func wirePalette(l *MainLayout) {
	l.SetSessions(testSessions)
	l.SetModels(testModels)
	l.OnAction = func(id string) {
		switch id {
		case "switch-session":
			l.OpenSessionPicker()
		case "change-model":
			l.OpenModelPicker()
		}
	}
}
