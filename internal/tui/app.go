package tui

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/core/update"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/skill"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/page/home"
	"github.com/vesvai/vesvai/internal/tui/page/settings"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

const blinkInterval = 450 * time.Millisecond

type redrawRequest struct{ tcell.EventTime }

type App struct {
	bus     event.Bus
	deps    settings.Deps
	agent   *agent.Agent
	screen  tcell.Screen
	root    components.Component
	overlay components.Component

	overlayMu sync.RWMutex
	chatMu    sync.Mutex

	home  *home.Page
	chat  *components.Chat
	model selectedModel

	main        *agentTranscript
	subs        map[string]*agentTranscript
	subItemByID map[string]*components.ChatItem
	viewID      string
	running     bool
	usage       llm.Usage
	history     []llm.Message
	session     *activeSession
	loadedFloor int

	ctx    context.Context
	cancel context.CancelFunc
	blink  bool
	quit   bool

	pasteActive bool
	pasteBuffer strings.Builder
}

type selectedModel struct {
	provider string
	model    llm.Model
}

type activeSession struct {
	info settings.SessionInfo
}

func (a *App) modelDisplay() string {
	if a.model.provider == "" {
		return ""
	}
	name := a.model.model.ID
	if a.model.model.Name != "" {
		name = a.model.model.Name
	}
	return a.model.provider + "/" + name
}

func (a *App) selectPreferred() {
	if a.deps.LLM == nil {
		return
	}
	res := a.deps.LLM.Select(llm.SelectRequest{Mode: llm.SelectModePreferred})
	if res.Err != nil {
		return
	}
	a.model = selectedModel{provider: res.Provider, model: res.Model}
}

func (a *App) setOverlay(c components.Component) {
	a.overlayMu.Lock()
	a.overlay = c
	a.overlayMu.Unlock()
}

func (a *App) getOverlay() components.Component {
	a.overlayMu.RLock()
	defer a.overlayMu.RUnlock()
	return a.overlay
}

func (a *App) requestRedraw() {
	ev := &redrawRequest{}
	ev.SetEventNow()
	_ = a.screen.PostEvent(ev)
}

func Run(bus event.Bus, deps settings.Deps) error {
	setBus(bus)

	s, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("tui: create screen: %w", err)
	}
	if err := s.Init(); err != nil {
		return fmt.Errorf("tui: init screen: %w", err)
	}
	defer s.Fini()

	ctx, cancel := context.WithCancel(context.Background())
	a := &App{
		bus:         bus,
		deps:        deps,
		agent:       deps.Agent,
		screen:      s,
		subs:        make(map[string]*agentTranscript),
		subItemByID: make(map[string]*components.ChatItem),
		ctx:         ctx,
		cancel:      cancel,
	}
	return a.start()
}

func (a *App) start() error {
	if a.bus != nil {
		if err := a.subscribeChat(a.bus); err != nil {
			return err
		}
		defer a.unsubscribeChat(a.bus)
	}
	if a.cancel == nil {
		a.ctx, a.cancel = context.WithCancel(context.Background())
	}
	defer a.cancel()

	a.build()
	dispatchReady()
	defer dispatchQuit()
	return a.loop()
}

func (a *App) build() {
	styles.RegisterDefaults()

	a.selectPreferred()

	go a.checkForUpdates()

	if a.agent != nil {
		a.viewID = a.agent.ID
	}

	page := home.New()
	a.home = page
	a.chat = page.Chat()
	page.Input().OnSubmit = a.submitMessage
	page.Input().Focus()
	page.SetModel(a.modelDisplay())

	a.chat.SetOnActivate(a.activateItem)
	a.chat.SetOnBack(a.backFromSubagent)
	a.chat.SetOnLoadMore(a.loadMore)

	var skillItems []components.ListItem
	for _, sk := range skill.List() {
		skillItems = append(skillItems, components.ListItem{Label: sk.Name, Detail: sk.Description})
	}
	page.SetSkills(skillItems)

	var mentionItems []components.ListItem
	for _, name := range agents.List() {
		a, err := agents.New(name)
		desc := ""
		if err == nil && a != nil {
			desc = a.Description
		}
		mentionItems = append(mentionItems, components.ListItem{Label: name, Detail: desc, Data: "agent"})
	}
	if fs := a.deps.VFS; fs != nil {
		filePaths, _ := fs.Glob("**", "")
		sort.Strings(filePaths)
		for _, p := range filePaths {
			mentionItems = append(mentionItems, components.ListItem{Label: p, Detail: "file", Data: "file"})
		}
	}
	page.SetMentionItems(mentionItems)

	comps := componentHook.Apply([]components.Component{page})
	if len(comps) > 0 {
		a.root = comps[0]
		if hp, ok := comps[0].(*home.Page); ok {
			a.home = hp
			a.chat = hp.Chat()
		}
	}
	a.refreshHomeLocked()

	th := styles.Current()
	a.screen.SetStyle(th.Base())
	a.screen.EnableMouse()
	a.screen.EnablePaste()
	a.draw()
}

func (a *App) checkForUpdates() {
	current := config.AppVersion
	ctx := context.Background()
	rel, found, err := update.DetectLatest(ctx)
	if err != nil || !found {
		return
	}

	if !update.IsNewerThan(current, rel.Version) {
		return
	}

	dismissed := update.GetDismissedVersion(a.deps.Cache)
	if dismissed == rel.Version {
		return
	}

	modal := NewUpdateModal(rel.Version,
		func() {
			a.setOverlay(nil)
			if err := update.UpdateToLatest(ctx); err != nil {
				a.showError("Update failed: " + err.Error())
				return
			}
			os.Exit(0)
		},
		func() {
			update.SetDismissedVersion(a.deps.Cache, rel.Version)
			a.setOverlay(nil)
		},
	)

	a.setOverlay(modal)
	a.requestRedraw()
}

func (a *App) loop() error {
	ticker := time.NewTicker(blinkInterval)
	defer ticker.Stop()

	stopTick := make(chan struct{})
	defer close(stopTick)
	go func() {
		for {
			select {
			case <-stopTick:
				return
			case <-ticker.C:
				ev := &tcell.EventTime{}
				ev.SetEventNow()
				_ = a.screen.PostEvent(ev)
			}
		}
	}()

	for {
		ev := a.screen.PollEvent()
		if ev == nil {
			return nil
		}
		switch e := ev.(type) {
		case *tcell.EventResize:
			a.screen.Sync()
			a.draw()
		case *tcell.EventPaste:
			if a.getOverlay() != nil {
				break
			}
			if e.Start() {
				a.pasteActive = true
				a.pasteBuffer.Reset()
			} else {
				a.pasteActive = false
				if a.pasteBuffer.Len() > 0 {
					a.handleFilePaste(a.pasteBuffer.String())
				}
				a.pasteBuffer.Reset()
			}
		case *tcell.EventKey:
			if a.pasteActive {
				if r := e.Rune(); r != 0 {
					a.pasteBuffer.WriteRune(r)
				}
				continue
			}
			if a.handleKey(e) {
				a.draw()
			}
		case *tcell.EventTime:
			a.blink = !a.blink
			if t, ok := a.root.(components.Ticker); ok {
				t.OnTick(a.blink)
			}
			a.draw()
		case *redrawRequest:
			a.draw()
		case *tcell.EventMouse:
			x, y := e.Position()
			switch e.Buttons() {
			case tcell.WheelUp:
				if a.getOverlay() == nil && a.home != nil {
					if a.home.HandleScroll(-3) {
						a.draw()
					}
				}
			case tcell.WheelDown:
				if a.getOverlay() == nil && a.home != nil {
					if a.home.HandleScroll(3) {
						a.draw()
					}
				}
			case tcell.ButtonPrimary:
				if a.getOverlay() == nil && a.home != nil {
					w, h := a.screen.Size()
					if a.home.HandleClick(x, y, w, h) {
						a.draw()
					}
				}
			}
		}
		if a.quit {
			return nil
		}
	}
}

func (a *App) handleKey(ev *tcell.EventKey) bool {
	ke := dispatchKey(ev)
	if ke.Consumed {
		return true
	}
	kev := tcell.NewEventKey(ke.Key, ke.Rune, ke.Mod)

	if ov := a.getOverlay(); ov != nil {
		if resolveGlobal(kev) == ActionQuit {
			a.quit = true
			return false
		}
		if ov.HandleKey(kev) {
			return true
		}
		return true
	}

	if a.root != nil && a.root.HandleKey(kev) {
		return true
	}
	switch resolveGlobal(kev) {
	case ActionQuit:
		a.quit = true
	case ActionThemeNext:
		styles.Next()
		return true
	case ActionSettings:
		a.openSettings()
		return true
	}
	return false
}

func (a *App) openSettings() {
	if a.getOverlay() != nil {
		return
	}
	s := settings.New(a.deps)
	s.SetSelectedModel(a.model.provider, a.model.model)
	s.SetOnModelChange(func(provider string, model llm.Model) {
		a.chatMu.Lock()
		a.model = selectedModel{provider: provider, model: model}
		a.refreshHomeLocked()
		a.chatMu.Unlock()
	})
	if a.session != nil {
		info := a.session.info
		s.SetActiveSession(&info)
	}
	s.SetOnSessionChange(func(info settings.SessionInfo) {
		a.chatMu.Lock()
		a.session = &activeSession{info: info}
		a.loadSessionIntoChatLocked()
		a.refreshHomeLocked()
		a.chatMu.Unlock()
	})
	s.SetOnSessionClear(func() {
		a.chatMu.Lock()
		a.session = nil
		a.history = nil
		a.chat.Clear()
		a.main = nil
		a.refreshHomeLocked()
		a.chatMu.Unlock()
	})
	s.SetOnClose(func() { a.setOverlay(nil) })
	a.setOverlay(s)
}

func (a *App) refreshHomeLocked() {
	if a.home == nil {
		return
	}
	a.home.SetModel(a.modelDisplay())
	a.home.SetRunning(a.running)
	a.home.SetUsage(a.usage)
	if a.model.model.Config != nil {
		a.home.SetMaxInputTokens(a.model.model.MaxInputTokens())
	}
	if a.session == nil {
		a.home.SetSession("")
		return
	}
	a.home.SetSession(a.session.info.Title)
}

func (a *App) loadSessionIntoChatLocked() {
	if a.session == nil {
		return
	}
	if a.main == nil {
		id := ""
		if a.agent != nil {
			id = a.agent.ID
		}
		a.main = newTranscript(id, "orchestrator")
	}
	msgs := a.session.info.Messages
	if len(msgs) > initialChatMessages {
		msgs = msgs[len(msgs)-initialChatMessages:]
	}
	items := messagesToItems(msgs)
	a.main.items = items
	a.loadedFloor = seqFloor(msgs)
	a.chat.SetHasMore(len(a.session.info.Messages) > len(msgs))
	mainID := ""
	if a.agent != nil {
		mainID = a.agent.ID
	}
	if a.viewID == "" || a.viewID == mainID {
		a.showTranscript(a.main)
		a.chat.SetBack(false)
	}
	a.seedHistoryFromSessionLocked()
}

func (a *App) seedHistoryFromSessionLocked() {
	msgs := a.session.info.Messages
	h := make([]llm.Message, 0, len(msgs)+1)
	if a.agent != nil && a.agent.SystemPrompt != "" {
		h = append(h, llm.SystemMessage(a.agent.SystemPrompt))
	}
	h = append(h, session.MessagesToLLM(msgs)...)
	a.history = h
}

const initialChatMessages = 50

func seqFloor(msgs []session.Message) int {
	floor := 0
	for _, m := range msgs {
		if floor == 0 || m.Seq < floor {
			floor = m.Seq
		}
	}
	return floor
}

func messageText(m session.Message) string {
	switch c := m.Content.(type) {
	case string:
		return c
	case []any:
		for _, item := range c {
			if mm, ok := item.(map[string]any); ok {
				if t, ok := mm["type"].(string); ok && t == "text" {
					if text, ok := mm["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

func (a *App) draw() {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	a.drawLocked()
}

func (a *App) drawLocked() {
	w, h := a.screen.Size()
	if w < 1 || h < 1 {
		return
	}
	bounds := layout.Region{Left: 0, Top: 0, Width: w, Height: h}
	if a.root != nil {
		a.root.Draw(a.screen, bounds, true)
	}
	if ov := a.getOverlay(); ov != nil {
		ov.Draw(a.screen, bounds, true)
	}
	a.screen.Show()
}

func (a *App) handleFilePaste(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		line := lines[0]
		if _, err := os.Stat(line); err == nil {
			a.tryAttachFile(line)
			return
		}
		if len(line) > components.LongTextThreshold {
			a.chatMu.Lock()
			if a.home != nil {
				a.home.Input().InsertLongText(line)
			}
			a.chatMu.Unlock()
			a.requestRedraw()
			return
		}
		a.pasteShortText(text)
		return
	}

	anyAttached := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, err := os.Stat(line); err == nil {
			a.tryAttachFile(line)
			anyAttached = true
		}
	}
	if !anyAttached {
		if len(text) > components.LongTextThreshold {
			a.chatMu.Lock()
			if a.home != nil {
				a.home.Input().InsertLongText(text)
			}
			a.chatMu.Unlock()
			a.requestRedraw()
		} else {
			a.pasteShortText(text)
		}
	}
}

func (a *App) pasteShortText(text string) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	if a.home == nil {
		return
	}
	input := a.home.Input()
	for _, r := range text {
		input.InsertRune(r)
	}
	a.requestRedraw()
}

func (a *App) tryAttachFile(path string) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}

	if !a.validateFileForModel(path) {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	mediaType := mime.TypeByExtension(filepath.Ext(path))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	var attType llm.AttachmentType
	switch {
	case strings.HasPrefix(mediaType, "image/"):
		attType = llm.AttachmentTypeImage
	case strings.HasPrefix(mediaType, "audio/"):
		attType = llm.AttachmentTypeAudio
	default:
		attType = llm.AttachmentTypeFile
	}

	att := llm.NewFileAttachmentFromBase64(mediaType, llm.EncodeFileToBase64(data), filepath.Base(path))
	att.Type = attType

	a.chatMu.Lock()
	if a.home != nil {
		a.home.AttachmentBar().Add(att)
		a.refreshHomeLocked()
	}
	a.chatMu.Unlock()
	a.requestRedraw()
}

func (a *App) validateFileForModel(path string) bool {
	if a.model.model.Config == nil {
		return true
	}

	mediaType := mime.TypeByExtension(filepath.Ext(path))
	config := a.model.model.Config

	switch {
	case strings.HasPrefix(mediaType, "image/"):
		if config.Modalities == nil || !strings.Contains(strings.Join(config.Modalities.Input, ","), "image") {
			a.showError("Model does not support image attachments")
			return false
		}
	case strings.HasPrefix(mediaType, "audio/"):
		if config.Modalities == nil || !strings.Contains(strings.Join(config.Modalities.Input, ","), "audio") {
			a.showError("Model does not support audio attachments")
			return false
		}
	}
	return true
}

func (a *App) showError(msg string) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	if a.main != nil {
		a.appendItem(a.main, &components.ChatItem{Kind: components.ItemError, Text: msg})
		a.requestRedraw()
	}
}
