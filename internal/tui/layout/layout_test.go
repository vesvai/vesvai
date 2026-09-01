package layout

import "testing"

func TestCenterIn(t *testing.T) {
	outer := Region{Left: 0, Top: 0, Width: 80, Height: 24}
	got := CenterIn(outer, 18, 8)
	want := Region{Left: 31, Top: 8, Width: 18, Height: 8}
	if got != want {
		t.Errorf("CenterIn: got %+v, want %+v", got, want)
	}
}

func TestCenterInLargerThanOuter(t *testing.T) {
	outer := Region{Left: 0, Top: 0, Width: 10, Height: 4}
	got := CenterIn(outer, 18, 8)
	if got.Width != 10 || got.Height != 4 {
		t.Errorf("CenterIn should clamp to outer, got %+v", got)
	}
}

func TestCenterH(t *testing.T) {
	outer := Region{Left: 0, Top: 0, Width: 80, Height: 24}
	got := CenterH(outer, 20, 2, 4)
	want := Region{Left: 30, Top: 2, Width: 20, Height: 4}
	if got != want {
		t.Errorf("CenterH: got %+v, want %+v", got, want)
	}
}

func TestBottomAligned(t *testing.T) {
	outer := Region{Left: 0, Top: 0, Width: 80, Height: 24}
	got := BottomAligned(outer, 1)
	want := Region{Left: 0, Top: 23, Width: 80, Height: 1}
	if got != want {
		t.Errorf("BottomAligned: got %+v, want %+v", got, want)
	}
}

func TestBottomAlignedClipped(t *testing.T) {
	outer := Region{Left: 0, Top: 0, Width: 80, Height: 4}
	got := BottomAligned(outer, 10)
	if got.Top != 0 || got.Height != 4 {
		t.Errorf("BottomAligned should clip height, got %+v", got)
	}
}

func TestTopAligned(t *testing.T) {
	outer := Region{Left: 0, Top: 0, Width: 80, Height: 24}
	got := TopAligned(outer, 3)
	want := Region{Left: 0, Top: 0, Width: 80, Height: 3}
	if got != want {
		t.Errorf("TopAligned: got %+v, want %+v", got, want)
	}
}

func TestPad(t *testing.T) {
	r := Region{Left: 0, Top: 0, Width: 40, Height: 10}
	got := Pad(r, 2, 1)
	want := Region{Left: 2, Top: 1, Width: 36, Height: 8}
	if got != want {
		t.Errorf("Pad: got %+v, want %+v", got, want)
	}
}

func TestPadTooBig(t *testing.T) {
	r := Region{Left: 0, Top: 0, Width: 4, Height: 4}
	got := Pad(r, 10, 10)
	if got.Width < 0 || got.Height < 0 || got.Left < 0 || got.Top < 0 {
		t.Errorf("Pad should clamp to non-negative, got %+v", got)
	}
}

func TestStackV(t *testing.T) {
	outer := Region{Left: 0, Top: 0, Width: 80, Height: 24}
	got := StackV(outer, 5, 10, 9)
	if len(got) != 3 {
		t.Fatalf("expected 3 regions, got %d", len(got))
	}
	if got[0].Top != 0 || got[1].Top != 5 || got[2].Top != 15 {
		t.Errorf("stack tops wrong: %+v", got)
	}
	if got[2].Height != 9 {
		t.Errorf("stack height wrong: %+v", got[2])
	}
}

func TestStackH(t *testing.T) {
	outer := Region{Left: 0, Top: 0, Width: 80, Height: 24}
	got := StackH(outer, 10, 70)
	if got[0].Left != 0 || got[1].Left != 10 {
		t.Errorf("stack lefts wrong: %+v", got)
	}
}

func TestClamp(t *testing.T) {
	r := Region{Left: -2, Top: 20, Width: 100, Height: 10}
	outer := Region{Left: 0, Top: 0, Width: 80, Height: 24}
	got := r.Clamp(outer)
	if got.Left != 0 || got.Width != 80 || got.Bottom() != 24 {
		t.Errorf("Clamp wrong: %+v", got)
	}
}

func TestRegionEmpty(t *testing.T) {
	zero := Region{Width: 0, Height: 0}
	one := Region{Width: 1, Height: 1}
	if !zero.Empty() {
		t.Error("zero region should be empty")
	}
	if one.Empty() {
		t.Error("non-empty region reported empty")
	}
}
