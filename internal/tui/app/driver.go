package app

import (
	"context"
	"time"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/tui"
)

type RunRequest struct {
	Text        string
	Attachments []*tui.Attachment
	History     []llm.Message
}

type Driver interface {
	Run(ctx context.Context, req RunRequest, emit func(tui.StreamEvent))
	Cancel()
}

type DemoDriver struct {
	cancel context.CancelFunc
}

func NewDemoDriver() *DemoDriver { return &DemoDriver{} }

func (d *DemoDriver) Cancel() {
	if d.cancel != nil {
		d.cancel()
	}
}

func (d *DemoDriver) Run(ctx context.Context, req RunRequest, emit func(tui.StreamEvent)) {
	input := req.Text
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	defer func() { d.cancel = nil }()

	emit(tui.StreamEvent{Kind: tui.EventStart})

	reason := func(text string, wait time.Duration) bool {
		emit(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: text + " "})
		return sleep(ctx, wait)
	}

	if !reason("Let me analyze this request carefully: "+input+" ", 500*time.Millisecond) {
		return
	}
	if !reason("I'll break it into small, verifiable steps.", 450*time.Millisecond) {
		return
	}
	if !reason("Then I'll delegate the heavy lifting to the right tools.", 400*time.Millisecond) {
		return
	}

	emit(tui.StreamEvent{
		Kind: tui.EventToolCall,
		ToolCall: &tui.ToolCallInfo{
			ToolName: "bash",
			Args:     map[string]any{"command": "ls -la"},
		},
	})
	if !sleep(ctx, 1600*time.Millisecond) {
		return
	}
	emit(tui.StreamEvent{
		Kind: tui.EventToolResult,
		ToolResult: &tui.ToolResultInfo{
			ToolName: "bash",
			Result:   "drwxrwxr-x  obuntu obuntu 4096 Aug 12 05:06 .\n-rw-rw-r--  obuntu obuntu  219 Jul 31 16:28 .gitignore\n-rw-rw-r--  obuntu obuntu 1049 Jul 27 14:49 go.mod\ndrwxrwxr-x  22 obuntu obuntu 4096 Aug 12 05:06 internal",
			Duration: 1600 * time.Millisecond,
		},
	})

	if !reason("The tree is small. Now I'll check the relevant file.", 450*time.Millisecond) {
		return
	}

	emit(tui.StreamEvent{
		Kind: tui.EventToolCall,
		ToolCall: &tui.ToolCallInfo{
			ToolName: "read",
			Args:     map[string]any{"path": "internal/tui/style.go"},
		},
	})
	if !sleep(ctx, 800*time.Millisecond) {
		return
	}
	emit(tui.StreamEvent{
		Kind: tui.EventToolResult,
		ToolResult: &tui.ToolResultInfo{
			ToolName: "read",
			Result:   "// Palette holds every semantic color the TUI uses.\ntype Palette struct {\n\tBackground tcell.Color\n\tAccent     tcell.Color\n}",
			Duration: 800 * time.Millisecond,
		},
	})

	if !sleep(ctx, 200*time.Millisecond) {
		return
	}

	stream := func(chunk string, wait time.Duration) bool {
		emit(tui.StreamEvent{Kind: tui.EventContent, Content: chunk})
		return sleep(ctx, wait)
	}

	if !stream("## Plan\n", 150*time.Millisecond) {
		return
	}
	if !stream("Here is the approach I verified:\n\n", 300*time.Millisecond) {
		return
	}
	if !stream("- **Inspect** the codebase to confirm current state\n", 250*time.Millisecond) {
		return
	}
	if !stream("- **Implement** in small steps, keeping the module compile-safe\n\n", 250*time.Millisecond) {
		return
	}
	if !stream("The key change lives in ", 180*time.Millisecond) {
		return
	}
	if !stream("`internal/tui/style.go`", 120*time.Millisecond) {
		return
	}
	if !stream(" — a single source of truth for colors:\n\n", 250*time.Millisecond) {
		return
	}
	code := "```go\nfunc DetectPalette(s tcell.Screen) *Palette {\n\tif s.Colors() >= 1<<24 {\n\t\treturn DefaultDark() // truecolor: RGB palette\n\t}\n\treturn FallbackDark() // 16/256-color terminals\n}\n```\n\n"
	if !stream(code, 120*time.Millisecond) {
		return
	}
	if !stream("| Step | Owner | State |\n|---|---|---|\n| Inspect | explorer | done |\n| Edit | orchestrator | running |\n\n", 200*time.Millisecond) {
		return
	}
	if !stream("> Verification: the module builds clean with `go build ./...`.\n\n", 200*time.Millisecond) {
		return
	}

	if !stream("I'll delegate this to specialist agents.\n\n", 250*time.Millisecond) {
		return
	}

	const planner = "planner"
	emit(tui.StreamEvent{
		Kind: tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{
			Name:   planner,
			Prompt: "Break this change into small, verifiable steps.",
		},
	})
	if !sleep(ctx, 250*time.Millisecond) {
		return
	}
	sub := func(id, chunk string, wait time.Duration) bool {
		emit(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: id, Content: chunk})
		return sleep(ctx, wait)
	}
	if !sub(planner, "Here are the steps I recommend:\n", 350*time.Millisecond) {
		return
	}
	if !sub(planner, "1. Verify the current layout wiring\n", 250*time.Millisecond) {
		return
	}
	if !sub(planner, "2. Confirm all widgets render at the new bounds\n", 250*time.Millisecond) {
		return
	}
	if !sub(planner, "3. Run the test suite as a final gate\n", 250*time.Millisecond) {
		return
	}
	emit(tui.StreamEvent{
		Kind:       tui.EventSubagentDone,
		SubagentID: planner,
		SubagentResult: &tui.SubagentResult{
			Name:     planner,
			Result:   "3 steps outlined",
			Duration: 1800 * time.Millisecond,
		},
	})

	if !stream("Planner finished. Now I'll have the explorer verify the structure.\n\n", 250*time.Millisecond) {
		return
	}

	const explorer = "explorer"
	emit(tui.StreamEvent{
		Kind: tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{
			Name:   explorer,
			Prompt: "Inspect internal/tui and report how the layout is structured.",
		},
	})
	if !sleep(ctx, 300*time.Millisecond) {
		return
	}

	if !stream("While explorer inspects the codebase, I'll line up the change list.\n", 300*time.Millisecond) {
		return
	}

	if !sub(explorer, "Scanning the TUI package… ", 400*time.Millisecond) {
		return
	}
	if !sub(explorer, "The layout is composed of three layers:\n", 300*time.Millisecond) {
		return
	}
	if !sub(explorer, "- `layouts/main.go` wires viewport, textarea and statusbar\n", 250*time.Millisecond) {
		return
	}

	emit(tui.StreamEvent{
		Kind: tui.EventToolCall,
		ToolCall: &tui.ToolCallInfo{
			ToolName:   "read",
			SubagentID: explorer,
			Args:       map[string]any{"path": "internal/tui/layouts/main.go"},
		},
	})
	if !sleep(ctx, 700*time.Millisecond) {
		return
	}
	emit(tui.StreamEvent{
		Kind: tui.EventToolResult,
		ToolResult: &tui.ToolResultInfo{
			ToolName:   "read",
			SubagentID: explorer,
			Result:     "// MainLayout composes the viewport, textarea, statusbar and help bar.",
			Duration:   700 * time.Millisecond,
		},
	})
	if !sub(explorer, "- `components/` holds the widgets and renderers\n", 250*time.Millisecond) {
		return
	}
	if !sub(explorer, "- `style.go` owns the color palette\n", 250*time.Millisecond) {
		return
	}
	if !sub(explorer, "No layout issues found.", 200*time.Millisecond) {
		return
	}
	emit(tui.StreamEvent{
		Kind:       tui.EventSubagentDone,
		SubagentID: explorer,
		SubagentResult: &tui.SubagentResult{
			Name:     explorer,
			Result:   "inspected 3 packages, no issues",
			Duration: 3200 * time.Millisecond,
		},
	})

	if !stream("Explorer confirmed the structure. All checks passed — the task is complete.\n", 250*time.Millisecond) {
		return
	}

	emit(tui.StreamEvent{
		Kind: tui.EventDone,
		Usage: &tui.Usage{
			PromptTokens:     1240,
			CompletionTokens: 830,
			TotalTokens:      2070,
		},
	})
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}
