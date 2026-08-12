package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestWrapSegmentsBreaksOnSpace(t *testing.T) {
	lines := WrapSegments([]Segment{{Text: "hello world foo", Style: tcell.StyleDefault}}, 8)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %v", len(lines), lineStrings(lines))
	}
	if w := lines[0].Width(); w > 8 {
		t.Fatalf("line 0 width %d exceeds 8", w)
	}
	if w := lines[1].Width(); w > 8 {
		t.Fatalf("line 1 width %d exceeds 8", w)
	}
	if got := textOf(lines[0]); got != "hello wo" {
		t.Fatalf("line 0 = %q, want %q", got, "hello wo")
	}
	if got := textOf(lines[1]); got != "rld foo" {
		t.Fatalf("line 1 = %q, want %q", got, "rld foo")
	}
}

func TestWrapSegmentsBreaksLongWord(t *testing.T) {
	lines := WrapSegments([]Segment{{Text: "supercalifragilistic", Style: tcell.StyleDefault}}, 6)
	if len(lines) < 2 {
		t.Fatal("long word must wrap")
	}
	for i, ln := range lines {
		if ln.Width() > 6 {
			t.Fatalf("line %d width %d exceeds 6", i, ln.Width())
		}
	}
}

func TestWrapSegmentsWideRunesDoNotStraddleEdge(t *testing.T) {
	lines := WrapSegments([]Segment{{Text: "你好世界", Style: tcell.StyleDefault}}, 4)
	for _, ln := range lines {
		if ln.Width() > 4 {
			t.Fatalf("line width %d exceeds 4", ln.Width())
		}
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
}

func TestWrapSegmentsHardBreak(t *testing.T) {
	lines := WrapSegments([]Segment{{Text: "a\nb", Style: tcell.StyleDefault}}, 20)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if textOf(lines[0]) != "a" || textOf(lines[1]) != "b" {
		t.Fatalf("unexpected lines: %v", lineStrings(lines))
	}
}

func TestTruncateAddsEllipsis(t *testing.T) {
	line := WrapText("abcdefghij", tcell.StyleDefault, 20)[0]
	short := Truncate(line, 5)
	if short.Width() > 5 {
		t.Fatalf("truncated width %d exceeds 5", short.Width())
	}
	last := short[len(short)-1].R
	if last != '…' {
		t.Fatalf("last rune = %q, want …", last)
	}
}

func TestDisplayWidth(t *testing.T) {
	if w := DisplayWidth("hello"); w != 5 {
		t.Fatalf("width = %d, want 5", w)
	}
	if w := DisplayWidth("你好"); w != 4 {
		t.Fatalf("width = %d, want 4", w)
	}
}

func TestClampScroll(t *testing.T) {
	if got := ClampScroll(-3, 10); got != 0 {
		t.Fatalf("negative clamp = %d", got)
	}
	if got := ClampScroll(15, 10); got != 10 {
		t.Fatalf("overflow clamp = %d", got)
	}
	if got := ClampScroll(5, 10); got != 5 {
		t.Fatalf("middle clamp = %d", got)
	}
}

func textOf(l Line) string {
	s := make([]rune, 0, len(l))
	for _, c := range l {
		s = append(s, c.R)
	}
	return string(s)
}

func lineStrings(lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = textOf(l)
	}
	return out
}
