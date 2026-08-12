package layouts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestAttachmentBarInLayout(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	dir := t.TempDir()
	img := filepath.Join(dir, "photo.png")
	pdf := filepath.Join(dir, "report.pdf")
	vid := filepath.Join(dir, "clip.mp4")
	txt := filepath.Join(dir, "notes.txt")
	for _, p := range []string{img, pdf, vid, txt} {
		os.WriteFile(p, []byte("x"), 0o644)
	}

	l.HandleEvent(tcell.NewEventPaste(true))
	for _, r := range img + " " + pdf + " " + vid + " " + txt + " check this" {
		l.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	l.HandleEvent(tcell.NewEventPaste(false))

	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "photo.png") || !strings.Contains(joined, "report.pdf") {
		t.Fatalf("attachment cards missing:\n%s", joined)
	}
	if !strings.Contains(joined, "‹") || !strings.Contains(joined, "›") {
		t.Fatalf("paging arrows missing:\n%s", joined)
	}
	if !strings.Contains(joined, "●") {
		t.Fatalf("pagination dots missing:\n%s", joined)
	}
	if !strings.Contains(joined, "check this") {
		t.Fatalf("pasted text missing:\n%s", joined)
	}
	if strings.Contains(joined, dir) {
		t.Fatalf("file path leaked into the input:\n%s", joined)
	}
	cardRow, textareaRow := -1, -1
	for i, r := range rows {
		if strings.Contains(r, "photo.png") && cardRow < 0 {
			cardRow = i
		}
		if strings.Contains(r, "❯") && strings.Contains(r, "check this") {
			textareaRow = i
		}
	}
	if cardRow < 0 || textareaRow < 0 || cardRow >= textareaRow {
		t.Fatalf("cards not above the textarea (card=%d textarea=%d)", cardRow, textareaRow)
	}
}

func TestTabOrderViewportAttachmentsTextarea(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	if l.focusIdx != 2 {
		t.Fatalf("default focus = %d, want 2 (textarea)", l.focusIdx)
	}
	l.HandleEvent(key(tcell.KeyTab))
	if l.focusIdx != 0 {
		t.Fatalf("Tab from textarea with empty bar = %d, want 0 (viewport)", l.focusIdx)
	}

	b := l.AttachmentBar()
	b.Add(&tui.Attachment{Name: "a.png", Kind: "image", Size: 1})
	l.Layout(l.Bounds())

	l.HandleEvent(key(tcell.KeyTab))
	if l.focusIdx != 1 {
		t.Fatalf("Tab from viewport = %d, want 1 (attachments)", l.focusIdx)
	}
	l.HandleEvent(key(tcell.KeyTab))
	if l.focusIdx != 2 {
		t.Fatalf("Tab from attachments = %d, want 2 (textarea)", l.focusIdx)
	}
	l.HandleEvent(key(tcell.KeyTab))
	if l.focusIdx != 0 {
		t.Fatalf("Tab from textarea = %d, want 0 (viewport)", l.focusIdx)
	}
}

func TestSubmitAttachesAndClearsBar(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	l.Textarea().OnSubmit = func(text string) {
		msg := model.Conv.AddUser(text)
		msg.Attachments = l.AttachmentBar().TakeAll()
		l.NotifyModelChange()
	}
	renderFrame(t, l, s)

	dir := t.TempDir()
	img := filepath.Join(dir, "photo.png")
	if err := os.WriteFile(img, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	l.HandleEvent(tcell.NewEventPaste(true))
	for _, r := range img {
		l.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	l.HandleEvent(tcell.NewEventPaste(false))
	if l.AttachmentBar().Count() != 1 {
		t.Fatalf("bar count = %d, want 1", l.AttachmentBar().Count())
	}

	for _, r := range "ship it" {
		l.HandleEvent(runeKey(r))
	}
	l.HandleEvent(key(tcell.KeyEnter))

	if l.AttachmentBar().Count() != 0 {
		t.Fatalf("bar must clear after submit, count = %d", l.AttachmentBar().Count())
	}
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "photo.png") || !strings.Contains(joined, "ship it") {
		t.Fatalf("attached file not in the user message:\n%s", joined)
	}
	msg := model.Conv.Messages[0]
	if msg.Role != tui.RoleUser || len(msg.Attachments) != 1 || msg.Attachments[0].Name != "photo.png" {
		t.Fatalf("message attachments = %+v", msg.Attachments)
	}
}

func TestAttachmentArrowsNavigate(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	b := l.AttachmentBar()
	for i := 0; i < 7; i++ {
		b.Add(&tui.Attachment{Name: "f" + string(rune('a'+i)) + ".png", Kind: "image", Size: 1})
	}
	l.Layout(l.Bounds())

	l.HandleEvent(key(tcell.KeyTab))
	l.HandleEvent(key(tcell.KeyTab))
	if l.focusIdx != 1 {
		t.Fatalf("focus = %d, want 1", l.focusIdx)
	}
	if b.Selected() == nil || b.Selected().Name != "fa.png" {
		t.Fatalf("focus must select the first card, got %+v", b.Selected())
	}
	for i := 0; i < 6; i++ {
		l.HandleEvent(key(tcell.KeyRight))
	}
	if b.Selected() == nil || b.Selected().Name != "fg.png" {
		t.Fatalf("cursor = %+v, want fg.png", b.Selected())
	}
	if b.Page() != 2 {
		t.Fatalf("page = %d, want 2", b.Page())
	}
	rows := renderFrame(t, l, s)
	if !strings.Contains(frameText(rows), "fg.png") {
		t.Fatalf("last page card missing:\n%s", frameText(rows))
	}
	l.HandleEvent(key(tcell.KeyLeft))
	rows = renderFrame(t, l, s)
	if b.Page() != 1 || !strings.Contains(frameText(rows), "fd.png") {
		t.Fatalf("page 1 card missing after left:\n%s", frameText(rows))
	}
	l.HandleEvent(key(tcell.KeyBackspace))
	if b.Selected() == nil || b.Selected().Name != "fg.png" {
		t.Fatalf("after delete cursor = %+v, want fg.png", b.Selected())
	}
	if b.Count() != 6 {
		t.Fatalf("attachments = %d, want 6", b.Count())
	}
}
