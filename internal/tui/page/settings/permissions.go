package settings

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/agent/middlewares"
	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/builtin/middlewares/permission"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

type permPreset int

const (
	presetCustom permPreset = iota
	presetAuto
	presetManual
	presetJudge
	presetRecommended
)

var presetNames = []string{"Custom", "Auto", "Manual", "Judge", "Recommended"}

var presetModes = map[permPreset]map[string]string{
	presetAuto: {
		"read": "allow", "write": "allow", "edit": "allow", "delete": "allow",
		"list": "allow", "glob": "allow", "grep": "allow", "bash": "allow",
		"todo": "allow", "todoread": "allow", "todowrite": "allow",
		"webfetch": "allow", "websearch": "allow", "task": "allow", "taskstatus": "allow",
		"askuserquestion": "allow", "enterplanmode": "allow", "exitplanmode": "allow",
	},
	presetManual: {
		"read": "semi-ask", "write": "semi-ask", "edit": "semi-ask", "delete": "semi-ask",
		"list": "semi-ask", "glob": "semi-ask", "grep": "semi-ask", "bash": "semi-ask",
		"todo": "semi-ask", "todoread": "semi-ask", "todowrite": "semi-ask",
		"webfetch": "semi-ask", "websearch": "semi-ask", "task": "semi-ask", "taskstatus": "semi-ask",
		"askuserquestion": "semi-ask", "enterplanmode": "semi-ask", "exitplanmode": "semi-ask",
	},
	presetJudge: {
		"read": "semi-judge", "write": "semi-judge", "edit": "semi-judge", "delete": "semi-judge",
		"list": "semi-judge", "glob": "semi-judge", "grep": "semi-judge", "bash": "semi-judge",
		"todo": "semi-judge", "todoread": "semi-judge", "todowrite": "semi-judge",
		"webfetch": "semi-judge", "websearch": "semi-judge", "task": "semi-judge", "taskstatus": "semi-judge",
		"askuserquestion": "semi-judge", "enterplanmode": "semi-judge", "exitplanmode": "semi-judge",
	},
}

var allModes = []string{"allow", "semi-ask", "ask", "semi-judge", "judge"}

type permFocus int

const (
	permFocusPresets permFocus = iota
	permFocusTools
)

type toolPerm struct {
	name string
	mode string
}

type permissionsTab struct {
	settings  *Settings
	focus     permFocus
	toolIdx   int
	scroll    int
	tools     []toolPerm
	presetIdx int
	loaded    bool
}

func newPermissions(s *Settings) *permissionsTab {
	return &permissionsTab{settings: s, focus: permFocusPresets}
}

func (t *permissionsTab) loadIfNeeded() {
	if t.loaded {
		return
	}
	t.loaded = true
	t.loadTools()
	t.presetIdx = int(t.detectPreset())
	t.toolIdx = 0
	t.scroll = 0
}

func (t *permissionsTab) HandleKey(ev *tcell.EventKey) bool {
	t.loadIfNeeded()

	switch ev.Key() {
	case tcell.KeyUp:
		if t.focus == permFocusTools {
			if t.toolIdx > 0 {
				t.toolIdx--
			} else {
				t.focus = permFocusPresets
			}
		} else if t.focus == permFocusPresets {
			return false
		}
		return true

	case tcell.KeyDown:
		if t.focus == permFocusPresets {
			t.focus = permFocusTools
		} else {
			if t.toolIdx < len(t.tools)-1 {
				t.toolIdx++
			}
		}
		return true

	case tcell.KeyLeft:
		if t.focus == permFocusPresets {
			t.presetIdx = (t.presetIdx - 1 + len(presetNames)) % len(presetNames)
			t.applyPreset(permPreset(t.presetIdx))
		} else {
			if t.toolIdx >= 0 && t.toolIdx < len(t.tools) {
				t.cycleMode(t.toolIdx)
			}
		}
		return true

	case tcell.KeyRight:
		if t.focus == permFocusPresets {
			t.presetIdx = (t.presetIdx + 1) % len(presetNames)
			t.applyPreset(permPreset(t.presetIdx))
		} else {
			if t.toolIdx >= 0 && t.toolIdx < len(t.tools) {
				t.cycleModeRev(t.toolIdx)
			}
		}
		return true

	case tcell.KeyTab:
		return false

	case tcell.KeyPgUp:
		t.toolIdx -= 10
		if t.toolIdx < 0 {
			t.toolIdx = 0
		}
		return true
	case tcell.KeyPgDn:
		t.toolIdx += 10
		if t.toolIdx >= len(t.tools) {
			t.toolIdx = len(t.tools) - 1
		}
		return true
	case tcell.KeyHome:
		t.toolIdx = 0
		return true
	case tcell.KeyEnd:
		if len(t.tools) > 0 {
			t.toolIdx = len(t.tools) - 1
		}
		return true

	case tcell.KeyRune:
		switch ev.Rune() {
		case 'h':
			if t.focus == permFocusPresets {
				t.presetIdx = (t.presetIdx - 1 + len(presetNames)) % len(presetNames)
				t.applyPreset(permPreset(t.presetIdx))
			} else if t.toolIdx >= 0 && t.toolIdx < len(t.tools) {
				t.cycleMode(t.toolIdx)
			}
			return true
		case 'l':
			if t.focus == permFocusPresets {
				t.presetIdx = (t.presetIdx + 1) % len(presetNames)
				t.applyPreset(permPreset(t.presetIdx))
			} else if t.toolIdx >= 0 && t.toolIdx < len(t.tools) {
				t.cycleModeRev(t.toolIdx)
			}
			return true
		}
	}

	return false
}

func (t *permissionsTab) Draw(screen tcell.Screen, bounds layout.Region, focused bool) {
	t.loadIfNeeded()
	th := styles.Current()

	presetY := bounds.Top
	presetLabel := fmt.Sprintf("  %s  ", presetNames[t.presetIdx])

	presetFocused := focused && t.focus == permFocusPresets
	presetStyle := th.Base().Foreground(th.Accent).Background(th.InputBg)
	if presetFocused {
		presetStyle = th.Base().Foreground(th.InputText).Background(th.Selection)
	}

	arrowLeft := "◄ "
	components.DrawText(screen, bounds.Left+2, presetY, arrowLeft, presetStyle)
	presetX := bounds.Left + (bounds.Width-len(presetLabel))/2
	components.DrawText(screen, presetX, presetY, presetLabel, presetStyle)
	arrowRight := " ►"
	components.DrawText(screen, bounds.Right()-4, presetY, arrowRight, presetStyle)

	sepY := presetY + 1
	sepStyle := th.Base().Foreground(th.Muted).Background(th.InputBg)
	for x := bounds.Left + 1; x < bounds.Right()-1; x++ {
		screen.SetContent(x, sepY, '─', nil, sepStyle)
	}

	listTop := sepY + 2
	listBottom := bounds.Bottom() - 1
	visible := listBottom - listTop

	if t.toolIdx < t.scroll {
		t.scroll = t.toolIdx
	}
	if t.toolIdx >= t.scroll+visible {
		t.scroll = t.toolIdx - visible + 1
	}

	headerStyle := th.Base().Foreground(th.Muted).Background(th.InputBg)
	components.DrawText(screen, bounds.Left+2, listTop, "Tool", headerStyle)
	components.DrawText(screen, bounds.Right()-14, listTop, "Mode", headerStyle)
	listTop++

	for i := 0; i < visible-1; i++ {
		idx := t.scroll + i
		if idx >= len(t.tools) {
			break
		}
		tp := t.tools[idx]
		y := listTop + i

		toolFocused := focused && t.focus == permFocusTools && idx == t.toolIdx
		rowStyle := th.Base().Background(th.InputBg)
		if toolFocused {
			rowStyle = th.Base().Foreground(th.InputText).Background(th.Selection)
		}

		components.DrawText(screen, bounds.Left+2, y, components.TruncateTo(tp.name, bounds.Width-20), rowStyle)
		modeDisplay := tp.mode
		components.DrawText(screen, bounds.Right()-len(modeDisplay)-3, y, modeDisplay, rowStyle)
	}
}

func (t *permissionsTab) loadTools() {
	cfg := t.settings.deps.Config
	names := tools.Names()
	sort.Strings(names)

	t.tools = make([]toolPerm, 0, len(names))
	for _, name := range names {
		mode := t.resolveMode(cfg, name)
		t.tools = append(t.tools, toolPerm{name: name, mode: mode})
	}
}

func (t *permissionsTab) resolveMode(cfg *config.Config, name string) string {
	if cfg != nil && cfg.Permission != nil {
		if r, ok := cfg.Permission.Rules[name]; ok {
			return r
		}
		return cfg.Permission.Default
	}
	if mode, ok := config.DefaultConfig().Permission.Rules[name]; ok {
		return mode
	}
	return "semi-ask"
}

func (t *permissionsTab) detectPreset() permPreset {
	defaults := config.DefaultConfig().Permission.Rules
	if len(t.tools) == 0 {
		return presetCustom
	}
	for _, tp := range t.tools {
		defMode := "semi-ask"
		if mode, ok := defaults[tp.name]; ok {
			defMode = mode
		}
		if tp.mode != defMode {
			goto checkUniform
		}
	}
	return presetRecommended

checkUniform:
	for _, p := range []permPreset{presetAuto, presetManual, presetJudge} {
		expected := presetModes[p]
		matches := true
		for _, tp := range t.tools {
			exp, inPreset := expected[tp.name]
			if !inPreset {
				exp = "semi-ask"
				if v, ok := expected["read"]; ok {
					exp = v
				}
			}
			if tp.mode != exp {
				matches = false
				break
			}
		}
		if matches {
			return p
		}
	}
	return presetCustom
}

func (t *permissionsTab) applyPreset(p permPreset) {
	if p == presetRecommended {
		defaults := config.DefaultConfig().Permission.Rules
		for i := range t.tools {
			defMode := "semi-ask"
			if mode, ok := defaults[t.tools[i].name]; ok {
				defMode = mode
			}
			t.tools[i].mode = defMode
		}
		t.applyLive()
		return
	}
	expected, ok := presetModes[p]
	if !ok {
		return
	}
	for i := range t.tools {
		if exp, inPreset := expected[t.tools[i].name]; inPreset {
			t.tools[i].mode = exp
		} else {
			defVal := "semi-ask"
			if v, ok := expected["read"]; ok {
				defVal = v
			}
			t.tools[i].mode = defVal
		}
	}
	t.applyLive()
}

func (t *permissionsTab) cycleMode(idx int) {
	current := t.tools[idx].mode
	next := ""
	found := false
	for _, m := range allModes {
		if found {
			next = m
			break
		}
		if m == current {
			found = true
		}
	}
	if next == "" {
		next = allModes[0]
	}
	t.tools[idx].mode = next
	t.presetIdx = int(t.detectPreset())
	t.applyLive()
}

func (t *permissionsTab) cycleModeRev(idx int) {
	current := t.tools[idx].mode
	prev := allModes[len(allModes)-1]
	for i := len(allModes) - 2; i >= 0; i-- {
		if allModes[i] == current {
			break
		}
		prev = allModes[i]
	}
	t.tools[idx].mode = prev
	t.presetIdx = int(t.detectPreset())
	t.applyLive()
}

func (t *permissionsTab) applyLive() {
	rules := make(map[string]string, len(t.tools))
	for _, tp := range t.tools {
		rules[tp.name] = tp.mode
	}
	cfg := &config.PermissionConfig{
		Default: "semi-ask",
		Rules:   rules,
	}
	_ = config.UpsertPermission(cfg)
	if m, ok := middlewares.Get("permission"); ok {
		if perm, ok := m.(*permission.Middleware); ok {
			perm.UpdateConfig(cfg)
		}
	}
}
