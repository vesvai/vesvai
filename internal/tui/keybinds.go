package tui

import "github.com/gdamore/tcell/v2"

type keyCombo struct {
	key  tcell.Key
	rune rune
	mod  tcell.ModMask
}

type keyAction int

const (
	ActionNone keyAction = iota
	ActionQuit
	ActionThemeNext
	ActionSettings
)

var globalBindings = map[keyCombo]keyAction{
	{key: tcell.KeyCtrlC}: ActionQuit,
	{key: tcell.KeyCtrlQ}: ActionQuit,
	{key: tcell.KeyCtrlT}: ActionThemeNext,
	{key: tcell.KeyCtrlP}: ActionSettings,
}

func resolveGlobal(ev *tcell.EventKey) keyAction {
	if ev.Key() == tcell.KeyRune {
		return globalBindings[keyCombo{key: ev.Key(), rune: ev.Rune()}]
	}
	return globalBindings[keyCombo{key: ev.Key()}]
}
