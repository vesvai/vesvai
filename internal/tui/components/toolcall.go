package components

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

const maxExpandedResultLines = 100

const maxToolStatusWidth = 16

func ToolLines(tc *tui.ToolCall, width int, pal *tui.Palette, now time.Duration) []tui.Line {
	var lines []tui.Line

	stateColor := pal.Running
	switch tc.State {
	case tui.ToolSuccess:
		stateColor = pal.Success
	case tui.ToolError:
		stateColor = pal.Error
	}

	marker := '▸'
	if tc.Expanded {
		marker = '▾'
	}

	left := tui.Line{
		{R: marker, S: pal.Style(pal.Accent, pal.Background)},
		{R: ' ', S: tcell.StyleDefault},
		{R: '⚙', S: pal.Style(pal.Muted, pal.Background)},
		{R: ' ', S: tcell.StyleDefault},
	}
	nameStyle := pal.Style(pal.Foreground, pal.Background).Bold(true)
	for _, r := range tc.Name {
		left = append(left, tui.Cell{R: r, S: nameStyle})
	}
	left = append(left, tui.Cell{R: ' ', S: tcell.StyleDefault})

	args := compactArgs(tc.Args)
	if args == "" {
		args = "—"
	}
	argsStyle := pal.Style(pal.TextDim, pal.Background)
	maxLeft := width - maxToolStatusWidth
	if maxLeft < 10 {
		maxLeft = 10
	}
	for _, r := range args {
		if left.Width()+2 > maxLeft {
			left = append(left, tui.Cell{R: '…', S: argsStyle})
			break
		}
		left = append(left, tui.Cell{R: r, S: argsStyle})
	}

	var status string
	switch tc.State {
	case tui.ToolRunning:
		status = fmt.Sprintf("%c running", spinnerAt(now))
	case tui.ToolSuccess:
		status = "✔ " + formatDuration(tc.Duration)
	case tui.ToolError:
		status = "✖ " + formatDuration(tc.Duration)
	}
	statusCells := tui.LineFromSegments([]tui.Segment{
		{Text: " " + status + " ", Style: pal.Style(stateColor, pal.Background)},
	}, len(status)+2)

	for left.Width()+statusCells.Width() < width {
		left = append(left, tui.Cell{R: ' ', S: tcell.StyleDefault})
	}
	left = append(left, statusCells...)
	if left.Width() > width {
		left = left[:width]
	}
	lines = append(lines, left)

	if tc.Expanded {
		body := tc.Result
		if tc.State == tui.ToolError && tc.Error != nil {
			body = "error: " + tc.Error.Error()
		}
		if body != "" {
			bodyStyle := pal.Style(pal.TextDim, pal.Background)
			wrapped := tui.WrapText(body, bodyStyle, width-4)
			if len(wrapped) > maxExpandedResultLines {
				wrapped = wrapped[:maxExpandedResultLines]
			}
			for _, ln := range wrapped {
				row := tui.Line{
					{R: ' ', S: tcell.StyleDefault},
					{R: ' ', S: tcell.StyleDefault},
				}
				row = append(row, ln...)
				lines = append(lines, row)
			}
			if len(wrapped) == maxExpandedResultLines {
				lines = append(lines, tui.LineFromSegments([]tui.Segment{
					{Text: "  … output truncated", Style: pal.Style(pal.Muted, pal.Background)},
				}, width))
			}
		}
	}

	return lines
}

func compactArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	b, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("%v", args)
	}
	return string(b)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
