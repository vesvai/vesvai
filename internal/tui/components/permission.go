package components

import (
	"encoding/json"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/permission"
	"github.com/vesvai/vesvai/internal/tui"
)

const maxPermissionBodyRows = 7

type PermissionModal struct {
	*Base
	toolName string
	params   map[string]any
	reason   string

	OnDecision func(d permission.Decision)
}

func NewPermissionModal(toolName string, params map[string]any, reason string) *PermissionModal {
	p := &PermissionModal{
		Base:     NewBase("permission"),
		toolName: toolName,
		params:   params,
		reason:   reason,
	}
	p.SetDraw(p.draw)
	return p
}

func (p *PermissionModal) DesiredHeight() int {
	n := len(p.bodyLines())
	if n > maxPermissionBodyRows {
		n = maxPermissionBodyRows
	}
	return 5 + n
}

func (p *PermissionModal) bodyLines() []string {
	var lines []string
	if p.reason != "" {
		for _, l := range strings.Split(p.reason, "\n") {
			lines = append(lines, l)
		}
		lines = append(lines, "")
	}
	if len(p.params) > 0 {
		if b, err := json.MarshalIndent(p.params, "", "  "); err == nil {
			for _, l := range strings.Split(string(b), "\n") {
				lines = append(lines, l)
			}
		}
	}
	return lines
}

func (p *PermissionModal) draw(s tcell.Screen, pal *tui.Palette) {
	rect := p.Bounds()
	if rect.Dx() < 8 || rect.Dy() < 5 {
		return
	}
	inner := drawModalBox(s, rect, "Permission required", pal)
	ix0, iy0 := inner.Min.X, inner.Min.Y
	iw := inner.Dx()

	headStyle := pal.Style(pal.Accent, pal.Surface).Bold(true)
	head := tui.Line{{R: ' ', S: headStyle}}
	for _, r := range "tool: " + p.toolName {
		head = append(head, tui.Cell{R: r, S: headStyle})
	}
	if len(head) > iw {
		head = head[:iw]
	}
	for len(head) < iw {
		head = append(head, tui.Cell{R: ' ', S: headStyle})
	}
	tui.DrawLine(s, ix0, iy0, head)

	textStyle := pal.Style(pal.Foreground, pal.Surface)
	lines := p.bodyLines()
	avail := inner.Dy() - 2
	if len(lines) > avail {
		lines = lines[:avail]
	}
	for i, l := range lines {
		line := tui.Line{{R: ' ', S: textStyle}}
		for _, r := range l {
			line = append(line, tui.Cell{R: r, S: textStyle})
		}
		if len(line) > iw {
			line = line[:iw]
		}
		for len(line) < iw {
			line = append(line, tui.Cell{R: ' ', S: textStyle})
		}
		tui.DrawLine(s, ix0, iy0+1+i, line)
	}

	hintStyle := pal.Style(pal.Muted, pal.Surface)
	hint := " [a] allow · [A] allow always · [d] deny · [esc] cancel"
	hintLine := tui.Line{{R: ' ', S: hintStyle}}
	for _, r := range hint {
		hintLine = append(hintLine, tui.Cell{R: r, S: hintStyle})
	}
	if len(hintLine) > iw {
		hintLine = hintLine[:iw]
	}
	for len(hintLine) < iw {
		hintLine = append(hintLine, tui.Cell{R: ' ', S: hintStyle})
	}
	tui.DrawLine(s, ix0, iy0+inner.Dy()-1, hintLine)
}

func (p *PermissionModal) HandleEvent(ev tcell.Event) bool {
	e, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}
	switch e.Key() {
	case tcell.KeyEsc:
		p.decide(permission.DecisionDeny)
		return true
	case tcell.KeyRune:
		switch e.Rune() {
		case 'a', 'y':
			p.decide(permission.DecisionAllow)
			return true
		case 'A':
			p.decide(permission.DecisionAllowAlways)
			return true
		case 'd', 'n':
			p.decide(permission.DecisionDeny)
			return true
		}
	}
	return false
}

func (p *PermissionModal) decide(d permission.Decision) {
	if p.OnDecision != nil {
		p.OnDecision(d)
	}
}
