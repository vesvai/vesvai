package components

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func SubagentStatusLine(sa *tui.Subagent, pal *tui.Palette, now time.Duration) tui.Line {
	dim := pal.Style(pal.TextDim, pal.Background)
	glow := func() tcell.Style {
		return pal.Style(lerpColor(pal.ThinkingDim, pal.ThinkingGlow, glowT(now)), pal.Background)
	}

	var line tui.Line
	switch sa.Status {
	case tui.StatusRunning:
		line = append(line, tui.Cell{R: '◌', S: glow()}, tui.Cell{R: ' ', S: dim})
		for _, r := range sa.Name {
			line = append(line, tui.Cell{R: r, S: pal.Style(pal.Subagent, pal.Background)})
		}
		line = append(line, tui.Cell{R: ' ', S: dim})

		if tc := lastSubagentTool(sa); tc != nil {
			line = append(line, tui.Cell{R: '·', S: dim}, tui.Cell{R: ' ', S: dim})
			if tc.State == tui.ToolRunning {
				line = append(line, tui.Cell{R: spinnerAt(now), S: pal.Style(pal.Accent, pal.Background)})
			} else {
				line = append(line, tui.Cell{R: '✔', S: pal.Style(pal.Success, pal.Background)})
			}
			line = append(line, tui.Cell{R: ' ', S: dim})
			for _, r := range tc.Name {
				line = append(line, tui.Cell{R: r, S: dim})
			}
		} else {
			for _, r := range "Thinking" {
				line = append(line, tui.Cell{R: r, S: glow()})
			}
			dots := int(now / (300 * time.Millisecond) % 4)
			for i := 0; i < 4; i++ {
				r := '·'
				if i >= dots {
					r = ' '
				}
				line = append(line, tui.Cell{R: r, S: dim})
			}
		}

	case tui.StatusDone:
		line = append(line, tui.Cell{R: '✔', S: pal.Style(pal.Success, pal.Background)}, tui.Cell{R: ' ', S: dim})
		for _, r := range sa.Name {
			line = append(line, tui.Cell{R: r, S: pal.Style(pal.Subagent, pal.Background)})
		}
		line = append(line, tui.Cell{R: ' ', S: dim})
		summary := "· " + formatDuration(sa.Duration)
		if n := len(sa.Tools); n > 0 {
			summary = fmt.Sprintf("· %d tools · %s", n, formatDuration(sa.Duration))
		}
		line = append(line, tui.LineFromSegments([]tui.Segment{{Text: summary, Style: dim}}, 40)...)

	case tui.StatusError:
		line = append(line, tui.Cell{R: '✖', S: pal.Style(pal.Error, pal.Background)}, tui.Cell{R: ' ', S: dim})
		for _, r := range sa.Name {
			line = append(line, tui.Cell{R: r, S: pal.Style(pal.Subagent, pal.Background)})
		}
		line = append(line, tui.Cell{R: ' ', S: dim})
		line = append(line, tui.LineFromSegments([]tui.Segment{
			{Text: "· error · " + formatDuration(sa.Duration), Style: pal.Style(pal.Error, pal.Background)},
		}, 40)...)
	}
	return line
}

func lastSubagentTool(sa *tui.Subagent) *tui.ToolCall {
	if len(sa.Tools) == 0 {
		return nil
	}
	return sa.Tools[len(sa.Tools)-1]
}
