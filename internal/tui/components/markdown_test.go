package components

import (
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/tui"
)

func render(t *testing.T, md string, width int) []tui.Line {
	t.Helper()
	r := NewMarkdownRenderer()
	return r.Render(md, width, DefaultTestPalette())
}

func DefaultTestPalette() *tui.Palette {
	p := tui.DefaultDark()
	return p
}

func TestRenderParagraphAndHeading(t *testing.T) {
	lines := render(t, "# Title\n\nSome **bold** text with `code` here.\n", 60)
	if len(lines) < 3 {
		t.Fatalf("got %d lines: %v", len(lines), lines)
	}
	first := textOfLine(lines[0])
	if !strings.HasPrefix(first, "▍ Title") {
		t.Fatalf("heading line = %q", first)
	}
	joined := joinLines(lines)
	if !strings.Contains(joined, "bold") || !strings.Contains(joined, "code") {
		t.Fatalf("missing content: %q", joined)
	}
}

func TestRenderCodeBlockBoxAndSyntax(t *testing.T) {
	code := "```go\nfunc main() {\n\tfmt.Println(\"hi\") // comment\n}\n```\n"
	lines := render(t, code, 50)
	if len(lines) < 5 {
		t.Fatalf("too few lines for a code box: %d", len(lines))
	}
	top := textOfLine(lines[0])
	if !strings.HasPrefix(top, "┌─go") {
		t.Fatalf("code box top = %q", top)
	}
	if !strings.HasSuffix(top, "┐") {
		t.Fatalf("code box top missing corner: %q", top)
	}
	last := textOfLine(lines[len(lines)-1])
	if !strings.HasPrefix(last, "└") || !strings.HasSuffix(last, "┘") {
		t.Fatalf("code box bottom = %q", last)
	}
	body := joinLines(lines)
	if !strings.Contains(body, "func main") || !strings.Contains(body, "comment") {
		t.Fatalf("code content missing: %q", body)
	}
	for i, ln := range lines {
		if ln.Width() > 50 {
			t.Fatalf("line %d exceeds width: %d", i, ln.Width())
		}
	}
}

func TestRenderListMarkers(t *testing.T) {
	lines := render(t, "- one\n- two\n", 40)
	joined := joinLines(lines)
	if !strings.Contains(joined, "• one") || !strings.Contains(joined, "• two") {
		t.Fatalf("list markers missing: %q", joined)
	}

	lines = render(t, "1. first\n2. second\n", 40)
	joined = joinLines(lines)
	if !strings.Contains(joined, "1. first") || !strings.Contains(joined, "2. second") {
		t.Fatalf("ordered markers missing: %q", joined)
	}
}

func TestRenderBlockquote(t *testing.T) {
	lines := render(t, "> quoted wisdom\n", 40)
	joined := joinLines(lines)
	if !strings.Contains(joined, "│") || !strings.Contains(joined, "quoted wisdom") {
		t.Fatalf("blockquote missing: %q", joined)
	}
}

func TestRenderTable(t *testing.T) {
	md := "| A | B |\n|---|---|\n| 1 | 2 |\n"
	lines := render(t, md, 40)
	joined := joinLines(lines)
	for _, want := range []string{"A", "B", "1", "2"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("table missing %q: %q", want, joined)
		}
	}
}

func TestRenderTableWideCellClamps(t *testing.T) {
	long := "https://example.com/very/long/path/that/exceeds/any/column/width"
	md := "| File | URL |\n|---|---|\n| main.go | " + long + " |\n"
	lines := render(t, md, 24)
	for i, ln := range lines {
		if ln.Width() > 24 {
			t.Fatalf("line %d width %d > 24: %q", i, ln.Width(), joinLines([]tui.Line{ln}))
		}
	}
	joined := joinLines(lines)
	if !strings.Contains(joined, "…") {
		t.Fatalf("wide cell should be ellipsized: %q", joined)
	}
}

func TestTruncateCell(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"short", 20, "short"},
		{"0123456789", 5, "0123…"},
		{"0123456789", 4, "012…"},
		{"0123456789", 2, "0…"},
		{"", 5, ""},
		{"hello", 0, ""},
	}
	for _, c := range cases {
		if got := truncateCell(c.in, c.w); got != c.want {
			t.Fatalf("truncateCell(%q, %d) = %q, want %q", c.in, c.w, got, c.want)
		}
	}
}

func TestRenderWrapsToWidth(t *testing.T) {
	long := strings.Repeat("word ", 40)
	lines := render(t, long, 30)
	for i, ln := range lines {
		if ln.Width() > 30 {
			t.Fatalf("line %d width %d > 30", i, ln.Width())
		}
	}
}

func TestRenderEmptyAndUnfenced(t *testing.T) {
	if got := render(t, "", 30); len(got) != 0 {
		t.Fatalf("empty markdown produced %d lines", len(got))
	}
	lines := render(t, "plain", 30)
	if joined := joinLines(lines); !strings.Contains(joined, "plain") {
		t.Fatalf("plain text missing: %q", joined)
	}
}

func textOfLine(l tui.Line) string {
	var sb strings.Builder
	for _, c := range l {
		sb.WriteRune(c.R)
	}
	return sb.String()
}

func joinLines(lines []tui.Line) string {
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(textOfLine(l))
		sb.WriteByte('\n')
	}
	return sb.String()
}
