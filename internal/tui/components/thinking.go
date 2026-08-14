package components

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type ThinkingView struct{}

func NewThinkingView() *ThinkingView { return &ThinkingView{} }

func (v *ThinkingView) Lines(p *tui.Part, width int, pal *tui.Palette, active bool, now time.Duration) []tui.Line {
	expanded := p.ThinkingExpanded

	var lines []tui.Line
	header := v.headerLine(p, pal, active, now)
	lines = append(lines, header)

	if expanded && p.ThinkingText() != "" {
		body := tui.WrapText(p.ThinkingText(), pal.Style(pal.Reasoning, pal.Background), width)
		lines = append(lines, body...)
	}
	return lines
}

func (v *ThinkingView) headerLine(p *tui.Part, pal *tui.Palette, active bool, now time.Duration) tui.Line {
	dim := pal.Style(pal.TextDim, pal.Background)

	var mark rune
	var markStyle tcell.Style
	if active {
		mark = '◌'
		markStyle = pal.Style(pal.ThinkingGlow, pal.Background)
	} else {
		mark = '▸'
		if p.ThinkingExpanded {
			mark = '▾'
		}
		markStyle = pal.Style(pal.Accent, pal.Background)
	}

	label := "Thinking"
	labelStyle := pal.Style(pal.TextDim, pal.Background)
	if active {
		labelStyle = pal.Style(lerpColor(pal.ThinkingDim, pal.ThinkingGlow, glowT(now)), pal.Background)
	}

	line := tui.Line{
		{R: mark, S: markStyle},
		{R: ' ', S: dim},
	}
	for _, r := range label {
		line = append(line, tui.Cell{R: r, S: labelStyle})
	}

	if active {
		dots := int(now / (300 * time.Millisecond) % 4)
		for i := 0; i < 4; i++ {
			r := '·'
			if i >= dots {
				r = ' '
			}
			line = append(line, tui.Cell{R: r, S: dim})
		}
	}
	return line
}
