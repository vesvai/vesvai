package tui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/permission"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layouts"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type App struct {
	screen         tcell.Screen
	layout         *layouts.ChatLayout
	agentService   *AgentService
	done           chan struct{}
	eventCh        chan tcell.Event
	tuiEventCh     chan TUIEvent
	streamingMsg   *components.StreamingMessage
	streamMu       sync.Mutex
	running        bool
	fps            int
	lastDraw       time.Time
	dirty          bool
	pasteBuffering bool
	pasteBuffer    []rune
	debugEnabled   bool
	cancelFunc     context.CancelFunc
	isPending      bool

	approvalMu    sync.Mutex
	approvalQueue []*ApprovalRequest
	approvalShown *ApprovalRequest
}

func New(cfg *config.Config, debugEnabled ...bool) (*App, error) {
	debug := false
	if len(debugEnabled) > 0 {
		debug = debugEnabled[0]
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %w", err)
	}

	screen.EnableMouse()
	screen.EnablePaste()
	screen.SetStyle(tcell.StyleDefault.Background(theme.BgPrimary).Foreground(theme.TextPrimary))
	screen.Clear()

	width, height := screen.Size()
	layout := layouts.NewChatLayout(width, height)
	layout.SetDebugEnabled(debug)

	svc, err := NewAgentService(cfg)
	if err != nil {
		screen.Fini()
		return nil, fmt.Errorf("failed to create agent service: %w", err)
	}

	app := &App{
		screen:       screen,
		layout:       layout,
		agentService: svc,
		done:         make(chan struct{}),
		eventCh:      make(chan tcell.Event, 20),
		tuiEventCh:   make(chan TUIEvent, 64),
		running:      false,
		fps:          60,
		debugEnabled: debug,
	}

	app.layout.SetOnSubmit(func(text string) {
		app.handleSubmit(text)
	})

	app.layout.SetOnCommandExecute(func(cmd components.Command) {
		app.handleCommand(cmd)
	})

	app.layout.Input().OnAttachmentsChange = func(atts []*components.Attachment) {
		app.layout.SetAttachments(atts)
	}

	app.layout.StatusBar().SetModel(svc.CurrentModel())
	app.layout.SetAvailableAgents(svc.AvailableAgents())

	svc.TUIEventAdapter().SetHandler(func(event TUIEvent) {
		select {
		case app.tuiEventCh <- event:
		case <-app.done:
		}
	})

	if err := svc.TUIEventAdapter().Start(); err != nil {
		screen.Fini()
		return nil, fmt.Errorf("failed to start TUI event adapter: %w", err)
	}

	go app.consumeTUIEvents()
	go app.consumeApprovalRequests()

	return app, nil
}

func (a *App) consumeApprovalRequests() {
	for {
		select {
		case req := <-a.agentService.ApprovalRequests():
			a.showApproval(&req)
		case <-a.done:
			return
		}
	}
}

func (a *App) showApproval(req *ApprovalRequest) {
	a.approvalMu.Lock()
	show := false
	if a.approvalShown == nil {
		a.approvalShown = req
		show = true
	} else {
		a.approvalQueue = append(a.approvalQueue, req)
	}
	a.approvalMu.Unlock()

	if show {
		a.layout.ApprovalPrompt().Show(req.ToolName, formatToolArgs(req.Args), req.Reason)
		a.markDirty()
	}
}

func (a *App) resolveApproval(decision permission.Decision) {
	a.approvalMu.Lock()
	req := a.approvalShown
	if req != nil {
		select {
		case req.Result <- decision:
		default:
		}
	}
	a.approvalShown = nil
	if len(a.approvalQueue) > 0 {
		a.approvalShown = a.approvalQueue[0]
		a.approvalQueue = a.approvalQueue[1:]
	}
	a.approvalMu.Unlock()

	if req != nil {
		a.layout.ApprovalPrompt().Hide()
	}
	if a.approvalShown != nil {
		a.layout.ApprovalPrompt().Show(a.approvalShown.ToolName, formatToolArgs(a.approvalShown.Args), a.approvalShown.Reason)
	}
	a.markDirty()
}

func (a *App) cancelPendingApprovals() {
	a.approvalMu.Lock()
	if a.approvalShown != nil {
		select {
		case a.approvalShown.Result <- permission.DecisionDeny:
		default:
		}
		a.approvalShown = nil
	}
	for _, req := range a.approvalQueue {
		select {
		case req.Result <- permission.DecisionDeny:
		default:
		}
	}
	a.approvalQueue = nil
	a.approvalMu.Unlock()
	a.layout.ApprovalPrompt().Hide()
}

func (a *App) debugLog(category, detail string, color tcell.Color) {
	if !a.debugEnabled {
		return
	}
	a.layout.DebugPanel().Add(category, detail, color)
}

func (a *App) consumeTUIEvents() {
	for {
		select {
		case event := <-a.tuiEventCh:
			a.streamMu.Lock()
			a.handleTUIEvent(event)
			a.streamMu.Unlock()
		case <-a.done:
			return
		}
	}
}

func (a *App) Run() error {
	a.running = true
	defer a.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		select {
		case <-sigCh:
			a.Stop()
		case <-a.done:
		}
	}()

	if a.debugEnabled {
		a.debugLog("INIT", "TUI started, debug enabled", theme.AccentGreen)
	}

	go a.pollEvents()

	a.draw()

	ticker := time.NewTicker(time.Second / time.Duration(a.fps))
	defer ticker.Stop()

	for {
		select {
		case <-a.done:
			return nil
		case <-ticker.C:
			a.draw()
		case ev := <-a.eventCh:
			a.handleEvent(ev)
		}
	}
}

func (a *App) Stop() {
	if a.running {
		a.running = false
		a.cancelPendingApprovals()
		a.agentService.TUIEventAdapter().Stop()
		a.agentService.Stop()
		a.screen.Fini()
		close(a.done)
	}
}

func (a *App) pollEvents() {
	for {
		ev := a.screen.PollEvent()
		if ev == nil {
			return
		}
		a.eventCh <- ev
	}
}

func (a *App) handleEvent(ev tcell.Event) {
	a.markDirty()
	defer func() {
		if r := recover(); r != nil {
			a.debugLog("PANIC", fmt.Sprintf("%v", r), theme.AccentRed)
			a.screen.Clear()
			a.draw()
		}
	}()

	a.debugLog("EVENT_RECV", formatEventDetail(ev), theme.AccentCyan)

	switch e := ev.(type) {
	case *tcell.EventResize:
		a.screen.Sync()
		width, height := a.screen.Size()
		a.layout.Resize(width, height)
		a.debugLog("RESIZE", fmt.Sprintf("%dx%d", width, height), theme.AccentAmber)

	case *tcell.EventKey:
		if a.pasteBuffering && e.Key() == tcell.KeyRune {
			a.pasteBuffer = append(a.pasteBuffer, e.Rune())
			return
		}
		a.handleKey(e)

	case *tcell.EventPaste:
		if e.Start() {
			a.pasteBuffering = true
			a.pasteBuffer = nil
			a.debugLog("PASTE_START", "paste buffer started", theme.AccentPurple)
		} else {
			a.pasteBuffering = false
			a.handlePaste(string(a.pasteBuffer))
			a.pasteBuffer = nil
		}

	case *tcell.EventMouse:
		a.handleMouse(e)
	}
}

func (a *App) handleKey(ev *tcell.EventKey) {
	a.debugLog("EVENT_KEY", formatKeyDetail(ev), theme.AccentCyan)

	if a.layout.ApprovalPrompt().IsVisible() {
		switch ev.Key() {
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'y', 'Y':
				a.resolveApproval(permission.DecisionAllow)
			case 'a', 'A':
				a.resolveApproval(permission.DecisionAllowAlways)
			case 'n', 'N':
				a.resolveApproval(permission.DecisionDeny)
			}
		case tcell.KeyEscape:
			a.resolveApproval(permission.DecisionDeny)
		}
		return
	}

	if a.layout.IsDebugPanelVisible() {
		a.layout.HandleEvent(ev)
		a.debugLog("EVENT_ROUTE", "→ debugPanel (consumed)", theme.AccentAmber)
		return
	}

	if a.layout.IsHelpVisible() {
		a.layout.HandleEvent(ev)
		a.debugLog("EVENT_ROUTE", "→ helpModal (consumed)", theme.AccentAmber)
		return
	}

	if a.layout.IsCommandPaletteVisible() {
		if ev.Key() == tcell.KeyEscape {
			a.layout.HideCommandPalette()
			a.debugLog("COMMAND", "palette closed (escape)", theme.AccentMagenta)
			return
		}
		a.layout.HandleEvent(ev)
		a.debugLog("EVENT_ROUTE", "→ commandPalette (consumed)", theme.AccentAmber)
		return
	}

	if a.layout.PreviewModal().IsVisible() {
		a.layout.HandleEvent(ev)
		a.debugLog("EVENT_ROUTE", "→ previewModal (consumed)", theme.AccentAmber)
		return
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		if a.layout.ErrorOverlay().IsVisible() {
			a.layout.ErrorOverlay().Hide()
			return
		}
		if a.cancelFunc != nil {
			a.debugLog("CANCEL", "cancel requested via Escape", theme.AccentAmber)
			a.cancelFunc()
			a.cancelFunc = nil
			a.agentService.CancelPending()
			a.streamMu.Lock()
			if a.streamingMsg != nil {
				a.streamingMsg.Finish()
				a.streamingMsg = nil
			}
			a.isPending = false
			a.streamMu.Unlock()
			return
		}
		a.debugLog("QUIT", "exit requested", theme.AccentRed)
		a.Stop()
		return

	case tcell.KeyCtrlC:
		a.debugLog("QUIT", "exit requested", theme.AccentRed)
		a.Stop()
		return

	case tcell.KeyCtrlP:
		a.layout.ToggleCommandPalette()
		a.debugLog("COMMAND", "palette toggled", theme.AccentMagenta)
		return

	case tcell.KeyF1:
		a.layout.ShowHelp()
		a.debugLog("COMMAND", "help shown", theme.AccentMagenta)
		return

	case tcell.KeyF2:
		a.layout.ToggleDebugPanel()
		a.debugLog("COMMAND", "debug panel toggled", theme.AccentAmber)
		return

	case tcell.KeyCtrlT:
		a.layout.ToggleTodoView()
		a.debugLog("COMMAND", "todo view toggled", theme.AccentAmber)
		return

	case tcell.KeyCtrlS:
		a.layout.ToggleSubagentPanel()
		a.debugLog("COMMAND", "subagent panel toggled", theme.AccentAmber)
		return

	case tcell.KeyCtrlN:
		a.handleCommand(components.Command{Label: "New Session"})
		return

	case tcell.KeyCtrlO:
		a.handleCommand(components.Command{Label: "Load Session"})
		return

	case tcell.KeyCtrlM:
		a.handleCommand(components.Command{Label: "Change Model"})
		return

	case tcell.KeyCtrlE:
		a.handleCommand(components.Command{Label: "Export Chat"})
		return
	}

	a.layout.HandleEvent(ev)
	a.debugLog("EVENT_ROUTE", "→ layout (default)", theme.TextDim)
}

func (a *App) handleMouse(ev *tcell.EventMouse) {
	a.debugLog("EVENT_MOUSE", formatMouseDetail(ev), theme.AccentCyan)
	a.layout.HandleEvent(ev)
}

func (a *App) markDirty() { a.dirty = true }

func (a *App) draw() {
	now := time.Now()
	if now.Sub(a.lastDraw) < time.Millisecond*8 {
		return
	}
	a.lastDraw = now

	a.streamMu.Lock()
	if !a.dirty {
		streaming := a.streamingMsg != nil && a.streamingMsg.IsStreaming()
		if !streaming {
			a.streamMu.Unlock()
			return
		}
	}
	a.dirty = false

	a.updateSubagentPanel()
	a.layout.Draw(a.screen)
	a.screen.Show()
	a.streamMu.Unlock()
}

func (a *App) updateSubagentPanel() {
	runner := a.agentService.Runner()
	if runner == nil {
		return
	}
	snapshots := runner.Subagents().All()
	if len(snapshots) == 0 {
		return
	}
	entries := make([]components.SubagentEntry, len(snapshots))
	for i, snap := range snapshots {
		entries[i] = components.SubagentEntry{
			ID:        snap.ID,
			Name:      snap.Name,
			Prompt:    snap.Prompt,
			Status:    string(snap.Status),
			Content:   snap.Content,
			Reasoning: snap.Reasoning,
			ToolCount: len(snap.ToolCalls),
		}
	}
	a.layout.SubagentPanel().SetEntries(entries)
}

func (a *App) handleSubmit(text string) {
	a.streamMu.Lock()
	if a.isPending {
		a.streamMu.Unlock()
		return
	}
	a.streamMu.Unlock()

	preview := text
	if len(preview) > 40 {
		preview = preview[:40] + "..."
	}
	a.debugLog("SUBMIT", fmt.Sprintf("%q", preview), theme.AccentGreen)

	atts := a.layout.Input().Attachments()
	if atts == nil {
		atts = []*components.Attachment{}
	}
	goAtts := make([]components.Attachment, len(atts))
	for i, att := range atts {
		goAtts[i] = *att
	}

	userMsg := components.NewUserMessage(text, goAtts)
	a.layout.AddMessage(userMsg)

	a.streamMu.Lock()
	a.streamingMsg = components.NewStreamingMessage()
	a.isPending = true
	a.streamMu.Unlock()
	a.layout.AddStreamingMessage(a.streamingMsg)

	a.layout.ClearAttachments()
	a.layout.Input().Clear()
	a.layout.Input().SetBlocked(true)
	a.layout.LoadingIndicator().Show()

	ctx, cancel := context.WithCancel(a.agentService.ctx)
	a.cancelFunc = cancel

	if err := a.agentService.SendWithContext(ctx, text, goAtts, func(chunk agent.StreamChunk) {
		a.streamMu.Lock()
		a.handleStreamChunk(chunk)
		a.streamMu.Unlock()
	}); err != nil {
		a.debugLog("ERROR", fmt.Sprintf("send failed: %v", err), theme.AccentRed)
		a.layout.ErrorOverlay().Show(agent.FormatError(err))
		a.streamMu.Lock()
		a.isPending = false
		a.streamingMsg = nil
		a.streamMu.Unlock()
		a.layout.Input().SetBlocked(false)
		a.layout.LoadingIndicator().Hide()
	}
}

func (a *App) handleStreamChunk(chunk agent.StreamChunk) {
	a.markDirty()
	if chunk.IsDone {
		a.debugLog("CHUNK_RECV", "done=true", theme.AccentGreen)
		if chunk.Content != "" {
			if a.streamingMsg == nil {
				a.streamingMsg = components.NewStreamingMessage()
				a.layout.AddStreamingMessage(a.streamingMsg)
			}
			a.streamingMsg.AppendContent(chunk.Content)
		}
		if a.streamingMsg != nil {
			a.streamingMsg.Finish()
			if a.streamingMsg.HasContent() {
				finalMsg := a.streamingMsg.Finalize()
				a.layout.MessageList().ClearStreaming()
				a.layout.AddMessage(finalMsg)
			}
			a.streamingMsg = nil
		}
		a.isPending = false
		a.layout.Input().SetBlocked(false)
		a.layout.LoadingIndicator().Hide()
		return
	}

	if chunk.Reasoning != "" {
		if a.streamingMsg == nil {
			a.streamingMsg = components.NewStreamingMessage()
			a.layout.AddStreamingMessage(a.streamingMsg)
		}
		a.streamingMsg.AppendThinking(chunk.Reasoning)
	}

	if chunk.Content != "" {
		a.layout.LoadingIndicator().Hide()
		detail := chunk.Content
		if len(detail) > 60 {
			detail = detail[:60] + "..."
		}
		a.debugLog("CHUNK_RECV", fmt.Sprintf("content=%q", detail), theme.AccentGreen)
		if a.streamingMsg == nil {
			a.streamingMsg = components.NewStreamingMessage()
			a.layout.AddStreamingMessage(a.streamingMsg)
		}
		a.streamingMsg.AppendContent(chunk.Content)
	}

	if chunk.ToolCall != nil {
		argsStr := formatToolArgs(chunk.ToolCall.Args)
		a.debugLog("CHUNK_RECV", fmt.Sprintf("tool_call=%s(%s)", chunk.ToolCall.ToolName, argsStr), theme.AccentAmber)
		if a.streamingMsg == nil {
			a.streamingMsg = components.NewStreamingMessage()
			a.layout.AddStreamingMessage(a.streamingMsg)
		}
		a.streamingMsg.AddToolCall(chunk.ToolCall.ToolName, chunk.ToolCall.Args)
	}

	if chunk.ToolResult != nil {
		success := chunk.ToolResult.Error == nil
		resultStr := chunk.ToolResult.Result
		if len(resultStr) > 200 {
			resultStr = resultStr[:200] + "..."
		}
		a.debugLog("CHUNK_RECV", fmt.Sprintf("tool_result=%s success=%v dur=%dms", chunk.ToolResult.ToolName, success, chunk.ToolResult.Duration), theme.AccentAmber)
		if a.streamingMsg == nil {
			a.streamingMsg = components.NewStreamingMessage()
			a.layout.AddStreamingMessage(a.streamingMsg)
		}
		a.streamingMsg.CompleteToolCall(chunk.ToolResult.ToolName, resultStr, success, chunk.ToolResult.Duration)

		if isTodoTool(chunk.ToolResult.ToolName) && success {
			a.layout.TodoView().HandleToolResult(chunk.ToolResult.ToolName, chunk.ToolResult.Result)
			a.layout.TodoView().Show()
		}
	}
}

func (a *App) handleTUIEvent(event TUIEvent) {
	a.markDirty()
	switch e := event.(type) {
	case TUIThinkingEvent:
		if a.streamingMsg == nil {
			a.streamingMsg = components.NewStreamingMessage()
			a.layout.AddStreamingMessage(a.streamingMsg)
		}
		a.streamingMsg.AppendThinking(e.Content)

	case TUIContentEvent:
		a.debugLog("TUI_EVENT", fmt.Sprintf("content=%q", truncate(e.Content, 60)), theme.AccentGreen)
		if a.streamingMsg == nil {
			a.streamingMsg = components.NewStreamingMessage()
			a.layout.AddStreamingMessage(a.streamingMsg)
		}
		a.streamingMsg.AppendContent(e.Content)
		a.layout.LoadingIndicator().Hide()

	case TUIToolCallEvent:
		if a.streamingMsg == nil {
			a.streamingMsg = components.NewStreamingMessage()
			a.layout.AddStreamingMessage(a.streamingMsg)
		}
		a.streamingMsg.AddToolCall(e.ToolName, e.Args)

	case TUIToolDoneEvent:
		success := e.Error == nil
		resultStr := e.Result
		if len(resultStr) > 200 {
			resultStr = resultStr[:200] + "..."
		}
		if a.streamingMsg == nil {
			a.streamingMsg = components.NewStreamingMessage()
			a.layout.AddStreamingMessage(a.streamingMsg)
		}
		a.streamingMsg.CompleteToolCall(e.ToolName, resultStr, success, e.Duration)

	case TUIDoneEvent:
		a.debugLog("TUI_EVENT", "done", theme.AccentGreen)
		if a.streamingMsg != nil {
			a.streamingMsg.Finish()
			if a.streamingMsg.HasContent() {
				finalMsg := a.streamingMsg.Finalize()
				a.layout.MessageList().ClearStreaming()
				a.layout.AddMessage(finalMsg)
			}
			a.streamingMsg = nil
		}
		a.isPending = false
		a.layout.Input().SetBlocked(false)
		a.layout.LoadingIndicator().Hide()

	case TUIErrorEvent:
		a.debugLog("TUI_EVENT", fmt.Sprintf("error=%v", e.Error), theme.AccentRed)
		a.layout.ErrorOverlay().Show(agent.FormatError(e.Error))
		a.streamingMsg = nil
		a.isPending = false
		a.layout.Input().SetBlocked(false)
		a.layout.LoadingIndicator().Hide()
	}
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func isTodoTool(name string) bool {
	switch name {
	case "set-todo", "get-todo", "list-todos", "update-todo", "delete-todo":
		return true
	}
	return false
}

func formatToolArgs(args map[string]any) string {
	if args == nil {
		return ""
	}
	parts := make([]string, 0, len(args))
	for k, v := range args {
		val := fmt.Sprintf("%v", v)
		if len(val) > 50 {
			val = val[:50] + "..."
		}
		parts = append(parts, k+"="+val)
	}
	return fmt.Sprintf("%v", parts)
}

func (a *App) handlePaste(text string) {
	filePath := components.LookupFileFromText(text)
	if filePath != "" {
		att, err := components.NewFileAttachment(filePath)
		if err == nil {
			a.debugLog("PASTE", fmt.Sprintf("file=%s", filePath), theme.AccentPurple)
			existing := a.layout.Attachments()
			if existing == nil {
				existing = []*components.Attachment{}
			}
			existing = append(existing, att)
			a.layout.SetAttachments(existing)
			return
		}
	}

	if len(text) > 50 {
		a.debugLog("PASTE", fmt.Sprintf("text_attachment len=%d", len(text)), theme.AccentPurple)
		att := components.NewTextAttachment(text)
		existing := a.layout.Attachments()
		if existing == nil {
			existing = []*components.Attachment{}
		}
		existing = append(existing, att)
		a.layout.SetAttachments(existing)
		return
	}

	a.debugLog("PASTE", fmt.Sprintf("typed len=%d", len(text)), theme.AccentPurple)
	for _, r := range text {
		a.layout.Input().HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
}

func (a *App) handleCommand(cmd components.Command) {
	a.debugLog("COMMAND", cmd.Label, theme.AccentMagenta)
	switch cmd.Label {
	case "New Session":
		a.agentService.NewSession()
		a.layout.Input().Clear()
		a.layout.MessageList().Clear()
		a.layout.ShowTitleScreen()
		a.streamMu.Lock()
		a.streamingMsg = nil
		a.streamMu.Unlock()

	case "Load Session":
		sessions, err := a.agentService.ListSessions()
		if err != nil || len(sessions) == 0 {
			return
		}
		var entries []components.SessionEntry
		for _, s := range sessions {
			entries = append(entries, components.SessionEntry{
				ID:           s.ID,
				Title:        s.Title,
				MessageCount: s.MessageCount,
				Model:        s.Model,
				UpdatedAt:    s.UpdatedAt,
			})
		}
		a.layout.SessionPicker().SetSessions(entries)
		a.layout.SessionPicker().OnSelect = func(id string) {
			a.debugLog("COMMAND", fmt.Sprintf("session loaded id=%s", id), theme.AccentMagenta)
			sess, err := a.agentService.LoadSession(id)
			if err != nil {
				return
			}
			a.layout.Input().Clear()
			a.layout.MessageList().Clear()
			a.streamMu.Lock()
			a.streamingMsg = nil
			a.streamMu.Unlock()
			for _, msg := range sess.Messages {
				switch msg.Role {
				case "user":
					content := extractStringContent(msg.Content)
					a.layout.AddMessage(components.NewUserMessage(content, nil))
				case "assistant":
					content := extractStringContent(msg.Content)
					if content != "" {
						a.layout.AddMessage(components.NewAssistantMessage(content))
					}
				}
			}
			a.layout.StatusBar().SetModel(a.agentService.CurrentModel())
		}
		a.layout.SessionPicker().Show()

	case "Delete Session":
		sid := a.agentService.CurrentSessionID()
		if sid != "" {
			a.agentService.DeleteSession(sid)
		}
		a.agentService.NewSession()
		a.layout.Input().Clear()
		a.layout.MessageList().Clear()
		a.layout.ShowTitleScreen()
		a.streamMu.Lock()
		a.streamingMsg = nil
		a.streamMu.Unlock()

	case "Change Model":
		models := a.agentService.AvailableModels()
		if len(models) > 0 {
			var entries []components.ModelEntry
			for _, m := range models {
				entries = append(entries, components.ModelEntry{
					Provider: "openrouter",
					ModelID:  m,
				})
			}
			a.layout.ModelSelector().SetModels(entries)
			a.layout.ModelSelector().SetCurrentModel(a.agentService.CurrentModel())
			a.layout.ModelSelector().OnSelect = func(model string) {
				a.debugLog("COMMAND", fmt.Sprintf("model changed to %s", model), theme.AccentMagenta)
				a.agentService.SetModel(model)
				a.layout.StatusBar().SetModel(model)
			}
			a.layout.ModelSelector().Show()
		}

	case "Show Help":
		a.layout.ShowHelp()

	case "Export Chat":
		a.exportChatToFile()

	case "Quit":
		a.Stop()

	default:
		a.layout.Input().Clear()
	}
}

func (a *App) exportChatToFile() {
	sess := a.agentService.CurrentSession()
	if sess == nil || len(sess.Messages) == 0 {
		a.layout.ErrorOverlay().Show("No conversation to export")
		return
	}

	var sb strings.Builder
	sb.WriteString("# Vesvai Chat Export\n\n")
	for _, msg := range sess.Messages {
		switch msg.Role {
		case "user":
			sb.WriteString("## You\n\n")
		case "assistant":
			sb.WriteString("## Vesvai\n\n")
		default:
			sb.WriteString("## " + string(msg.Role) + "\n\n")
		}
		content := extractStringContent(msg.Content)
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	filename := fmt.Sprintf("vesvai-export-%s.md", time.Now().Format("2006-01-02-150405"))
	if err := os.WriteFile(filename, []byte(sb.String()), 0644); err != nil {
		a.layout.ErrorOverlay().Show(fmt.Sprintf("Failed to export: %v", err))
		return
	}
	a.debugLog("EXPORT", "exported to "+filename, theme.AccentGreen)
}

func (a *App) Layout() *layouts.ChatLayout {
	return a.layout
}

func (a *App) Screen() tcell.Screen {
	return a.screen
}

func formatEventDetail(ev tcell.Event) string {
	switch e := ev.(type) {
	case *tcell.EventKey:
		return formatKeyDetail(e)
	case *tcell.EventMouse:
		return formatMouseDetail(e)
	case *tcell.EventResize:
		w, h := e.Size()
		return fmt.Sprintf("resize %dx%d", w, h)
	case *tcell.EventPaste:
		if e.Start() {
			return "paste start"
		}
		return "paste end"
	default:
		return fmt.Sprintf("%T", ev)
	}
}

func formatKeyDetail(ev *tcell.EventKey) string {
	key := ev.Key()
	runeVal := ev.Rune()
	mods := ev.Modifiers()

	var parts []string

	switch {
	case key == tcell.KeyRune:
		if runeVal == ' ' {
			parts = append(parts, "key=Space")
		} else if runeVal < 32 || runeVal > 126 {
			parts = append(parts, fmt.Sprintf("key=Rune(0x%04x)", runeVal))
		} else {
			parts = append(parts, fmt.Sprintf("key='%c'", runeVal))
		}
	case key == tcell.KeyEnter:
		parts = append(parts, "key=Enter")
	case key == tcell.KeyEscape:
		parts = append(parts, "key=Escape")
	case key == tcell.KeyBackspace || key == tcell.KeyBackspace2:
		parts = append(parts, "key=Backspace")
	case key == tcell.KeyDelete:
		parts = append(parts, "key=Delete")
	case key == tcell.KeyTab:
		parts = append(parts, "key=Tab")
	case key == tcell.KeyUp:
		parts = append(parts, "key=Up")
	case key == tcell.KeyDown:
		parts = append(parts, "key=Down")
	case key == tcell.KeyLeft:
		parts = append(parts, "key=Left")
	case key == tcell.KeyRight:
		parts = append(parts, "key=Right")
	case key == tcell.KeyPgUp:
		parts = append(parts, "key=PgUp")
	case key == tcell.KeyPgDn:
		parts = append(parts, "key=PgDn")
	case key == tcell.KeyHome:
		parts = append(parts, "key=Home")
	case key == tcell.KeyEnd:
		parts = append(parts, "key=End")
	case key == tcell.KeyF1:
		parts = append(parts, "key=F1")
	case key == tcell.KeyF2:
		parts = append(parts, "key=F2")
	case key == tcell.KeyF3:
		parts = append(parts, "key=F3")
	case key == tcell.KeyF4:
		parts = append(parts, "key=F4")
	case key == tcell.KeyF5:
		parts = append(parts, "key=F5")
	case key == tcell.KeyF6:
		parts = append(parts, "key=F6")
	case key == tcell.KeyF7:
		parts = append(parts, "key=F7")
	case key == tcell.KeyF8:
		parts = append(parts, "key=F8")
	case key == tcell.KeyF9:
		parts = append(parts, "key=F9")
	case key == tcell.KeyF10:
		parts = append(parts, "key=F10")
	case key == tcell.KeyF11:
		parts = append(parts, "key=F11")
	case key == tcell.KeyF12:
		parts = append(parts, "key=F12")
	case key == tcell.KeyCtrlA:
		parts = append(parts, "key=Ctrl+A")
	case key == tcell.KeyCtrlB:
		parts = append(parts, "key=Ctrl+B")
	case key == tcell.KeyCtrlC:
		parts = append(parts, "key=Ctrl+C")
	case key == tcell.KeyCtrlD:
		parts = append(parts, "key=Ctrl+D")
	case key == tcell.KeyCtrlE:
		parts = append(parts, "key=Ctrl+E")
	case key == tcell.KeyCtrlF:
		parts = append(parts, "key=Ctrl+F")
	case key == tcell.KeyCtrlN:
		parts = append(parts, "key=Ctrl+N")
	case key == tcell.KeyCtrlO:
		parts = append(parts, "key=Ctrl+O")
	case key == tcell.KeyCtrlP:
		parts = append(parts, "key=Ctrl+P")
	case key == tcell.KeyCtrlS:
		parts = append(parts, "key=Ctrl+S")
	default:
		parts = append(parts, fmt.Sprintf("key=%d", key))
	}

	if mods&tcell.ModShift != 0 {
		parts = append(parts, "shift")
	}
	if mods&tcell.ModCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	if mods&tcell.ModAlt != 0 {
		parts = append(parts, "alt")
	}

	return strings.Join(parts, " ")
}

func formatMouseDetail(ev *tcell.EventMouse) string {
	x, y := ev.Position()
	buttons := ev.Buttons()

	var parts []string
	parts = append(parts, fmt.Sprintf("pos=%d,%d", x, y))

	if buttons&tcell.Button1 != 0 {
		parts = append(parts, "btn=left")
	}
	if buttons&tcell.Button2 != 0 {
		parts = append(parts, "btn=right")
	}
	if buttons&tcell.Button3 != 0 {
		parts = append(parts, "btn=middle")
	}
	if buttons&tcell.WheelUp != 0 {
		parts = append(parts, "wheel=up")
	}
	if buttons&tcell.WheelDown != 0 {
		parts = append(parts, "wheel=down")
	}
	if buttons == 0 {
		parts = append(parts, "btn=none")
	}

	mods := ev.Modifiers()
	if mods&tcell.ModShift != 0 {
		parts = append(parts, "shift")
	}
	if mods&tcell.ModCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	if mods&tcell.ModAlt != 0 {
		parts = append(parts, "alt")
	}

	return strings.Join(parts, " ")
}

func extractStringContent(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case map[string]any:
		if text, ok := c["text"].(string); ok {
			return text
		}
	}
	return fmt.Sprintf("%v", content)
}
