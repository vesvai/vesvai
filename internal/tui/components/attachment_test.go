package components

import (
	"image"

	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/tui"

	"github.com/gdamore/tcell/v2"
)

func pasteInto(ta *Textarea, text string) {
	ta.HandleEvent(tcell.NewEventPaste(true))
	for _, r := range text {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	ta.HandleEvent(tcell.NewEventPaste(false))
}

func TestPasteDetectsFilesAsAttachments(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "photo.png")
	vid := filepath.Join(dir, "clip.mp4")
	pdf := filepath.Join(dir, "paper.pdf")
	for _, p := range []string{img, vid, pdf} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ta, _ := newMentionTestTextarea()
	var got []*tui.Attachment
	ta.OnAttachment = func(a *tui.Attachment) { got = append(got, a) }

	pasteInto(ta, "see "+img+" "+vid+" and "+pdf+" please")
	if len(got) != 3 {
		t.Fatalf("attachments = %d, want 3", len(got))
	}
	kinds := map[string]bool{}
	for _, a := range got {
		kinds[a.Kind] = true
	}
	if !kinds["image"] || !kinds["video"] || !kinds["pdf"] {
		t.Fatalf("kinds = %v, want image/video/pdf", kinds)
	}
	if ta.text() != "see and please" {
		t.Fatalf("text = %q, want %q", ta.text(), "see and please")
	}
}

func TestPasteNonFileTextStays(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	pasteInto(ta, "hello world")
	if ta.text() != "hello world" {
		t.Fatalf("text = %q", ta.text())
	}
}

func TestPasteIgnoresNonexistentPaths(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	pasteInto(ta, "go run ./cmd/vesvai")
	if ta.text() != "go run ./cmd/vesvai" {
		t.Fatalf("text = %q", ta.text())
	}
}

func TestPasteSpacedPathDetectsFile(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "Screenshot from 2025-04-10 22-45-34.png")
	if err := os.WriteFile(img, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ta, _ := newMentionTestTextarea()
	var got []*tui.Attachment
	ta.OnAttachment = func(a *tui.Attachment) { got = append(got, a) }

	pasteInto(ta, img+"\n")
	if len(got) != 1 || got[0].Kind != "image" {
		t.Fatalf("attachments = %+v, want the spaced image path", got)
	}
	if ta.text() != "" {
		t.Fatalf("spaced path must not remain in the text, got %q", ta.text())
	}
}

func TestUnbracketedPasteSpacedPath(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "Screenshot from 2025-04-10 22-45-34.png")
	if err := os.WriteFile(img, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ta, _ := newMentionTestTextarea()
	var got []*tui.Attachment
	ta.OnAttachment = func(a *tui.Attachment) { got = append(got, a) }
	for _, r := range img {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	if len(got) != 1 {
		t.Fatalf("attachments = %+v, want the spaced image path", got)
	}
	if ta.text() != "" {
		t.Fatalf("spaced path must not remain in the text, got %q", ta.text())
	}
}

func TestUnbracketedPasteSpacedPathWithTrailingText(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "my shot.png")
	if err := os.WriteFile(img, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ta, _ := newMentionTestTextarea()
	var got []*tui.Attachment
	ta.OnAttachment = func(a *tui.Attachment) { got = append(got, a) }
	for _, r := range img + " look at this" {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	if len(got) != 1 {
		t.Fatalf("attachments = %+v, want one", got)
	}
	if ta.text() != " look at this" {
		t.Fatalf("text = %q, want %q", ta.text(), " look at this")
	}
}

func TestUnbracketedPasteDetectsFiles(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(img, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ta, _ := newMentionTestTextarea()
	var got []*tui.Attachment
	ta.OnAttachment = func(a *tui.Attachment) { got = append(got, a) }

	for _, r := range img {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	if len(got) != 1 || got[0].Kind != "image" {
		t.Fatalf("attachments = %+v, want the image", got)
	}
	if ta.text() != "" {
		t.Fatalf("path must not remain in the text, got %q", ta.text())
	}
}

func TestUnbracketedPasteKeepsNonPaths(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	var got []*tui.Attachment
	ta.OnAttachment = func(a *tui.Attachment) { got = append(got, a) }
	for _, r := range "hello world" {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	if len(got) != 0 {
		t.Fatalf("unexpected attachments: %+v", got)
	}
	if ta.text() != "hello world" {
		t.Fatalf("text = %q", ta.text())
	}
}

func TestUnbracketedPasteTrailingNewlineNotSubmit(t *testing.T) {
	ta, _ := newMentionTestTextarea()
	submitted := false
	ta.OnSubmit = func(string) { submitted = true }
	for _, r := range "/etc/hosts " {
		ta.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	ta.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if submitted {
		t.Fatal("trailing paste newline must not submit")
	}
	if !strings.Contains(ta.text(), "\n") {
		t.Fatalf("newline not inserted, text = %q", ta.text())
	}
}

func TestAttachmentSelectionHighlight(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(60, 20)

	b := NewAttachmentBar()
	for i := 0; i < 3; i++ {
		b.Add(&tui.Attachment{Name: "f" + string(rune('a'+i)) + ".png", Kind: "image", Size: 1})
	}
	b.Layout(image.Rect(2, 10, 58, 15))
	b.MarkDirty()
	b.Render(s, DefaultTestPalette())
	s.Show()

	pal := DefaultTestPalette()
	accentCards := func() int {
		cells, _, _ := s.GetContents()
		n := 0
		for i := 0; i < len(cells); i++ {
			c := cells[i]
			if c.Runes == nil || len(c.Runes) == 0 {
				continue
			}
			if c.Runes[0] == '┌' {
				fg, _, _ := c.Style.Decompose()
				if fg == pal.Accent {
					n++
				}
			}
		}
		return n
	}

	if n := accentCards(); n != 0 {
		t.Fatalf("unfocused bar shows %d highlighted cards", n)
	}

	b.SetFocused(true)
	b.MarkDirty()
	b.Render(s, pal)
	s.Show()
	if n := accentCards(); n != 1 {
		t.Fatalf("focused bar shows %d highlighted cards, want 1", n)
	}

	b.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	b.MarkDirty()
	b.Render(s, pal)
	s.Show()
	if n := accentCards(); n != 1 {
		t.Fatalf("after moving selection: %d highlighted cards, want 1", n)
	}
}

func TestAttachmentArrowsHiddenForSinglePage(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(60, 20)

	render := func(b *AttachmentBar) string {
		b.Layout(image.Rect(2, 10, 58, 15))
		b.MarkDirty()
		b.Render(s, DefaultTestPalette())
		s.Show()
		cells, _, _ := s.GetContents()
		var sb strings.Builder
		for i := 0; i < len(cells); i++ {
			c := cells[i]
			if c.Runes != nil && len(c.Runes) > 0 {
				sb.WriteRune(c.Runes[0])
			}
		}
		return sb.String()
	}

	b := NewAttachmentBar()
	b.Add(&tui.Attachment{Name: "a.png", Kind: "image", Size: 1})
	frame := render(b)
	if strings.Contains(frame, "‹") || strings.Contains(frame, "›") {
		t.Fatalf("single attachment must not show paging arrows:\n%s", frame)
	}

	b2 := NewAttachmentBar()
	for i := 0; i < 4; i++ {
		b2.Add(&tui.Attachment{Name: "f.png", Kind: "image", Size: 1})
	}
	frame = render(b2)
	if !strings.Contains(frame, "‹") || !strings.Contains(frame, "›") {
		t.Fatalf("multi-page bar must show paging arrows:\n%s", frame)
	}
}

func TestAttachmentLastPageSingleHighlight(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(60, 20)

	b := NewAttachmentBar()
	for i := 0; i < 4; i++ {
		b.Add(&tui.Attachment{Name: "f" + string(rune('a'+i)) + ".png", Kind: "image", Size: 1})
	}
	b.SetFocused(true)
	b.Layout(image.Rect(2, 10, 58, 15))

	accentCards := func() int {
		cells, _, _ := s.GetContents()
		n := 0
		for i := 0; i < len(cells); i++ {
			c := cells[i]
			if c.Runes != nil && len(c.Runes) > 0 && c.Runes[0] == '┌' {
				fg, _, _ := c.Style.Decompose()
				if fg == DefaultTestPalette().Accent {
					n++
				}
			}
		}
		return n
	}

	for i := 0; i < 3; i++ {
		b.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	}
	b.MarkDirty()
	b.Render(s, DefaultTestPalette())
	s.Show()
	if n := accentCards(); n != 1 {
		t.Fatalf("last page shows %d highlighted cards, want exactly 1", n)
	}
}

func TestAttachmentKindDetection(t *testing.T) {
	cases := map[string]string{
		"a.png": "image", "b.jpeg": "image", "c.webp": "image",
		"d.mp4": "video", "e.mov": "video", "f.webm": "video",
		"g.pdf": "pdf",
		"h.txt": "file", "i.go": "file",
	}
	for name, want := range cases {
		if got := tui.AttachmentKind(name); got != want {
			t.Fatalf("%s: kind = %q, want %q", name, got, want)
		}
	}
}

func TestAttachmentBarPaging(t *testing.T) {
	b := NewAttachmentBar()
	if b.DesiredHeight() != 0 || b.Focusable() {
		t.Fatal("empty bar should collapse and not be focusable")
	}
	for i := 0; i < 7; i++ {
		b.Add(&tui.Attachment{Name: "f" + string(rune('a'+i)) + ".png", Kind: "image", Size: 1024})
	}
	if b.DesiredHeight() != 5 {
		t.Fatalf("height = %d, want 5", b.DesiredHeight())
	}
	if !b.Focusable() {
		t.Fatal("bar with attachments must be focusable")
	}
	if n := b.pageCount(); n != 3 {
		t.Fatalf("pages = %d, want 3", n)
	}
	if b.Page() != 0 {
		t.Fatalf("page = %d, want 0", b.Page())
	}
	for i := 0; i < 5; i++ {
		b.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	}
	if b.Page() != 1 {
		t.Fatalf("page after five rights = %d, want 1", b.Page())
	}
	b.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if b.Page() != 1 {
		t.Fatalf("page after left = %d, want 1", b.Page())
	}
	b.HandleEvent(tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone))
	if b.Page() != 2 || b.cursor != 6 {
		t.Fatalf("End should select the last attachment (cursor=6, page=2), got cursor=%d page=%d", b.cursor, b.Page())
	}
	b.HandleEvent(tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone))
	if b.Page() != 0 || b.cursor != 0 {
		t.Fatalf("Home should select the first attachment, got cursor=%d page=%d", b.cursor, b.Page())
	}
}

func TestAttachmentSelectionAndDelete(t *testing.T) {
	b := NewAttachmentBar()
	for i := 0; i < 4; i++ {
		b.Add(&tui.Attachment{Name: "f" + string(rune('a'+i)) + ".png", Kind: "image", Size: 1})
	}
	if !b.Focused() {
		b.SetFocused(true)
	}
	if b.Selected() == nil || b.Selected().Name != "fa.png" {
		t.Fatalf("focus must select the first card, got %+v", b.Selected())
	}

	b.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if b.Selected().Name != "fb.png" {
		t.Fatalf("cursor = %+v, want fb.png", b.Selected())
	}
	b.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if b.Selected().Name != "fa.png" {
		t.Fatalf("cursor = %+v, want fa.png", b.Selected())
	}

	b.HandleEvent(tcell.NewEventKey(tcell.KeyDelete, 0, tcell.ModNone))
	if len(b.attachments) != 3 {
		t.Fatalf("attachments = %d, want 3", len(b.attachments))
	}
	if b.Selected().Name != "fb.png" {
		t.Fatalf("cursor after delete = %+v, want fb.png", b.Selected())
	}

	b.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone))
	if len(b.attachments) != 2 {
		t.Fatalf("attachments = %d, want 2", len(b.attachments))
	}
	for i := 0; i < 2; i++ {
		b.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone))
	}
	if len(b.attachments) != 0 || b.DesiredHeight() != 0 {
		t.Fatalf("empty bar must collapse, n=%d h=%d", len(b.attachments), b.DesiredHeight())
	}
	if b.Selected() != nil {
		t.Fatalf("empty bar must have no selection")
	}
}

func TestAttachmentBarRendersCardsAndDots(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(60, 20)

	b := NewAttachmentBar()
	for i := 0; i < 5; i++ {
		b.Add(&tui.Attachment{Name: "photo" + string(rune('1'+i)) + ".png", Kind: "image", Size: 2 * 1024 * 1024})
	}
	b.Layout(image.Rect(2, 10, 58, 14))
	b.MarkDirty()
	b.Render(s, DefaultTestPalette())
	s.Show()

	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	var sb strings.Builder
	for i := 0; i < len(cells); i++ {
		c := cells[i]
		if c.Runes != nil && len(c.Runes) > 0 {
			sb.WriteRune(c.Runes[0])
		} else {
			sb.WriteByte(' ')
		}
	}
	frame := sb.String()
	for _, want := range []string{"‹", "›", "photo1", "photo3", "●", "○"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("attachment bar missing %q:\n%s", want, frame)
		}
	}
	_ = w
}
