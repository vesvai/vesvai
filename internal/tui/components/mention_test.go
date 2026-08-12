package components

import (
	"image"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func newMentionTestTextarea() (*Textarea, tcell.SimulationScreen) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	s.SetSize(60, 20)
	ta := NewTextarea()
	ta.SetMentionCatalog([]Mention{
		{ID: "planner", Kind: "agent", Label: "planner"},
		{ID: "explorer", Kind: "agent", Label: "explorer"},
		{ID: "internal", Kind: "dir", Label: "internal"},
		{ID: "go.mod", Kind: "file", Label: "go.mod"},
	})
	ta.Layout(image.Rect(2, 13, 58, 19))
	ta.SetFocused(true)
	return ta, s
}

func TestMentionPickerOpensOnAtAndEscReverts(t *testing.T) {
	ta, s := newMentionTestTextarea()
	typeRunes(ta, "say @")
	if !ta.pickerOpen {
		t.Fatal("picker should open after '@'")
	}
	renderTextarea(ta, s)
	cells, _, _ := s.GetContents()
	var sb strings.Builder
	for i := 0; i < len(cells); i++ {
		c := cells[i]
		if c.Runes != nil && len(c.Runes) > 0 {
			sb.WriteRune(c.Runes[0])
		}
	}
	frame := sb.String()
	for _, want := range []string{"planner", "explorer", "internal", "go.mod"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("picker missing %q:\n%s", want, frame)
		}
	}

	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone))
	if ta.pickerOpen {
		t.Fatal("picker should close on Esc")
	}
	if ta.text() != "say " {
		t.Fatalf("text = %q, want %q", ta.text(), "say ")
	}
}

func TestMentionPickerFiltersAndSelects(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	typeRunes(ta, "@plan")
	if !ta.pickerOpen {
		t.Fatal("picker should be open")
	}
	if len(ta.pickerResults) != 1 || ta.pickerResults[0].ID != "planner" {
		t.Fatalf("results = %+v, want only planner", ta.pickerResults)
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if ta.pickerOpen {
		t.Fatal("picker should close after selection")
	}
	if ta.text() != "@planner" {
		t.Fatalf("text = %q, want %q", ta.text(), "@planner")
	}
	if ta.col != len("@planner") {
		t.Fatalf("cursor = %d, want after the mention", ta.col)
	}
}

func TestMentionPickerArrowNavigation(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	typeRunes(ta, "@e")
	if len(ta.pickerResults) == 0 {
		t.Fatal("expected results")
	}
	first := ta.pickerResults[ta.pickerCursor].ID
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	if ta.pickerResults[ta.pickerCursor].ID == first {
		t.Fatal("Down should move the picker cursor")
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	if ta.pickerResults[ta.pickerCursor].ID != first {
		t.Fatal("Up should move back to the first result")
	}
}

func TestMentionBackspaceDeletesWholeBlock(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	typeRunes(ta, "use @")
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'p', tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'n', tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if ta.text() != "use @planner" {
		t.Fatalf("text = %q", ta.text())
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone))
	if ta.text() != "use " {
		t.Fatalf("text after backspace = %q, want %q", ta.text(), "use ")
	}
	if ta.col != len("use ") {
		t.Fatalf("cursor = %d, want %d", ta.col, len("use "))
	}
}

func TestMentionDeleteForwardWholeBlock(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	typeRunes(ta, "@")
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'p', tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyDelete, 0, tcell.ModNone))
	if ta.text() != "" {
		t.Fatalf("text after delete-forward = %q, want empty", ta.text())
	}
}

func TestMentionCursorSkipsOverBlock(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	typeRunes(ta, "x @planner")
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if ta.col != 2 {
		t.Fatalf("cursor after one Left = %d, want 2 (before the block)", ta.col)
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if ta.col != 1 {
		t.Fatalf("cursor after second Left = %d, want 1", ta.col)
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if ta.col != 2 {
		t.Fatalf("cursor after first Right = %d, want 2 (block start)", ta.col)
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if ta.col != 10 {
		t.Fatalf("cursor after second Right = %d, want 10 (after the block)", ta.col)
	}
}

func TestMentionRendersGold(t *testing.T) {
	ta, s := newMentionTestTextarea()
	typeRunes(ta, "@planner")
	renderTextarea(ta, s)

	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	pal := tui.DefaultDark()
	found := false
	for i := 0; i < len(cells); i++ {
		c := cells[i]
		if c.Runes != nil && len(c.Runes) > 0 && c.Runes[0] == '@' {
			fg, _, _ := c.Style.Decompose()
			if fg == pal.Mention {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("mention '@' cell is not gold")
	}
	_ = w
}

func TestMentionSpaceClosesPickerKeepsText(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	typeRunes(ta, "@pla")
	if !ta.pickerOpen {
		t.Fatal("picker should be open")
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone))
	if ta.pickerOpen {
		t.Fatal("picker should close on a space")
	}
	if ta.text() != "@pla " {
		t.Fatalf("text = %q, want %q", ta.text(), "@pla ")
	}
}

func TestMentionPickerDisappearsAfterSelect(t *testing.T) {
	ta, s := newMentionTestTextarea()
	typeRunes(ta, "@")
	renderTextarea(ta, s)

	boxChars := func() bool {
		cells, _, _ := s.GetContents()
		w, _ := s.Size()
		for i := 0; i < len(cells); i++ {
			c := cells[i]
			if c.Runes == nil || len(c.Runes) == 0 {
				continue
			}
			switch c.Runes[0] {
			case '┌', '┐', '└', '┘', '│':
				y := i / w
				if y < 13 {
					return true
				}
			}
		}
		return false
	}
	if !boxChars() {
		t.Fatal("picker box should be visible while open")
	}

	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if ta.pickerOpen {
		t.Fatal("picker should close after selection")
	}
	renderTextarea(ta, s)
	if boxChars() {
		t.Fatal("ghost picker box still on screen after selection")
	}
}

func TestMentionPickerClearsWhenSearchEmpties(t *testing.T) {
	ta, s := newMentionTestTextarea()
	typeRunes(ta, "@")
	renderTextarea(ta, s)

	for _, r := range "zzz" {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	renderTextarea(ta, s)
	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	for i := 0; i < len(cells); i++ {
		c := cells[i]
		if c.Runes != nil && len(c.Runes) > 0 && c.Runes[0] == '┌' && i/w < 13 {
			t.Fatal("ghost picker box still shown for an empty search")
		}
	}
}

func TestMentionPickerShrinkingLeavesNoGhost(t *testing.T) {
	ta, s := newMentionTestTextarea()
	typeRunes(ta, "@")
	renderTextarea(ta, s)

	boxRows := func() int {
		cells, _, _ := s.GetContents()
		w, _ := s.Size()
		rows := 0
		for i := 0; i < len(cells); i++ {
			c := cells[i]
			if c.Runes != nil && len(c.Runes) > 0 && c.Runes[0] == '│' && i/w < 13 {
				rows++
			}
		}
		return rows
	}
	if n := boxRows(); n < 4 {
		t.Fatalf("expected a tall picker box, got %d border cells", n)
	}

	for _, r := range "plan" {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	renderTextarea(ta, s)
	if !ta.pickerOpen {
		t.Fatal("picker should still be open")
	}

	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	hasBoxGlyph := func(y int) bool {
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if c.Runes == nil || len(c.Runes) == 0 {
				continue
			}
			switch c.Runes[0] {
			case '┌', '┐', '└', '┘', '│':
				return true
			}
		}
		return false
	}
	for y := 6; y <= 9; y++ {
		if hasBoxGlyph(y) {
			t.Fatalf("ghost picker rows left above the shrunk box at row %d", y)
		}
	}
	if !hasBoxGlyph(10) {
		t.Fatal("shrunk picker box missing")
	}
}
