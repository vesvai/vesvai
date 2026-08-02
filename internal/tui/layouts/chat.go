package layouts

import (
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type ChatLayout struct {
	title              *components.Title
	textarea           *components.Textarea
	messageList        *components.MessageList
	statusBar          *components.StatusBar
	commandPalette     *components.CommandPalette
	attachmentBar      *components.AttachmentBar
	previewModal       *components.PreviewModal
	actionList         *components.ActionList
	modelSelector      *components.ModelSelector
	sessionPicker      *components.SessionPicker
	helpModal          *components.HelpModal
	debugPanel         *components.DebugPanel
	errorOverlay       *components.ErrorOverlay
	todoView           *components.TodoView
	subagentPanel      *components.SubagentPanel
	approvalPrompt     *components.ApprovalPrompt
	loadingIndicator   *components.LoadingIndicator
	screenWidth        int
	screenHeight       int
	showTitle          bool
	OnCommandExecute   func(command components.Command)
	suppressActionList bool
	availableAgents    []string
	debugEnabled       bool
}

func NewChatLayout(screenWidth, screenHeight int) *ChatLayout {
	inputWidth := screenWidth - 4
	if inputWidth > components.MaxInputWidth {
		inputWidth = components.MaxInputWidth
	}

	cl := &ChatLayout{
		title:            components.NewTitle(),
		textarea:         components.NewTextarea("Ask VESVAI to build, refactor or investigate... @mention /skill", inputWidth, nil),
		messageList:      components.NewMessageList(),
		statusBar:        components.NewStatusBar(),
		commandPalette:   components.NewCommandPalette(),
		attachmentBar:    components.NewAttachmentBar(),
		previewModal:     components.NewPreviewModal(),
		actionList:       components.NewActionList(),
		modelSelector:    components.NewModelSelector(),
		sessionPicker:    components.NewSessionPicker(),
		helpModal:        components.NewHelpModal(),
		debugPanel:       components.NewDebugPanel(),
		errorOverlay:     components.NewErrorOverlay(),
		todoView:         components.NewTodoView(),
		subagentPanel:    components.NewSubagentPanel(),
		approvalPrompt:   components.NewApprovalPrompt(),
		loadingIndicator: components.NewLoadingIndicator(),
		screenWidth:      screenWidth,
		screenHeight:     screenHeight,
		showTitle:        true,
	}

	cl.textarea.SetWidth(inputWidth)
	cl.commandPalette.SetScreenSize(screenWidth, screenHeight)
	cl.previewModal.SetScreenSize(screenWidth, screenHeight)
	cl.actionList.SetScreenSize(screenWidth, screenHeight)
	cl.modelSelector.SetScreenSize(screenWidth, screenHeight)
	cl.sessionPicker.SetScreenSize(screenWidth, screenHeight)
	cl.helpModal.SetScreenSize(screenWidth, screenHeight)
	cl.debugPanel.SetScreenSize(screenWidth, screenHeight)

	cl.actionList.OnSelect = func(action components.Action) {
		cl.suppressActionList = true
		text := cl.textarea.Text()
		if len(text) > 0 && text[0] == '/' {
			cl.textarea.SetCommittedText("/" + action.Name)
		} else {
			atPos := -1
			for i, r := range text {
				if r == '@' {
					atPos = i
				}
			}
			if atPos >= 0 {
				endPos := len(text)
				for i := atPos + 1; i < len(text); i++ {
					if text[i] == ' ' {
						endPos = i
						break
					}
				}
				newText := string(text[:atPos]) + "@" + action.Name + string(text[endPos:])
				cl.textarea.SetCommittedText(newText)
			}
		}
		cl.suppressActionList = false
	}

	cl.textarea.OnTextChange = func() {
		cl.updateActionListVisibility()
	}

	return cl
}

func (cl *ChatLayout) SetOnSubmit(fn func(string)) {
	cl.textarea.OnSubmit = fn
}

func (cl *ChatLayout) OnSubmit(text string) {
	cl.textarea.OnSubmit(text)
}

func (cl *ChatLayout) AddMessage(msg *components.Message) {
	cl.messageList.AddMessage(msg)
	cl.hideTitle()
}

func (cl *ChatLayout) AddStreamingMessage(sm *components.StreamingMessage) {
	cl.messageList.AddStreamingMessage(sm)
	cl.hideTitle()
}

func (cl *ChatLayout) UpdateLastMessage(msg *components.Message) {
	cl.messageList.UpdateLastMessage(msg)
}

func (cl *ChatLayout) RemoveLastMessage() {
	cl.messageList.RemoveLastMessage()
}

func (cl *ChatLayout) hideTitle() {
	cl.showTitle = false
	cl.title.Hide()
}

func (cl *ChatLayout) showTitleScreen() {
	cl.showTitle = true
	cl.title.Show()
}

func (cl *ChatLayout) ShowTitleScreen() {
	cl.showTitle = true
	cl.title.Show()
}

func (cl *ChatLayout) HandleEvent(ev tcell.Event) bool {
	if cl.errorOverlay.IsVisible() {
		if cl.errorOverlay.HandleEvent(ev) {
			return true
		}
	}

	if cl.subagentPanel.IsVisible() {
		if cl.subagentPanel.HandleEvent(ev) {
			return true
		}
	}

	if cl.helpModal.IsVisible() {
		return cl.helpModal.HandleEvent(ev)
	}

	if cl.sessionPicker.IsVisible() {
		return cl.sessionPicker.HandleEvent(ev)
	}

	if cl.modelSelector.IsVisible() {
		return cl.modelSelector.HandleEvent(ev)
	}

	if cl.commandPalette.IsVisible() {
		return cl.commandPalette.HandleEvent(ev)
	}

	if cl.previewModal.IsVisible() {
		return cl.previewModal.HandleEvent(ev)
	}

	if cl.debugPanel.IsVisible() {
		return cl.debugPanel.HandleEvent(ev)
	}

	if cl.actionList.IsVisible() {
		if cl.actionList.HandleEvent(ev) {
			return true
		}
	}

	switch e := ev.(type) {
	case *tcell.EventMouse:
		return cl.messageList.HandleEvent(e)
	case *tcell.EventKey:
		if cl.attachmentBar.IsFocused() {
			handled := cl.attachmentBar.HandleEvent(e)
			if handled {
				if e.Key() == tcell.KeyEnter {
					att := cl.attachmentBar.SelectedAttachment()
					if att != nil {
						cl.previewModal.Show(att)
					}
				}
				return true
			}
			cl.textarea.Focus()
			return cl.textarea.HandleEvent(e)
		}

		if e.Key() == tcell.KeyTab {
			if cl.textarea.HandleEvent(ev) {
				cl.updateActionListVisibility()
				return true
			}
			if cl.attachmentBar.Len() > 0 {
				if cl.attachmentBar.FocusFirst() {
					cl.textarea.Blur()
					return true
				}
			}
			return true
		}

		if cl.textarea.HandleEvent(ev) {
			cl.updateActionListVisibility()
			return true
		}
		return cl.messageList.HandleEvent(ev)
	}
	return false
}

func (cl *ChatLayout) textareaYPosition() int {
	headerHeight := cl.statusBar.Height()
	textareaHeight := cl.textarea.Height()
	if cl.showTitle {
		welcomeHeight := cl.screenHeight - headerHeight - textareaHeight - 2
		titleHeight := cl.title.Height()
		titleY := (welcomeHeight-titleHeight)/2 + headerHeight + 1
		return titleY + titleHeight + 3
	}
	return cl.screenHeight - cl.statusBar.Height() - textareaHeight
}

func (cl *ChatLayout) buildMentionItems() []components.Action {
	var items []components.Action

	sanitize := func(s string) string {
		return strings.ReplaceAll(s, " ", "_")
	}

	agentIcons := map[string]rune{
		"orchestrator": '◆',
		"explorer":     '✦',
		"planner":      '◇',
	}
	defaultIcon := '●'

	for _, name := range cl.availableAgents {
		icon := defaultIcon
		if ic, ok := agentIcons[name]; ok {
			icon = ic
		}
		items = append(items, components.Action{
			Category:    "Agents",
			Name:        name,
			Label:       name,
			Description: "Agent: " + name,
			Icon:        icon,
		})
	}

	hasAttachments := false
	for _, att := range cl.textarea.Attachments() {
		if !hasAttachments {
			hasAttachments = true
		}
		name := sanitize(att.Name)
		items = append(items, components.Action{
			Category:    "Attachments",
			Name:        name,
			Label:       name,
			Description: "Attached " + att.TypeLabel(),
			Icon:        []rune(att.Icon())[0],
		})
	}

	wd, err := os.Getwd()
	if err == nil {
		entries, err := os.ReadDir(wd)
		if err == nil {
			count := 0
			for _, entry := range entries {
				if count >= 20 {
					break
				}
				if strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				desc := "File"
				if entry.IsDir() {
					desc = "Directory"
				}
				name := sanitize(entry.Name())
				items = append(items, components.Action{
					Category:    "Files",
					Name:        name,
					Label:       name,
					Description: "Workspace " + desc,
					Icon:        '○',
				})
				count++
			}
		}
	}

	return items
}

func (cl *ChatLayout) updateActionListVisibility() {
	if cl.suppressActionList {
		return
	}
	text := cl.textarea.Text()
	if len(text) > 0 && text[0] == '/' {
		for _, r := range text[1:] {
			if r == ' ' {
				if cl.actionList.IsVisible() {
					cl.actionList.Hide()
				}
				return
			}
		}
		boxWidth := cl.textarea.BoxWidth(cl.screenWidth)
		boxX := render.CenterX(boxWidth, cl.screenWidth)
		textareaY := cl.textareaYPosition()
		cl.actionList.SetTextareaBox(boxX, boxWidth)
		cl.actionList.SetTextareaY(textareaY)
		cl.actionList.SetActions(components.DefaultActions())
		cl.actionList.UpdateFilter(string(text[1:]))
		if !cl.actionList.IsVisible() {
			cl.actionList.Show()
		}
		return
	}

	atPos := -1
	for i, r := range text {
		if r == '@' {
			atPos = i
		}
	}
	if atPos >= 0 {
		query := text[atPos+1:]
		for _, r := range query {
			if r == ' ' {
				if cl.actionList.IsVisible() {
					cl.actionList.Hide()
				}
				return
			}
		}
		boxWidth := cl.textarea.BoxWidth(cl.screenWidth)
		boxX := render.CenterX(boxWidth, cl.screenWidth)
		textareaY := cl.textareaYPosition()
		cl.actionList.SetTextareaBox(boxX, boxWidth)
		cl.actionList.SetTextareaY(textareaY)
		cl.actionList.SetActions(cl.buildMentionItems())
		cl.actionList.UpdateFilter(query)
		if !cl.actionList.IsVisible() {
			cl.actionList.Show()
		}
		return
	}

	if cl.actionList.IsVisible() {
		cl.actionList.Hide()
	}
}

func (cl *ChatLayout) ToggleCommandPalette() {
	if cl.commandPalette.IsVisible() {
		cl.commandPalette.Hide()
	} else {
		cl.commandPalette.Show()
	}
}

func (cl *ChatLayout) ShowCommandPalette() {
	cl.commandPalette.Show()
}

func (cl *ChatLayout) HideCommandPalette() {
	cl.commandPalette.Hide()
}

func (cl *ChatLayout) IsCommandPaletteVisible() bool {
	return cl.commandPalette.IsVisible()
}

func (cl *ChatLayout) CommandPalette() *components.CommandPalette {
	return cl.commandPalette
}

func (cl *ChatLayout) SetOnCommandExecute(fn func(command components.Command)) {
	cl.OnCommandExecute = fn
	cl.commandPalette.OnExecute = func(cmd components.Command) {
		if cl.OnCommandExecute != nil {
			cl.OnCommandExecute(cmd)
		}
	}
}

func (cl *ChatLayout) AttachmentBar() *components.AttachmentBar {
	return cl.attachmentBar
}

func (cl *ChatLayout) PreviewModal() *components.PreviewModal {
	return cl.previewModal
}

func (cl *ChatLayout) SetAttachments(atts []*components.Attachment) {
	cl.textarea.SetAttachments(atts)
	cl.attachmentBar.SetAttachments(atts)
}

func (cl *ChatLayout) Attachments() []*components.Attachment {
	return cl.textarea.Attachments()
}

func (cl *ChatLayout) ClearAttachments() {
	cl.textarea.SetAttachments(nil)
	cl.attachmentBar.Clear()
}

func (cl *ChatLayout) Resize(width, height int) {
	cl.screenWidth = width
	cl.screenHeight = height

	inputWidth := width - 4
	if inputWidth > components.MaxInputWidth {
		inputWidth = components.MaxInputWidth
	}
	cl.textarea.SetWidth(inputWidth)
	cl.commandPalette.SetScreenSize(width, height)
	cl.previewModal.SetScreenSize(width, height)
	cl.actionList.SetScreenSize(width, height)
	cl.modelSelector.SetScreenSize(width, height)
	cl.sessionPicker.SetScreenSize(width, height)
	cl.helpModal.SetScreenSize(width, height)
	cl.debugPanel.SetScreenSize(width, height)
}

func (cl *ChatLayout) Draw(s tcell.Screen) {
	s.Clear()

	width := cl.screenWidth
	height := cl.screenHeight

	bgStyle := tcell.StyleDefault.Background(theme.BgPrimary)
	render.FillArea(s, 0, 0, width, height, bgStyle)

	headerHeight := cl.statusBar.Height()
	textareaHeight := cl.textarea.Height()
	attBarHeight := 0
	if cl.attachmentBar.Len() > 0 {
		attBarHeight = cl.attachmentBar.Height()
	}

	var textareaY int
	if cl.showTitle {
		welcomeHeight := height - headerHeight - textareaHeight - 2
		textareaY = cl.textareaYPosition()
		cl.drawWelcomeScreen(s, width, height, headerHeight, textareaHeight, welcomeHeight)
	} else {
		chatHeight := height - headerHeight - textareaHeight - attBarHeight - 2
		textareaY = cl.textareaYPosition()
		cl.drawChatScreen(s, width, height, headerHeight, textareaHeight, chatHeight)
	}

	if attBarHeight > 0 {
		attBarY := textareaY - attBarHeight
		boxWidth := cl.textarea.BoxWidth(width)
		boxStartX := render.CenterX(boxWidth, width)
		cl.attachmentBar.SetCardBox(boxStartX, boxWidth)
		cl.attachmentBar.Draw(s, attBarY, width)
	}

	cl.statusBar.Draw(s, 0, width)

	cl.errorOverlay.Draw(s, 0, width, height-headerHeight-textareaHeight-attBarHeight-2)

	if cl.todoView.IsVisible() && cl.todoView.TotalCount() > 0 {
		todoHeight := cl.todoView.Height(width)
		todoY := 0
		cl.todoView.Draw(s, todoY, width)
		_ = todoHeight
	}

	if cl.subagentPanel.IsVisible() && len(cl.subagentPanel.Entries()) > 0 {
		cl.subagentPanel.Draw(s, 0, width)
	}

	cl.commandPalette.Draw(s)
	cl.previewModal.Draw(s)
	cl.actionList.Draw(s)
	cl.modelSelector.Draw(s)
	cl.sessionPicker.Draw(s)
	cl.helpModal.Draw(s)
	cl.debugPanel.Draw(s)

	cl.approvalPrompt.Draw(s, width, height)
}

func (cl *ChatLayout) drawWelcomeScreen(s tcell.Screen, width, height, headerHeight, textareaHeight, availableHeight int) {
	titleHeight := cl.title.Height()
	titleY := headerHeight + (availableHeight-titleHeight)/2 + 1

	cl.title.Draw(s, titleY, width)

	textareaY := titleY + titleHeight + 3
	cl.textarea.Draw(s, textareaY, width)
}

func (cl *ChatLayout) drawChatScreen(s tcell.Screen, width, height, headerHeight, textareaHeight, availableHeight int) {
	cl.messageList.SetScreenWidth(width)
	cl.messageList.SetMaxVisible(availableHeight)

	messageY := headerHeight
	cl.messageList.Draw(s, messageY, width, availableHeight)

	textareaY := height - cl.statusBar.Height() - textareaHeight
	cl.textarea.Draw(s, textareaY, width)

	if cl.loadingIndicator.IsVisible() {
		loadingY := textareaY + 1
		cl.loadingIndicator.Draw(s, loadingY, width)
	}
}

func (cl *ChatLayout) Input() *components.Textarea {
	return cl.textarea
}

func (cl *ChatLayout) MessageList() *components.MessageList {
	return cl.messageList
}

func (cl *ChatLayout) StatusBar() *components.StatusBar {
	return cl.statusBar
}

func (cl *ChatLayout) ModelSelector() *components.ModelSelector {
	return cl.modelSelector
}

func (cl *ChatLayout) SetAvailableAgents(agents []string) {
	cl.availableAgents = agents
}

func (cl *ChatLayout) SessionPicker() *components.SessionPicker {
	return cl.sessionPicker
}

func (cl *ChatLayout) HelpModal() *components.HelpModal {
	return cl.helpModal
}

func (cl *ChatLayout) ShowHelp() {
	cl.helpModal.Show()
}

func (cl *ChatLayout) HideHelp() {
	cl.helpModal.Hide()
}

func (cl *ChatLayout) IsHelpVisible() bool {
	return cl.helpModal.IsVisible()
}

func (cl *ChatLayout) DebugPanel() *components.DebugPanel {
	return cl.debugPanel
}

func (cl *ChatLayout) ErrorOverlay() *components.ErrorOverlay {
	return cl.errorOverlay
}

func (cl *ChatLayout) TodoView() *components.TodoView {
	return cl.todoView
}

func (cl *ChatLayout) ToggleTodoView() {
	cl.todoView.Toggle()
}

func (cl *ChatLayout) SubagentPanel() *components.SubagentPanel {
	return cl.subagentPanel
}

func (cl *ChatLayout) ToggleSubagentPanel() {
	cl.subagentPanel.Toggle()
}

func (cl *ChatLayout) ApprovalPrompt() *components.ApprovalPrompt {
	return cl.approvalPrompt
}

func (cl *ChatLayout) LoadingIndicator() *components.LoadingIndicator {
	return cl.loadingIndicator
}

func (cl *ChatLayout) ToggleDebugPanel() {
	if cl.debugPanel.IsVisible() {
		cl.debugPanel.Hide()
	} else {
		cl.debugPanel.Show()
	}
}

func (cl *ChatLayout) ShowDebugPanel() {
	cl.debugPanel.Show()
}

func (cl *ChatLayout) HideDebugPanel() {
	cl.debugPanel.Hide()
}

func (cl *ChatLayout) IsDebugPanelVisible() bool {
	return cl.debugPanel.IsVisible()
}

func (cl *ChatLayout) SetDebugEnabled(enabled bool) {
	cl.debugEnabled = enabled
}

func (cl *ChatLayout) DebugEnabled() bool {
	return cl.debugEnabled
}
