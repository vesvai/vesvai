package components

import (
	"image"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func newTestTextarea() (*Textarea, tcell.SimulationScreen) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	s.SetSize(60, 10)
	ta := NewTextarea()
	ta.Layout(image.Rect(2, 1, 58, 9))
	ta.SetFocused(true)
	return ta, s
}

func renderTextarea(ta *Textarea, s tcell.SimulationScreen) {
	ta.Render(s, tui.DefaultDark())
	s.Show()
}

func typeRunes(ta *Textarea, text string) {
	for _, r := range text {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
}

func press(ta *Textarea, key tcell.Key, mods tcell.ModMask) {
	ta.HandleEvent(tcell.NewEventKey(key, 0, mods))
}

func cellAt(s tcell.SimulationScreen, x, y int) (rune, tcell.Color) {
	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	c := cells[y*w+x]
	_, bg, _ := c.Style.Decompose()
	if c.Runes == nil {
		return ' ', bg
	}
	return c.Runes[0], bg
}

func TestTextareaDesiredHeightGrowsFrom3To6(t *testing.T) {
	ta := NewTextarea()
	if h := ta.DesiredHeight(); h != 5 {
		t.Fatalf("initial height = %d, want 5", h)
	}
	ta.lines = [][]rune{{'a'}, {'b'}}
	if h := ta.DesiredHeight(); h != 5 {
		t.Fatalf("2-line height = %d, want 5", h)
	}
	ta.lines = [][]rune{{'a'}, {'b'}, {'c'}}
	if h := ta.DesiredHeight(); h != 5 {
		t.Fatalf("3-line height = %d, want 5", h)
	}
	ta.lines = [][]rune{{'a'}, {'b'}, {'c'}, {'d'}}
	if h := ta.DesiredHeight(); h != 6 {
		t.Fatalf("4-line height = %d, want 6", h)
	}
	ta.lines = make([][]rune, 7)
	if h := ta.DesiredHeight(); h != 8 {
		t.Fatalf("7-line height = %d, want 8", h)
	}
}

func TestShiftSelectionHighlightsCells(t *testing.T) {
	ta, s := newTestTextarea()
	typeRunes(ta, "hello world")

	press(ta, tcell.KeyHome, tcell.ModNone)
	for i := 0; i < 3; i++ {
		press(ta, tcell.KeyRight, tcell.ModNone)
	}
	for i := 0; i < 5; i++ {
		press(ta, tcell.KeyRight, tcell.ModShift)
	}
	if !ta.hasSel {
		t.Fatal("selection not active after Shift+Right")
	}
	if got := ta.selectedText(); got != "lo wo" {
		t.Fatalf("selected text = %q, want %q", got, "lo wo")
	}

	renderTextarea(ta, s)
	sel := tui.DefaultDark().Selection
	for x := 8; x <= 12; x++ {
		_, bg := cellAt(s, x, 2)
		if bg != sel {
			t.Fatalf("cell x=%d not highlighted (bg=%v)", x, bg)
		}
	}
	_, bg := cellAt(s, 7, 2)
	if bg == sel {
		t.Fatal("cell before selection incorrectly highlighted")
	}
}

func TestTypingReplacesSelection(t *testing.T) {
	ta, _ := newTestTextarea()
	typeRunes(ta, "hello world")
	press(ta, tcell.KeyHome, tcell.ModNone)
	for i := 0; i < 5; i++ {
		press(ta, tcell.KeyRight, tcell.ModShift)
	}
	typeRunes(ta, "hi")
	if got := ta.text(); got != "hi world" {
		t.Fatalf("text after replace = %q, want %q", got, "hi world")
	}
	if ta.hasSel {
		t.Fatal("selection should be cleared after typing")
	}
}

func TestBackspaceDeletesSelection(t *testing.T) {
	ta, _ := newTestTextarea()
	typeRunes(ta, "abcdef")
	press(ta, tcell.KeyLeft, tcell.ModNone)
	for i := 0; i < 3; i++ {
		press(ta, tcell.KeyLeft, tcell.ModShift)
	}
	press(ta, tcell.KeyBackspace, tcell.ModNone)
	if got := ta.text(); got != "abf" {
		t.Fatalf("text after backspace = %q, want %q", got, "abf")
	}
	if ta.row != 0 || ta.col != 2 {
		t.Fatalf("cursor = (%d,%d), want (0,2)", ta.row, ta.col)
	}
}

func TestCopyPasteRoundtrip(t *testing.T) {
	ta, _ := newTestTextarea()
	typeRunes(ta, "alpha beta")
	press(ta, tcell.KeyHome, tcell.ModNone)
	for i := 0; i < 5; i++ {
		press(ta, tcell.KeyRight, tcell.ModShift)
	}
	press(ta, tcell.KeyInsert, tcell.ModCtrl)
	if ta.clipboard != "alpha" {
		t.Fatalf("clipboard = %q, want %q", ta.clipboard, "alpha")
	}
	press(ta, tcell.KeyEnd, tcell.ModNone)
	press(ta, tcell.KeyInsert, tcell.ModShift)
	if got := ta.text(); got != "alpha betaalpha" {
		t.Fatalf("text after paste = %q", got)
	}
}

func TestCutRemovesSelection(t *testing.T) {
	ta, _ := newTestTextarea()
	typeRunes(ta, "one two three")
	press(ta, tcell.KeyHome, tcell.ModNone)
	for i := 0; i < 3; i++ {
		press(ta, tcell.KeyRight, tcell.ModShift)
	}
	press(ta, tcell.KeyDelete, tcell.ModShift)
	if ta.clipboard != "one" {
		t.Fatalf("clipboard = %q, want %q", ta.clipboard, "one")
	}
	if got := ta.text(); got != " two three" {
		t.Fatalf("text after cut = %q", got)
	}
}

func TestMultiLineSelectionAcrossLines(t *testing.T) {
	ta, _ := newTestTextarea()
	typeRunes(ta, "line one")
	press(ta, tcell.KeyEnter, tcell.ModShift)
	typeRunes(ta, "line two")

	press(ta, tcell.KeyHome, tcell.ModNone)
	press(ta, tcell.KeyUp, tcell.ModNone)
	for i := 0; i < 5; i++ {
		press(ta, tcell.KeyRight, tcell.ModShift)
	}
	press(ta, tcell.KeyDown, tcell.ModShift)
	got := ta.selectedText()
	if !strings.HasPrefix(got, "line ") || !strings.Contains(got, "\n") {
		t.Fatalf("multi-line selection = %q", got)
	}
	press(ta, tcell.KeyBackspace, tcell.ModNone)
	if ta.text() != "two" {
		t.Fatalf("text after multi-line delete = %q, want %q", ta.text(), "two")
	}
}

func TestEscapeClearsSelection(t *testing.T) {
	ta, _ := newTestTextarea()
	typeRunes(ta, "abcdef")
	press(ta, tcell.KeyRight, tcell.ModShift)
	if !ta.hasSel {
		t.Fatal("selection should be active")
	}
	press(ta, tcell.KeyEsc, tcell.ModNone)
	if ta.hasSel {
		t.Fatal("selection should be cleared by Escape")
	}
}

func TestPlainMovementClearsSelection(t *testing.T) {
	ta, _ := newTestTextarea()
	typeRunes(ta, "abcdef")
	press(ta, tcell.KeyRight, tcell.ModShift)
	if !ta.hasSel {
		t.Fatal("selection should be active")
	}
	press(ta, tcell.KeyRight, tcell.ModNone)
	if ta.hasSel {
		t.Fatal("plain movement must clear selection")
	}
}

func TestUpDownMovesAcrossWrappedRows(t *testing.T) {
	ta, _ := newTestTextarea()
	ta.Layout(image.Rect(2, 1, 16, 9))
	typeRunes(ta, "aaaa bbbb cccc")

	press(ta, tcell.KeyEnd, tcell.ModNone)
	if ta.row != 0 || ta.col != len(ta.lines[0]) {
		t.Fatalf("cursor after End = (%d,%d), want bottom of line 0", ta.row, ta.col)
	}
	press(ta, tcell.KeyUp, tcell.ModNone)
	if ta.row != 0 {
		t.Fatalf("Up should stay on line 0 (wrap), got row %d", ta.row)
	}
	if ta.col == len(ta.lines[0]) {
		t.Fatal("Up did not move the cursor")
	}
	press(ta, tcell.KeyDown, tcell.ModNone)
	if ta.row != 0 || ta.col != len(ta.lines[0]) {
		t.Fatalf("Down did not restore bottom of line 0: (%d,%d)", ta.row, ta.col)
	}

	press(ta, tcell.KeyHome, tcell.ModNone)
	press(ta, tcell.KeyUp, tcell.ModNone)
	if ta.row != 0 || ta.col != 0 {
		t.Fatalf("Up at top moved cursor to (%d,%d)", ta.row, ta.col)
	}
	if len(ta.history) != 0 {
		t.Fatal("Up must not trigger history")
	}
}

func TestUpDownNavigatesLogicalLines(t *testing.T) {
	ta, _ := newTestTextarea()
	typeRunes(ta, "first")
	press(ta, tcell.KeyEnter, tcell.ModShift)
	typeRunes(ta, "second")

	press(ta, tcell.KeyHome, tcell.ModNone)
	press(ta, tcell.KeyUp, tcell.ModNone)
	if ta.row != 0 {
		t.Fatalf("Up should move to line 0, got row %d", ta.row)
	}
	if ta.col > len(ta.lines[0]) {
		t.Fatalf("col %d out of range for line 0", ta.col)
	}
	press(ta, tcell.KeyDown, tcell.ModNone)
	if ta.row != 1 {
		t.Fatalf("Down should move to line 1, got row %d", ta.row)
	}
}
