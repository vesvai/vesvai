package app

import (
	"context"
	"errors"
	"image"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/filesystem"
	"github.com/vesvai/vesvai/internal/permission"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/skill"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layouts"
)

const tickInterval = 50 * time.Millisecond

const interruptWindow = 500 * time.Millisecond

var errInterrupted = errors.New("interrupted by user")

type interruptGate struct {
	window  time.Duration
	armedAt time.Time
}

func (g *interruptGate) armed() bool { return !g.armedAt.IsZero() }

func (g *interruptGate) press(now time.Time) bool {
	if g.armed() {
		if now.Sub(g.armedAt) <= g.window {
			g.armedAt = time.Time{}
			return true
		}
		g.armedAt = now
		return false
	}
	g.armedAt = now
	return false
}

func (g *interruptGate) expire(now time.Time) {
	if g.armed() && now.Sub(g.armedAt) > g.window {
		g.armedAt = time.Time{}
	}
}

type App struct {
	screen tcell.Screen
	pal    *tui.Palette
	model  *tui.Model
	layout *layouts.MainLayout
	driver Driver

	stream chan tui.StreamEvent
	input  chan tcell.Event
	ticker *time.Ticker

	backend Backend

	sessionID  string
	permission <-chan permissionRequest
	permModal  components.Component

	ctx    context.Context
	cancel context.CancelFunc
	start  time.Time

	interrupt interruptGate

	dirty bool
	done  bool
}

func New() *App {
	return NewWithDriver(NewDemoDriver())
}

func NewWithDriver(driver Driver) *App {
	return &App{
		driver:    driver,
		stream:    make(chan tui.StreamEvent, 512),
		input:     make(chan tcell.Event, 128),
		interrupt: interruptGate{window: interruptWindow},
	}
}

func (a *App) Run() error {
	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}
	defer s.Fini()
	a.screen = s

	a.pal = tui.DetectPalette(s)
	s.SetStyle(tcell.StyleDefault.Background(a.pal.Background).Foreground(a.pal.Foreground))
	s.EnableMouse()
	s.EnablePaste()

	sW, sH := s.Size()
	a.model = tui.NewModel("demo")
	a.layout = layouts.NewMainLayout(a.model, a.pal)

	if b, ok := a.driver.(Backend); ok {
		a.seedBackend(b)
	} else {
		info := a.layout.DefaultModel()
		if info.Name == "" {
			info = tui.ModelInfo{Name: "demo", Provider: "", Effort: "", ContextWindow: 0}
		}
		applyModel(a.model, info)
	}

	if d, ok := a.driver.(interface {
		ApprovalRequests() <-chan permissionRequest
	}); ok {
		a.permission = d.ApprovalRequests()
	}

	if cwd, err := os.Getwd(); err == nil {
		if fsys, err := filesystem.New(filesystem.Config{RootDir: cwd, IgnoreDotfiles: true}); err == nil {
			a.layout.Textarea().SetMentionCatalog(buildMentionCatalog(fsys))
			a.layout.Textarea().SetSkillCatalog(buildSkillCatalog(fsys))
		}
	}
	a.layout.Layout(image.Rect(0, 0, sW, sH))

	a.layout.Textarea().OnSubmit = a.submit
	a.layout.Viewport().OnToggle = a.toggleItem
	a.layout.OnAction = a.handleAction
	a.layout.OnSessionSelect = a.handleSessionSelect
	a.layout.OnModelSelect = a.handleModelSelect
	a.layout.OnUserAction = a.handleUserAction

	a.ctx, a.cancel = context.WithCancel(context.Background())
	defer a.cancel()
	a.start = time.Now()

	go func() {
		for {
			ev := s.PollEvent()
			if ev == nil {
				return
			}
			select {
			case a.input <- ev:
			case <-a.ctx.Done():
				return
			}
		}
	}()

	a.ticker = time.NewTicker(tickInterval)
	defer a.ticker.Stop()

	a.dirty = true
	for !a.done {
		select {
		case ev := <-a.input:
			a.handleInput(ev)

		case ev := <-a.stream:
			a.model.Apply(ev)
			if a.permModal != nil && (ev.Kind == tui.EventDone || ev.Kind == tui.EventError) {
				a.layout.CloseModal()
				a.permModal = nil
			}
			if ev.Kind == tui.EventDone || ev.Kind == tui.EventError {
				a.saveSession()
			}
			a.layout.NotifyModelChange()
			a.dirty = true

		case req := <-a.permission:
			a.showPermissionModal(req)

		case now := <-a.ticker.C:
			elapsed := now.Sub(a.start)
			a.interrupt.expire(now)
			if a.layout.Tick(elapsed) || a.model.Busy {
				a.dirty = true
			}

		case <-a.ctx.Done():
			a.done = true
		}

		if a.dirty {
			a.draw()
			a.dirty = false
		}
	}
	return nil
}

func (a *App) draw() {
	for i := 0; i < 4; i++ {
		if !a.layout.Render(a.screen, a.pal) {
			break
		}
	}
	a.screen.Show()
}

func (a *App) handleInput(ev tcell.Event) {
	a.dirty = true
	switch e := ev.(type) {
	case *tcell.EventKey:
		if e.Key() == tcell.KeyCtrlC {
			a.quit()
			return
		}
		if e.Key() == tcell.KeyEsc && a.model.Busy && a.layout.CanInterrupt() {
			if a.interrupt.press(time.Now()) {
				a.interruptRun()
			} else {
				a.model.SetStatusMsg("press Esc again to interrupt")
			}
		}
		a.layout.HandleEvent(e)
	case *tcell.EventResize:
		w, h := e.Size()
		a.layout.Layout(image.Rect(0, 0, w, h))
		a.screen.Sync()
	case *tcell.EventMouse:
		a.layout.HandleEvent(e)
	case *tcell.EventPaste:
		a.layout.HandleEvent(e)
	}
}

func (a *App) submit(text string) {
	if a.model.Busy {
		return
	}
	if a.backend != nil && a.model.Model == "" {
		a.model.SetStatusMsg("pick a model first: ⌃P → Change model")
		a.layout.NotifyModelChange()
		return
	}
	a.layout.Viewport().BackToMain()

	req := RunRequest{
		Text:    text,
		History: nil,
	}
	if a.backend != nil {
		req.History = a.backend.CurrentHistory(a.model.Conv)
	}
	if a.sessionID == "" && a.backend != nil {
		a.sessionID = a.backend.NewSessionID()
	}

	msg := a.model.Conv.AddUser(text)
	msg.Attachments = a.layout.AttachmentBar().TakeAll()
	req.Attachments = msg.Attachments
	a.layout.NotifyModelChange()

	runCtx := a.ctx
	if a.sessionID != "" {
		runCtx = agent.WithAgentContext(a.ctx, "orchestrator", a.sessionID)
	}
	go func() {
		a.driver.Run(runCtx, req, func(ev tui.StreamEvent) {
			select {
			case a.stream <- ev:
			case <-a.ctx.Done():
			}
		})
	}()
}

var errMentionWalkStop = errors.New("mention catalog cap reached")

const maxMentionEntries = 300

func buildMentionCatalog(fsys *filesystem.FileSystem) []components.Mention {
	out := components.DefaultMentions()
	count := 0
	fsys.Walk(context.Background(), func(rel string, info filesystem.FileInfo) error {
		if count >= maxMentionEntries {
			return errMentionWalkStop
		}
		count++
		kind := "file"
		if info.IsDir {
			kind = "dir"
		}
		out = append(out, components.Mention{ID: rel, Kind: kind, Label: rel})
		return nil
	})
	return out
}

func buildSkillCatalog(fsys *filesystem.FileSystem) []components.Mention {
	if fsys != nil {
		if mgr, err := skill.NewManager(fsys.Root(), fsys); err == nil {
			if skills, err := mgr.List(); err == nil && len(skills) > 0 {
				out := make([]components.Mention, 0, len(skills))
				for _, s := range skills {
					out = append(out, components.Mention{ID: s.Name, Kind: "skill", Label: s.Name})
				}
				return out
			}
		}
	}
	return []components.Mention{
		{ID: "graphify", Kind: "skill", Label: "graphify"},
		{ID: "impeccable", Kind: "skill", Label: "impeccable"},
		{ID: "refactor", Kind: "skill", Label: "refactor"},
		{ID: "review", Kind: "skill", Label: "review"},
	}
}

func applyModel(m *tui.Model, info tui.ModelInfo) {
	m.Model = info.Name
	m.Provider = info.Provider
	m.Effort = info.Effort
	m.ContextWindow = info.ContextWindow
}

func (a *App) handleAction(id string) {
	switch id {
	case "new-session", "clear-conversation":
		a.saveSession()
		a.model.Conv.Reset()
		a.model.SessionName = ""
		if a.backend != nil {
			a.sessionID = ""
		}
		a.layout.Viewport().BackToMain()
		a.layout.CloseAllModals()
		a.model.SetStatusMsg("new session")
	case "switch-session":
		a.refreshSessions()
		a.layout.CloseAllModals()
		a.layout.OpenSessionPicker()
	case "delete-session":
		a.layout.CloseAllModals()
		a.confirmDeleteCurrentSession()
	case "share-session":
		a.model.SetStatusMsg("share link copied")
	case "connect-provider":
		a.showProviderModal()
	case "change-model":
		a.layout.CloseAllModals()
		a.layout.OpenModelPicker()
	}
	a.layout.NotifyModelChange()
}

func (a *App) handleSessionSelect(id string) {
	if a.backend == nil {
		if s := a.layout.SessionByID(id); s != nil {
			a.model.SessionName = s.Title
			a.model.SetStatusMsg("switched session")
		}
		a.layout.NotifyModelChange()
		return
	}
	sess, err := a.backend.LoadSession(id)
	if err != nil {
		a.model.SetStatusMsg("failed to load session")
		a.layout.NotifyModelChange()
		return
	}
	MessagesToConv(a.model.Conv, sess.Messages)
	ApplySessionParts(a.model.Conv, sess.Parts)
	a.sessionID = sess.ID
	a.model.SessionName = sess.Title
	a.model.Usage = tui.Usage{}
	if sess.Metadata.Model != "" {
		a.model.Model = sess.Metadata.Model
		a.backend.SetModel(sess.Metadata.Model)
	}
	a.layout.Viewport().BackToMain()
	a.model.SetStatusMsg("session loaded")
	a.layout.NotifyModelChange()
}

func (a *App) confirmDeleteCurrentSession() {
	if a.sessionID == "" {
		a.model.SetStatusMsg("no session to delete")
		a.layout.NotifyModelChange()
		return
	}
	title := a.model.SessionName
	if title == "" {
		title = a.sessionID
	}
	confirm := components.NewConfirmModal("Delete session", "Delete current session '"+title+"'?")
	confirm.OnConfirm = func() {
		a.layout.CloseAllModals()
		if a.backend != nil {
			if err := a.backend.DeleteSession(a.sessionID); err != nil {
				a.model.SetStatusMsg("delete failed")
			} else {
				a.model.SetStatusMsg("session deleted")
			}
		}
		a.sessionID = ""
		a.model.Conv.Reset()
		a.model.SessionName = ""
		a.layout.Viewport().BackToMain()
		a.refreshSessions()
		a.layout.NotifyModelChange()
	}
	confirm.OnCancel = func() {
		a.layout.CloseModal()
		a.layout.NotifyModelChange()
	}
	a.layout.PushModal(confirm)
	a.layout.NotifyModelChange()
}

func (a *App) seedBackend(b Backend) {
	a.backend = b
	a.model.Model = ""
	a.refreshSessions()
	a.layout.SetModels(b.Models())
	if models := b.Models(); len(models) > 0 {
		applyModel(a.model, models[0])
		b.SetModel(models[0].Name)
	}
}

func (a *App) refreshSessions() {
	if a.backend == nil {
		return
	}
	sessions, err := a.backend.ListSessions()
	if err != nil {
		return
	}
	a.layout.SetSessions(sessions)
}

func (a *App) saveSession() {
	if a.backend == nil || len(a.model.Conv.Messages) == 0 {
		return
	}
	if a.sessionID == "" {
		a.sessionID = a.backend.NewSessionID()
	}
	msgs := ConvToMessages(a.model.Conv)
	title := a.model.SessionName
	if title == "" {
		for _, m := range a.model.Conv.Messages {
			if m.Role == tui.RoleUser {
				title = titleFromText(m.Content)
				break
			}
		}
	}
	a.model.SessionName = title
	sess := &session.Session{
		ID:    a.sessionID,
		Title: title,
		Metadata: session.SessionMetadata{
			Model:        a.model.Model,
			Provider:     a.model.Provider,
			MessageCount: len(msgs),
			TotalTokens:  a.model.Usage.TotalTokens,
			Agent:        "orchestrator",
		},
		Messages: msgs,
		Parts:    ConvToSessionParts(a.model.Conv),
	}
	if err := a.backend.SaveSession(sess); err == nil {
		a.refreshSessions()
	}
}

func (a *App) showPermissionModal(req permissionRequest) {
	m := components.NewPermissionModal(req.toolName, req.params, req.reason)
	m.OnDecision = func(d permission.Decision) {
		a.layout.CloseModal()
		a.permModal = nil
		req.resp <- d
	}
	a.permModal = m
	a.layout.PushModal(m)
	a.layout.NotifyModelChange()
}

func (a *App) showProviderModal() {
	var providers []string
	if a.backend != nil {
		providers = a.backend.SupportedProviders()
	}
	m := components.NewProviderModal(providers)
	m.OnSubmit = func(name, apiKey string) {
		a.layout.CloseModal()
		a.model.SetStatusMsg("connecting " + name + "…")
		a.layout.NotifyModelChange()
		go func() {
			var msg string
			if a.backend != nil {
				if err := a.backend.ConnectProvider(name, apiKey); err != nil {
					msg = "provider: " + err.Error()
				} else {
					msg = "provider " + name + " connected"
					a.layout.SetModels(a.backend.Models())
					if a.model.Model == "" {
						if models := a.backend.Models(); len(models) > 0 {
							a.handleModelSelect(models[0])
						}
					}
				}
			} else {
				msg = "provider unavailable (demo mode)"
			}
			a.model.SetStatusMsg(msg)
			a.layout.NotifyModelChange()
		}()
	}
	m.OnClose = func() {
		a.layout.CloseModal()
		a.layout.NotifyModelChange()
	}
	a.layout.PushModal(m)
	a.layout.NotifyModelChange()
}

func (a *App) handleModelSelect(info tui.ModelInfo) {
	applyModel(a.model, info)
	if a.backend != nil {
		a.backend.SetModel(info.Name)
	}
	a.model.SetStatusMsg("model: " + info.Name)
	a.layout.NotifyModelChange()
}

func (a *App) handleUserAction(actionID string, m *tui.Message) {
	switch actionID {
	case "fork", "revert":
		if a.model.Busy {
			a.model.SetStatusMsg("wait for the run to finish")
		} else {
			a.model.Conv.TruncateAfter(m)
			a.saveSession()
			a.model.SetStatusMsg(actionID + "ed session")
		}
	case "copy":
		a.layout.Textarea().SetClipboard(m.Content)
		a.model.SetStatusMsg("message copied")
	}
	a.layout.NotifyModelChange()
}

func (a *App) toggleItem(id string) {
	if a.model.Conv.TogglePartByID(id) {
		a.layout.NotifyModelChange()
	}
}

func (a *App) interruptRun() {
	a.driver.Cancel()
	a.stream = make(chan tui.StreamEvent, 512)
	a.model.Apply(tui.StreamEvent{Kind: tui.EventError, Error: errInterrupted})
	a.model.SetStatusMsg("interrupted")
	a.layout.NotifyModelChange()
}

func (a *App) quit() {
	a.driver.Cancel()
	a.saveSession()
	a.cancel()
	a.done = true
}
