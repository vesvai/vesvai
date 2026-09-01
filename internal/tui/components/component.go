package components

import (
	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/layout"
)

type Component interface {
	HandleKey(ev *tcell.EventKey) bool
	Draw(s tcell.Screen, bounds layout.Region, focused bool)
}

type Ticker interface {
	OnTick(blinkOn bool)
}

type Focusable interface {
	Component
	Focus()
	Blur()
	Focused() bool
}
