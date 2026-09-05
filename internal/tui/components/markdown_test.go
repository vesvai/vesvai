package components

import "testing"

func TestRenderMarkdownHeading(t *testing.T) {
	lines := RenderMarkdown("# Hello")
	if len(lines) != 1 || lines[0].Heading != 1 {
		t.Fatalf("expected one h1 line, got %+v", lines)
	}
	if lines[0].Text() != "Hello" {
		t.Errorf("text = %q, want Hello", lines[0].Text())
	}
}

func TestRenderMarkdownInline(t *testing.T) {
	lines := RenderMarkdown("this is **bold** and `code`")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	foundBold, foundCode := false, false
	for _, s := range lines[0].Segs {
		if s.Bold && s.Text == "bold" {
			foundBold = true
		}
		if s.Code && s.Text == "code" {
			foundCode = true
		}
	}
	if !foundBold || !foundCode {
		t.Errorf("segments missing: bold=%v code=%v segs=%+v", foundBold, foundCode, lines[0].Segs)
	}
}

func TestRenderMarkdownCodeBlock(t *testing.T) {
	src := "before\n```go\nfmt.Println(1)\n```\nafter"
	lines := RenderMarkdown(src)
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %+v", len(lines), lines)
	}
	if !lines[1].Code || !lines[2].Code || !lines[3].Code {
		t.Errorf("code block lines should be flagged code")
	}
	if lines[2].Text() != "fmt.Println(1)" {
		t.Errorf("code line = %q", lines[2].Text())
	}
}

func TestRenderMarkdownBulletAndQuote(t *testing.T) {
	lines := RenderMarkdown("- item")
	if !lines[0].Bullet {
		t.Error("expected bullet")
	}
	lines = RenderMarkdown("> note")
	if !lines[0].Quote {
		t.Error("expected quote")
	}
}

func TestRenderMarkdownHr(t *testing.T) {
	lines := RenderMarkdown("---")
	if len(lines) != 1 || !lines[0].Hr {
		t.Errorf("expected hr, got %+v", lines)
	}
}

func TestComputeDiff(t *testing.T) {
	hunks := ComputeDiff("line1\nold\nline3", "line1\nnew\nline3")
	if !HasDiff(hunks) {
		t.Fatal("expected diff")
	}
	var minus, plus string
	for _, hunk := range hunks {
		for _, l := range hunk.Lines {
			switch l.Kind {
			case '-':
				minus = l.Text
			case '+':
				plus = l.Text
			}
		}
	}
	if minus != "old" || plus != "new" {
		t.Errorf("minus=%q plus=%q, want old/new", minus, plus)
	}
}

func TestComputeDiffNoChange(t *testing.T) {
	if d := ComputeDiff("same\n", "same\n"); HasDiff(d) {
		t.Errorf("identical text should produce no diff, got %+v", d)
	}
}

func TestDiffHunkBounds(t *testing.T) {
	hunks := ComputeDiff("a", "b")
	if len(hunks) == 0 {
		t.Fatal("expected one hunk")
	}
	h := hunks[0]
	if h.Lines[0].Kind != '-' || h.Lines[1].Kind != '+' {
		t.Errorf("unexpected lines: %+v", h.Lines)
	}
}
