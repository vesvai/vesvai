package components

import (
	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

type Field struct {
	title       string
	value       []rune
	pos         int
	mask        rune
	placeholder string
	focused     bool
	onEnter     func(string)
	onCancel    func()
}

func NewField(title string) *Field { return &Field{title: title} }

func (f *Field) SetMask(r rune) { f.mask = r }

func (f *Field) SetPlaceholder(s string) { f.placeholder = s }

func (f *Field) SetOnEnter(fn func(string)) { f.onEnter = fn }

func (f *Field) SetOnCancel(fn func()) { f.onCancel = fn }

func (f *Field) Value() string { return string(f.value) }

func (f *Field) Focus() { f.focused = true }

func (f *Field) SetText(s string) {
	f.value = []rune(s)
	f.pos = len(f.value)
}

func (f *Field) HandleKey(ev *tcell.EventKey) bool {
	if !f.focused {
		return false
	}
	switch ev.Key() {
	case tcell.KeyEnter:
		if f.onEnter != nil {
			f.onEnter(f.Value())
		}
		return true
	case tcell.KeyEsc:
		if f.onCancel != nil {
			f.onCancel()
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if f.pos > 0 {
			f.value = append(f.value[:f.pos-1], f.value[f.pos:]...)
			f.pos--
		}
		return true
	case tcell.KeyDelete:
		if f.pos < len(f.value) {
			f.value = append(f.value[:f.pos], f.value[f.pos+1:]...)
		}
		return true
	case tcell.KeyLeft:
		if f.pos > 0 {
			f.pos--
		}
		return true
	case tcell.KeyRight:
		if f.pos < len(f.value) {
			f.pos++
		}
		return true
	case tcell.KeyHome:
		f.pos = 0
		return true
	case tcell.KeyEnd:
		f.pos = len(f.value)
		return true
	case tcell.KeyRune:
		if r := ev.Rune(); r != 0 && r >= ' ' {
			f.value = append(f.value[:f.pos], append([]rune{r}, f.value[f.pos:]...)...)
			f.pos++
		}
		return true
	}
	return false
}

func (f *Field) Draw(s tcell.Screen, bounds layout.Region, focused bool) {
	th := styles.Current()

	DrawText(s, bounds.Left, bounds.Top, f.title, th.Base().Foreground(th.Hint).Background(th.InputBg))

	box := layout.Region{Left: bounds.Left, Top: bounds.Top + 1, Width: bounds.Width, Height: 3}
	DrawBox(s, box, th.Base().Foreground(th.Border))
	inner := layout.Pad(box, 1, 1)

	var display []rune
	if len(f.value) > 0 {
		if f.mask != 0 {
			display = make([]rune, len(f.value))
			for i := range display {
				display[i] = f.mask
			}
		} else {
			display = f.value
		}
	} else if f.placeholder != "" {
		display = []rune(f.placeholder)
	}

	textStyle := th.Base().Foreground(th.InputText).Background(th.InputBg)
	if len(f.value) == 0 {
		textStyle = th.Base().Foreground(th.Placeholder).Background(th.InputBg)
	}
	DrawText(s, inner.Left, inner.Top, TruncateTo(string(display), inner.Width), textStyle)

	cx := inner.Left + f.pos
	cy := inner.Top
	if cx < inner.Right() {
		ch := ' '
		if f.pos < len(display) {
			ch = display[f.pos]
		}
		s.SetContent(cx, cy, ch, nil, th.Base().Foreground(th.InputBg).Background(th.Cursor))
	}

	DrawText(s, bounds.Left, bounds.Bottom()-1, "Enter save  ·  Esc cancel",
		th.Base().Foreground(th.Hint).Background(th.InputBg))
}
