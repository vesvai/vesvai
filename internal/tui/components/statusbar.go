package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

const statusMsgDuration = 4 * time.Second

type Statusbar struct {
	*Base
	model *tui.Model
}

func NewStatusbar(model *tui.Model) *Statusbar {
	s := &Statusbar{Base: NewBase("statusbar"), model: model}
	s.SetDraw(s.draw)
	return s
}

func effortColor(effort string, pal *tui.Palette) tcell.Color {
	switch strings.ToLower(effort) {
	case "max":
		return pal.Error
	case "high":
		return pal.Warning
	case "low":
		return pal.Success
	default:
		return pal.Accent
	}
}

func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1e3)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (s *Statusbar) draw(screen tcell.Screen, pal *tui.Palette) {
	width := s.Width()
	if width < 1 {
		return
	}
	x0, y0 := s.bounds.Min.X, s.bounds.Min.Y

	bg := pal.Style(pal.TextDim, pal.StatusBg)
	tui.FillLine(screen, y0, bg)

	name := s.model.Model
	if name == "" {
		name = "vesvai"
	}
	segs := []tui.Segment{
		{Text: " " + name, Style: pal.Style(pal.Foreground, pal.StatusBg).Bold(true)},
	}
	if s.model.Provider != "" {
		segs = append(segs, tui.Segment{
			Text:  " " + s.model.Provider,
			Style: pal.Style(pal.Muted, pal.StatusBg),
		})
	}
	if s.model.Effort != "" {
		segs = append(segs, tui.Segment{
			Text:  " " + s.model.Effort,
			Style: pal.Style(effortColor(s.model.Effort, pal), pal.StatusBg),
		})
	}
	leftLine := tui.LineFromSegments(segs, width)
	tui.DrawLine(screen, x0, y0, leftLine)
	leftW := leftLine.Width()

	usage := s.model.Usage
	pct := ""
	pctStyle := pal.Style(pal.TextDim, pal.StatusBg)
	if s.model.ContextWindow > 0 && usage.TotalTokens > 0 {
		frac := s.model.UsageFraction() * 100
		pct = fmt.Sprintf(" (%.0f%%)", frac)
		switch {
		case frac > 90:
			pctStyle = pal.Style(pal.Error, pal.StatusBg)
		case frac > 75:
			pctStyle = pal.Style(pal.Warning, pal.StatusBg)
		}
	}
	midSegs := []tui.Segment{
		{Text: " " + formatTokens(usage.TotalTokens), Style: pal.Style(pal.Foreground, pal.StatusBg)},
		{Text: pct, Style: pctStyle},
		{Text: fmt.Sprintf(" · step %d ", s.model.Step), Style: pal.Style(pal.TextDim, pal.StatusBg)},
	}
	mid := tui.LineFromSegments(midSegs, width)
	tui.DrawLine(screen, x0+1+leftW, y0, mid)

	var right string
	var color tcell.Color

	if s.model.StatusMsgFresh(statusMsgDuration) {
		right = s.model.StatusMsg
		color = pal.Accent
	} else {
		switch {
		case s.model.Busy:
			right = "● running"
			color = pal.Running
		case s.model.Err != nil:
			right = "✖ error"
			color = pal.Error
		default:
			right = "✔ idle"
			color = pal.Success
		}
	}
	hints := " ⇥ focus · ⌃P cmd "
	rightSegs := []tui.Segment{
		{Text: " " + right, Style: pal.Style(color, pal.StatusBg).Bold(true)},
		{Text: hints, Style: pal.Style(pal.Muted, pal.StatusBg)},
	}
	rightText := " " + right + hints
	rightCells := tui.LineFromSegments(rightSegs, len(rightText)+1)
	rightX := x0 + width - len(rightText)
	tui.DrawLine(screen, rightX, y0, rightCells)
}

func (s *Statusbar) Tick(elapsed time.Duration) bool {
	s.now = elapsed
	dirty := false
	if s.model.StatusMsg != "" && !s.model.StatusMsgFresh(statusMsgDuration) {
		s.model.StatusMsg = ""
		dirty = true
	}
	return s.Hooks.Tick(elapsed) || dirty
}
