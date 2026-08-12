package layouts

import (
	"image"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/components"
)

var defaultActions = []components.Action{
	{ID: "new-session", Label: "New session", Category: "Session", Hint: "Ctrl+N"},
	{ID: "switch-session", Label: "Switch session", Category: "Session"},
	{ID: "delete-session", Label: "Delete session", Category: "Session"},
	{ID: "share-session", Label: "Share session", Category: "Session"},
	{ID: "connect-provider", Label: "Connect provider", Category: "Provider"},
	{ID: "change-model", Label: "Change model", Category: "Provider", Hint: "Ctrl+M"},
	{ID: "clear-conversation", Label: "Clear conversation", Category: "Display"},
}

type MainLayout struct {
	*components.Base

	viewport    *components.Viewport
	attachments *components.AttachmentBar
	textarea    *components.Textarea
	status      *components.Statusbar

	model *tui.Model
	pal   *tui.Palette

	focusIdx    int
	setFocusIdx func(int)

	modalStack []components.Component
	palette    *components.Palette
	sessions   *components.SessionPicker
	models     *components.ModelPicker

	menu *components.Menu

	OnAction        func(actionID string)
	OnSessionSelect func(sessionID string)
	OnModelSelect   func(info tui.ModelInfo)
	OnUserAction    func(actionID string, msg *tui.Message)

	relayoutPending bool
	rerenderPending bool
}

func NewMainLayout(model *tui.Model, pal *tui.Palette) *MainLayout {
	l := &MainLayout{
		Base:     components.NewBase("main"),
		model:    model,
		pal:      pal,
		status:   components.NewStatusbar(model),
		focusIdx: 2,
	}
	l.viewport = components.NewViewport(model.Conv)
	l.textarea = components.NewTextarea()
	l.attachments = components.NewAttachmentBar()
	l.textarea.OnAttachment = l.addAttachment

	l.palette = components.NewPalette(defaultActions)
	l.palette.OnRun = l.runAction
	l.palette.OnClose = l.closeModal

	l.sessions = components.NewSessionPicker(nil)
	l.sessions.OnSelect = l.pickSession
	l.sessions.OnBack = l.closeModal

	l.models = components.NewModelPicker(nil)
	l.models.OnSelect = l.pickModel
	l.models.OnBack = l.closeModal

	l.viewport.OnUserClick = l.openUserMenuAt
	return l
}

func (l *MainLayout) DefaultModel() tui.ModelInfo {
	models := l.models.Models()
	if len(models) == 0 {
		return tui.ModelInfo{}
	}
	return models[0]
}

func (l *MainLayout) SetSessions(sessions []components.Session) {
	l.sessions.SetSessions(sessions)
}

func (l *MainLayout) SetModels(models []tui.ModelInfo) {
	l.models.SetModels(models)
}

func (l *MainLayout) Models() []tui.ModelInfo {
	return l.models.Models()
}

func (l *MainLayout) PushModal(m components.Component) {
	l.openModal(m)
}

func (l *MainLayout) CloseModal() {
	l.closeModal()
}

func (l *MainLayout) CloseAllModals() {
	l.closeAllModals()
}

func (l *MainLayout) TopModal() components.Component {
	return l.topModal()
}

func (l *MainLayout) OpenSessionPicker() {
	l.closeAllModals()
	l.openModal(l.sessions)
}

func (l *MainLayout) OpenModelPicker() {
	l.closeAllModals()
	l.openModal(l.models)
}

func (l *MainLayout) addAttachment(a *tui.Attachment) {
	l.attachments.Add(a)
	l.Layout(l.Bounds())
}

func (l *MainLayout) openUserMenuAt(x, y int, msg *tui.Message) {
	items := []components.MenuItem{
		{ID: "fork", Label: "Fork", Run: func() { l.userAction("fork", msg) }},
		{ID: "revert", Label: "Revert", Run: func() { l.userAction("revert", msg) }},
		{ID: "copy", Label: "Copy", Run: func() { l.userAction("copy", msg) }},
	}
	menu := components.NewMenu(items)
	b := l.Bounds()
	menu.OpenAt(x, y, b.Dx(), b.Dy())
	menu.OnClose = l.closeMenu
	l.menu = menu
	l.MarkDirty()
}

func (l *MainLayout) closeMenu() {
	if l.menu == nil {
		return
	}
	l.menu = nil
	l.markAllDirty()
}

func (l *MainLayout) userAction(actionID string, msg *tui.Message) {
	l.closeMenu()
	if l.OnUserAction != nil {
		l.OnUserAction(actionID, msg)
	}
}

func (l *MainLayout) SessionByID(id string) *components.Session {
	for i := range l.sessions.Sessions() {
		s := l.sessions.Sessions()[i]
		if s.ID == id {
			return &s
		}
	}
	return nil
}

func (l *MainLayout) runAction(a components.Action) {
	if l.OnAction != nil {
		l.OnAction(a.ID)
	}
}

func (l *MainLayout) pickSession(s *components.Session) {
	if l.OnSessionSelect != nil {
		l.OnSessionSelect(s.ID)
	}
	l.closeAllModals()
}

func (l *MainLayout) pickModel(m tui.ModelInfo) {
	if l.OnModelSelect != nil {
		l.OnModelSelect(m)
	}
	l.closeAllModals()
}

func (l *MainLayout) hasModal() bool { return len(l.modalStack) > 0 }

func (l *MainLayout) topModal() components.Component {
	if len(l.modalStack) == 0 {
		return nil
	}
	return l.modalStack[len(l.modalStack)-1]
}

func (l *MainLayout) openModal(m components.Component) {
	if r, ok := m.(interface{ Reset() }); ok {
		r.Reset()
	}
	l.modalStack = append(l.modalStack, m)
	m.MarkDirty()
	l.layoutModal()
	l.MarkDirty()
}

func (l *MainLayout) closeModal() {
	if len(l.modalStack) == 0 {
		return
	}
	l.modalStack = l.modalStack[:len(l.modalStack)-1]
	l.markAllDirty()
	l.layoutModal()
}

func (l *MainLayout) closeAllModals() {
	if len(l.modalStack) == 0 {
		return
	}
	l.modalStack = nil
	l.markAllDirty()
}

func (l *MainLayout) modalDirty() bool {
	if m := l.topModal(); m != nil {
		return m.IsDirty()
	}
	return false
}

func (l *MainLayout) layoutModal() {
	m := l.topModal()
	if m == nil {
		return
	}
	h := 12
	if sz, ok := m.(interface{ DesiredHeight() int }); ok {
		h = sz.DesiredHeight()
	}
	w := 56
	b := l.Bounds()
	m.Layout(components.CenteredRect(b.Dx(), b.Dy(), w, h))
}

func (l *MainLayout) togglePalette() {
	if m := l.topModal(); m == l.palette {
		l.closeAllModals()
		return
	}
	l.closeAllModals()
	l.openModal(l.palette)
}

func (l *MainLayout) CanInterrupt() bool {
	return l.menu == nil && !l.hasModal() && !l.viewport.InSubagentView()
}

func (l *MainLayout) Viewport() *components.Viewport { return l.viewport }

func (l *MainLayout) AttachmentBar() *components.AttachmentBar { return l.attachments }

func (l *MainLayout) Textarea() *components.Textarea { return l.textarea }

func (l *MainLayout) Layout(bounds image.Rectangle) {
	l.SetBounds(bounds)
	x0, y0 := bounds.Min.X, bounds.Min.Y
	w, h := bounds.Dx(), bounds.Dy()

	if w < 10 || h < 5 {
		l.markAllDirty()
		return
	}

	statusH := 1
	barH := l.attachments.DesiredHeight()
	taH := l.textarea.DesiredHeight()
	if taH > h-statusH-barH-1 {
		taH = h - statusH - barH - 1
	}
	if taH < 1 {
		taH = 1
	}

	y := y0
	vh := h - statusH - taH - barH
	if vh < 1 {
		vh = 1
	}
	l.viewport.Layout(image.Rect(x0, y, x0+w, y+vh))
	y += vh
	l.attachments.Layout(image.Rect(x0, y, x0+w, y+barH))
	y += barH

	margin := 2
	if w < 2*margin+4 {
		margin = 0
	}
	l.textarea.Layout(image.Rect(x0+margin, y, x0+w-margin, y+taH))
	y += taH

	l.status.Layout(image.Rect(x0, y, x0+w, y+statusH))

	if l.hasModal() {
		l.layoutModal()
	}

	l.markAllDirty()
}

func (l *MainLayout) markAllDirty() {
	l.viewport.MarkDirty()
	l.attachments.MarkDirty()
	l.textarea.MarkDirty()
	l.status.MarkDirty()
	l.MarkDirty()
	l.relayoutPending = true
}

func (l *MainLayout) NotifyModelChange() {
	l.viewport.MarkDirty()
	l.status.MarkDirty()
}

func (l *MainLayout) Render(s tcell.Screen, pal *tui.Palette) bool {
	if !l.IsDirty() && !l.anyChildDirty() {
		return false
	}
	l.Hooks.BeginRender()
	before := l.Requests()

	if l.textarea.Height() != l.textarea.DesiredHeight() ||
		l.attachments.Height() != l.attachments.DesiredHeight() {
		l.Layout(l.Bounds())
	}

	focusIdx, setFocusIdx := tui.UseState(l.Hooks, l.focusIdx)
	l.focusIdx = focusIdx
	l.setFocusIdx = setFocusIdx

	l.Draw(s, pal)

	rerender := l.Hooks.EndRender(before, func() int { return l.Requests() })
	l.ClearDirty()
	return rerender || l.renderPending()
}

func (l *MainLayout) anyChildDirty() bool {
	return l.viewport.IsDirty() || l.attachments.IsDirty() ||
		l.textarea.IsDirty() || l.status.IsDirty() || l.modalDirty() ||
		(l.menu != nil && l.menu.IsDirty())
}

func (l *MainLayout) Draw(s tcell.Screen, pal *tui.Palette) {
	l.viewport.SetFocused(l.focusIdx == 0)
	l.attachments.SetFocused(l.focusIdx == 1)
	l.textarea.SetFocused(l.focusIdx == 2)

	if l.relayoutPending {
		bg := pal.Style(pal.Foreground, pal.Background)
		rect := l.Bounds()
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				s.SetContent(x, y, ' ', nil, bg)
			}
		}
		l.relayoutPending = false
	}

	rerender := false
	rerender = l.viewport.Render(s, pal) || rerender
	rerender = l.attachments.Render(s, pal) || rerender
	rerender = l.textarea.Render(s, pal) || rerender
	rerender = l.status.Render(s, pal) || rerender
	if m := l.topModal(); m != nil {
		m.MarkDirty()
		rerender = m.Render(s, pal) || rerender
	}
	if l.menu != nil {
		l.menu.MarkDirty()
		rerender = l.menu.Render(s, pal) || rerender
	}
	l.rerenderPending = rerender
}

func (l *MainLayout) renderPending() bool { return l.rerenderPending }

func (l *MainLayout) Tick(elapsed time.Duration) bool {
	dirty := false
	dirty = l.viewport.Tick(elapsed) || dirty
	dirty = l.attachments.Tick(elapsed) || dirty
	dirty = l.textarea.Tick(elapsed) || dirty
	dirty = l.status.Tick(elapsed) || dirty
	return dirty
}

func (l *MainLayout) HandleEvent(ev tcell.Event) bool {
	if l.menu != nil {
		return l.menu.HandleEvent(ev)
	}
	switch e := ev.(type) {
	case *tcell.EventKey:
		if e.Key() == tcell.KeyCtrlP {
			l.togglePalette()
			return true
		}
		if l.hasModal() {
			return l.topModal().HandleEvent(ev)
		}
		switch e.Key() {
		case tcell.KeyTab:
			l.cycleFocus(e.Modifiers()&tcell.ModShift == 0)
			return true
		}
	case *tcell.EventMouse:
		if l.hasModal() {
			return true
		}
		_, y := e.Position()
		ta := l.textarea.Bounds()
		if y >= ta.Min.Y && y < ta.Max.Y {
			if e.Buttons()&tcell.Button1 != 0 {
				l.focusTo(1)
			}
			return l.textarea.HandleEvent(e)
		}
		vp := l.viewport.Bounds()
		if y >= vp.Min.Y && y < vp.Max.Y {
			if e.Buttons()&tcell.Button1 != 0 {
				l.focusTo(0)
			}
			return l.viewport.HandleEvent(e)
		}
		return false
	case *tcell.EventPaste:
		if l.hasModal() {
			return true
		}
		return l.textarea.HandleEvent(e)
	}

	if w := l.focusTarget(l.focusIdx); w != nil {
		return w.HandleEvent(ev)
	}
	return false
}

func (l *MainLayout) focusTo(idx int) {
	if l.setFocusIdx != nil {
		l.setFocusIdx(idx)
	}
	l.focusIdx = idx
	l.viewport.MarkDirty()
	l.attachments.MarkDirty()
	l.textarea.MarkDirty()
}

func (l *MainLayout) focusTarget(idx int) components.Component {
	switch idx {
	case 0:
		return l.viewport
	case 1:
		return l.attachments
	case 2:
		return l.textarea
	}
	return nil
}

func (l *MainLayout) cycleFocus(forward bool) {
	const widgets = 3
	next := l.focusIdx
	for i := 0; i < widgets; i++ {
		if forward {
			next = (next + 1) % widgets
		} else {
			next = (next - 1 + widgets) % widgets
		}
		if w := l.focusTarget(next); w != nil && w.Focusable() {
			break
		}
	}
	l.focusTo(next)
}
