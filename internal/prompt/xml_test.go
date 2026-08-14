package prompt

import (
	"context"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/skill"
)

type fakeTool struct {
	name   string
	desc   string
	schema map[string]any
}

func (t *fakeTool) Name() string        { return t.name }
func (t *fakeTool) Description() string { return t.desc }
func (t *fakeTool) Schema() map[string]any {
	if t.schema != nil {
		return t.schema
	}
	return map[string]any{}
}
func (t *fakeTool) Handle(ctx context.Context, params map[string]any) (string, error) {
	return "", nil
}

func testHooks(t *testing.T, tools []agent.Tool, skills []skill.Skill) {
	t.Helper()
	hs := hook.New(nil)
	hs.AddFilter(hook.HookToolsCollect, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		existing, _ := value.([]agent.Tool)
		return append(existing, tools...)
	}, 50)
	hs.AddFilter(hook.HookSkillsCollect, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		existing, _ := value.([]skill.Skill)
		return append(existing, skills...)
	}, 50)
	SetHooks(hs)
	t.Cleanup(func() { SetHooks(nil) })
}

func TestToolsXML(t *testing.T) {
	testHooks(t, []agent.Tool{
		&fakeTool{
			name: "read",
			desc: "Read a file from the workspace",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "File path to read",
					},
				},
				"required": []any{"path"},
			},
		},
		&fakeTool{name: "bash", desc: "Run a & shell command"},
	}, nil)

	out := New().Role("You are a helper.").Tools().Build()

	for _, want := range []string{
		"<tools>",
		`<tool name="read">`,
		"Read a file from the workspace",
		`<parameter name="path" type="string" required="true" description="File path to read" />`,
		`<tool name="bash">`,
		"Run a &amp; shell command",
		"</tools>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("tools XML missing %q:\n%s", want, out)
		}
	}
}

func TestSkillsXML(t *testing.T) {
	testHooks(t, nil, []skill.Skill{
		{Name: "graphify", Description: "Turn any input into a knowledge graph"},
		{Name: "impeccable", Description: "Polish the interface"},
	})

	out := New().Role("You are a helper.").Skills().Build()

	for _, want := range []string{
		"<skills>",
		`<skill name="graphify">`,
		"Turn any input into a knowledge graph",
		`<skill name="impeccable">`,
		"</skills>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("skills XML missing %q:\n%s", want, out)
		}
	}
}

func TestToolsSkillsChained(t *testing.T) {
	testHooks(t,
		[]agent.Tool{&fakeTool{name: "glob", desc: "Find files by pattern"}},
		[]skill.Skill{{Name: "graphify", Description: "Knowledge graphs"}},
	)

	out := New().Role("You are a helper.").Tools().Skills().Build()

	if !strings.Contains(out, "<tools>") || !strings.Contains(out, "</tools>") {
		t.Fatalf("tools block missing:\n%s", out)
	}
	if !strings.Contains(out, "<skills>") || !strings.Contains(out, "</skills>") {
		t.Fatalf("skills block missing:\n%s", out)
	}
	if strings.Index(out, "<tools>") > strings.Index(out, "<skills>") {
		t.Fatalf("tools must precede skills:\n%s", out)
	}
}

func TestToolsItemsLegacy(t *testing.T) {
	out := New().Tools("read", "bash").Build()
	if !strings.Contains(out, "# Tools") || !strings.Contains(out, "- read") {
		t.Fatalf("legacy tools list missing:\n%s", out)
	}
}

func TestToolsNoHooks(t *testing.T) {
	SetHooks(nil)
	defer SetHooks(nil)

	out := New().Role("You are a helper.").Tools().Skills().Build()
	if strings.Contains(out, "<tools>") || strings.Contains(out, "<skills>") {
		t.Fatalf("unwired builder must not render XML:\n%s", out)
	}
	if !strings.Contains(out, "You are a helper.") {
		t.Fatalf("role missing:\n%s", out)
	}
}

func TestRenderToolsXMLEmpty(t *testing.T) {
	if got := RenderToolsXML(nil); got != "" {
		t.Fatalf("RenderToolsXML(nil) = %q, want empty", got)
	}
	if got := RenderSkillsXML(nil); got != "" {
		t.Fatalf("RenderSkillsXML(nil) = %q, want empty", got)
	}
}
