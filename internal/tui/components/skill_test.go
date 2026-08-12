package components

import (
	"image"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func newSkillTestTextarea() (*Textarea, tcell.SimulationScreen) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	s.SetSize(60, 20)
	ta := NewTextarea()
	ta.SetSkillCatalog([]Mention{
		{ID: "graphify", Kind: "skill", Label: "graphify"},
		{ID: "impeccable", Kind: "skill", Label: "impeccable"},
		{ID: "refactor", Kind: "skill", Label: "refactor"},
	})
	ta.Layout(image.Rect(2, 13, 58, 19))
	ta.SetFocused(true)
	return ta, s
}

func TestSkillPickerOpensOnSlash(t *testing.T) {
	ta, s := newSkillTestTextarea()
	typeRunes(ta, "load /")
	if !ta.pickerOpen || ta.pickerKind != '/' {
		t.Fatalf("skill picker should open on '/', kind=%q", ta.pickerKind)
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
	for _, want := range []string{"graphify", "impeccable", "refactor"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("skill picker missing %q:\n%s", want, frame)
		}
	}
	if !strings.Contains(frame, "❯ /") {
		t.Fatalf("skill trigger not shown in query row:\n%s", frame)
	}
}

func TestSkillSelectInsertsBlock(t *testing.T) {
	ta, _ := newSkillTestTextarea()
	typeRunes(ta, "load /gra")
	if !ta.pickerOpen {
		t.Fatal("picker should be open")
	}
	if len(ta.pickerResults) != 1 || ta.pickerResults[0].ID != "graphify" {
		t.Fatalf("results = %+v, want only graphify", ta.pickerResults)
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if ta.pickerOpen {
		t.Fatal("picker should close after selection")
	}
	if ta.text() != "load /graphify" {
		t.Fatalf("text = %q, want %q", ta.text(), "load /graphify")
	}
}

func TestSkillBlockDeletesWhole(t *testing.T) {
	ta, _ := newSkillTestTextarea()
	typeRunes(ta, "/")
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'i', tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModNone))
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if ta.text() != "/impeccable" {
		t.Fatalf("text = %q", ta.text())
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone))
	if ta.text() != "" {
		t.Fatalf("one backspace should delete the whole skill block, got %q", ta.text())
	}
}

func TestUnknownSlashWordNotBlock(t *testing.T) {
	ta, _ := newSkillTestTextarea()
	for _, r := range "/notaskill" {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone))
	if ta.pickerOpen {
		t.Fatal("picker should close on a space")
	}
	if ta.text() != "/notaskill " {
		t.Fatalf("text = %q", ta.text())
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone))
	if ta.text() != "/notaskill" {
		t.Fatalf("unknown /word must delete char by char, got %q", ta.text())
	}
}

func TestSkillBlockRendersBabyBlue(t *testing.T) {
	ta, s := newSkillTestTextarea()
	typeRunes(ta, "/gra")
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	renderTextarea(ta, s)

	cells, _, _ := s.GetContents()
	pal := tui.DefaultDark()
	found := false
	for i := 0; i < len(cells); i++ {
		c := cells[i]
		if c.Runes != nil && len(c.Runes) > 0 && c.Runes[0] == '/' {
			fg, _, _ := c.Style.Decompose()
			if fg == pal.SkillBlock {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("skill block '/' cell is not baby blue")
	}
}

func TestSkillAndMentionColorsDiffer(t *testing.T) {
	ta, s := newSkillTestTextarea()
	ta.SetMentionCatalog([]Mention{{ID: "planner", Kind: "agent", Label: "planner"}})
	typeRunes(ta, "/gra")
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	typeRunes(ta, " and @")
	for _, r := range "pl" {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	renderTextarea(ta, s)

	cells, _, _ := s.GetContents()
	pal := tui.DefaultDark()
	blue := false
	gold := false
	for i := 0; i < len(cells); i++ {
		c := cells[i]
		if c.Runes == nil || len(c.Runes) == 0 {
			continue
		}
		fg, _, _ := c.Style.Decompose()
		switch c.Runes[0] {
		case '/':
			if fg == pal.SkillBlock {
				blue = true
			}
		case '@':
			if fg == pal.Mention {
				gold = true
			}
		}
	}
	if !blue || !gold {
		t.Fatalf("expected baby-blue skill and gold mention on the same line (blue=%v gold=%v)", blue, gold)
	}
}
