package components

import "testing"

func TestShiftRightSelectsChar(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftRight()
	in.ShiftRight()
	if !in.HasSelection() {
		t.Fatal("expected a selection")
	}
	start, end, ok := in.SelectionBounds()
	if !ok {
		t.Fatal("expected selection bounds")
	}
	if start != (textPos{0, 0}) || end != (textPos{0, 2}) {
		t.Errorf("bounds = %+v..%+v, want (0,0)..(0,2)", start, end)
	}
	if in.SelectedText() != "he" {
		t.Errorf("SelectedText = %q, want he", in.SelectedText())
	}
}

func TestShiftLeftSelectsBackward(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.ShiftLeft()
	in.ShiftLeft()
	if in.SelectedText() != "lo" {
		t.Errorf("SelectedText = %q, want lo", in.SelectedText())
	}
}

func TestSelectionContractsPastAnchor(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftRight()
	in.ShiftRight()
	in.ShiftLeft()
	if in.SelectedText() != "h" {
		t.Errorf("SelectedText = %q, want h", in.SelectedText())
	}
	in.ShiftLeft()
	if in.HasSelection() {
		t.Error("selection should collapse to nothing")
	}
}

func TestShiftUpDownSelectsLines(t *testing.T) {
	in := NewInput()
	for _, r := range "ab" {
		in.InsertRune(r)
	}
	in.Newline()
	for _, r := range "cd" {
		in.InsertRune(r)
	}
	in.Newline()
	for _, r := range "ef" {
		in.InsertRune(r)
	}
	in.ShiftUp()
	in.ShiftUp()
	if !in.HasSelection() {
		t.Fatal("expected selection")
	}
	if in.SelectedText() != "\ncd\nef" {
		t.Errorf("SelectedText = %q, want \\ncd\\nef", in.SelectedText())
	}
}

func TestShiftWordSelection(t *testing.T) {
	in := NewInput()
	for _, r := range "one two three" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftWordRight()
	if in.SelectedText() != "one " {
		t.Errorf("ShiftWordRight SelectedText = %q, want one ", in.SelectedText())
	}
	in.ShiftWordRight()
	if in.SelectedText() != "one two " {
		t.Errorf("ShiftWordRight twice SelectedText = %q, want one two ", in.SelectedText())
	}
}

func TestSelectAll(t *testing.T) {
	in := NewInput()
	for _, r := range "abc" {
		in.InsertRune(r)
	}
	in.Newline()
	for _, r := range "def" {
		in.InsertRune(r)
	}
	in.SelectAll()
	if in.SelectedText() != "abc\ndef" {
		t.Errorf("SelectedText = %q, want abc\\ndef", in.SelectedText())
	}
}

func TestTypingReplacesSelection(t *testing.T) {
	in := NewInput()
	for _, r := range "hello world" {
		in.InsertRune(r)
	}
	in.SelectAll()
	in.InsertRune('x')
	if in.Value() != "x" {
		t.Errorf("Value = %q, want x", in.Value())
	}
	if in.HasSelection() {
		t.Error("selection should be cleared after replace")
	}
}

func TestBackspaceDeletesSelection(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftRight()
	in.ShiftRight()
	in.Backspace()
	if in.Value() != "llo" {
		t.Errorf("after deleting selection Value = %q, want llo", in.Value())
	}
}

func TestBackspaceDeletesWordSelection(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftWordRight()
	in.Backspace()
	if in.Value() != "" {
		t.Errorf("after deleting word selection Value = %q, want empty", in.Value())
	}
}

func TestDeleteDeletesSelection(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftRight()
	in.Delete()
	if in.Value() != "ello" {
		t.Errorf("after deleting selection Value = %q, want ello", in.Value())
	}
}

func TestSelectAcrossLinesAndDelete(t *testing.T) {
	in := NewInput()
	for _, r := range "ab" {
		in.InsertRune(r)
	}
	in.Newline()
	for _, r := range "cd" {
		in.InsertRune(r)
	}
	in.SelectAll()
	if in.SelectedText() != "ab\ncd" {
		t.Errorf("SelectedText = %q, want ab\\ncd", in.SelectedText())
	}
	in.Backspace()
	if in.Value() != "" {
		t.Errorf("after deleting multi-line selection Value = %q, want empty", in.Value())
	}
	if len(in.Lines()) != 1 {
		t.Errorf("after delete lines = %d, want 1", len(in.Lines()))
	}
}

func TestKillDeletesSelection(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftRight()
	in.ShiftRight()
	in.KillToEnd()
	if in.Value() != "llo" {
		t.Errorf("KillToEnd with selection Value = %q, want llo", in.Value())
	}
	if in.killRegister != "he" {
		t.Errorf("killRegister = %q, want he", in.killRegister)
	}
	in.PasteKill()
	if in.Value() != "hello" {
		t.Errorf("after PasteKill Value = %q, want hello", in.Value())
	}
}

func TestPlainMovementClearsSelection(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftRight()
	if !in.HasSelection() {
		t.Fatal("expected selection")
	}
	in.MoveRight()
	if in.HasSelection() {
		t.Error("plain movement should clear selection")
	}
}

func TestUndoRestoresSelection(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.SelectAll()
	in.InsertRune('x')
	if in.Value() != "x" {
		t.Fatalf("Value = %q, want x", in.Value())
	}
	in.Undo()
	if in.Value() != "hello" {
		t.Errorf("after undo Value = %q, want hello", in.Value())
	}
	if !in.HasSelection() {
		t.Error("undo should restore the selection")
	}
	in.Redo()
	if in.Value() != "x" {
		t.Errorf("after redo Value = %q, want x", in.Value())
	}
}

func TestShiftHomeEnd(t *testing.T) {
	in := NewInput()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.ShiftEnd()
	if in.SelectedText() != "hello" {
		t.Errorf("ShiftEnd SelectedText = %q, want hello", in.SelectedText())
	}

	in2 := NewInput()
	for _, r := range "hello" {
		in2.InsertRune(r)
	}
	in2.End()
	in2.ShiftLeft()
	in2.ShiftHome()
	if in2.SelectedText() != "hello" {
		t.Errorf("ShiftHome SelectedText = %q, want hello", in2.SelectedText())
	}
}

func TestCellSelected(t *testing.T) {
	s, e := textPos{1, 2}, textPos{3, 4}
	cases := []struct {
		row, col int
		want     bool
	}{
		{0, 0, false},
		{1, 1, false},
		{1, 2, true},
		{2, 0, true},
		{2, 9, true},
		{3, 3, true},
		{3, 4, false},
		{4, 0, false},
	}
	for _, c := range cases {
		if got := cellSelected(c.row, c.col, s, e); got != c.want {
			t.Errorf("cellSelected(%d,%d) = %v, want %v", c.row, c.col, got, c.want)
		}
	}
}
