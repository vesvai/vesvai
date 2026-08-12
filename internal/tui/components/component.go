package components

import (
	"image"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type Component interface {
	Layout(bounds image.Rectangle)
	Render(s tcell.Screen, pal *tui.Palette) bool
	Tick(elapsed time.Duration) bool
	HandleEvent(ev tcell.Event) bool
	SetFocused(f bool)
	Focused() bool
	Focusable() bool
	IsDirty() bool
	ClearDirty()
	MarkDirty()
}

type Base struct {
	id       string
	bounds   image.Rectangle
	dirty    bool
	focused  bool
	requests int
	now      time.Duration
	drawFn   func(s tcell.Screen, pal *tui.Palette)

	Hooks *tui.Hooks
}

func NewBase(id string) *Base {
	b := &Base{id: id, dirty: true}
	b.Hooks = tui.NewHooks(b)
	return b
}

func (b *Base) SetDraw(fn func(s tcell.Screen, pal *tui.Palette)) { b.drawFn = fn }

func (b *Base) ID() string { return b.id }

func (b *Base) requestRender() {
	b.dirty = true
	b.requests++
}

func (b *Base) RequestRender() { b.requestRender() }

func (b *Base) MarkDirty()                  { b.dirty = true }
func (b *Base) ClearDirty()                 { b.dirty = false }
func (b *Base) IsDirty() bool               { return b.dirty }
func (b *Base) SetFocused(f bool)           { b.focused = f }
func (b *Base) Focused() bool               { return b.focused }
func (b *Base) Focusable() bool             { return false }
func (b *Base) Layout(r image.Rectangle)    { b.bounds = r }
func (b *Base) Bounds() image.Rectangle     { return b.bounds }
func (b *Base) SetBounds(r image.Rectangle) { b.bounds = r }
func (b *Base) Requests() int               { return b.requests }
func (b *Base) Width() int                  { return b.bounds.Dx() }
func (b *Base) Height() int                 { return b.bounds.Dy() }

func (b *Base) Render(s tcell.Screen, pal *tui.Palette) bool {
	if !b.dirty {
		return false
	}
	b.Hooks.BeginRender()
	before := b.requests
	if b.drawFn != nil {
		b.drawFn(s, pal)
	}
	b.Hooks.EndRender(before, func() int { return b.requests })
	b.dirty = false
	return b.requests > before
}

func (b *Base) Tick(elapsed time.Duration) bool {
	return b.Hooks.Tick(elapsed)
}

func (b *Base) Draw(s tcell.Screen, pal *tui.Palette) {}

func (b *Base) HandleEvent(ev tcell.Event) bool { return false }
