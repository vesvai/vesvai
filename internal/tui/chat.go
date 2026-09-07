package tui

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"
	"sort"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/page/settings"
)

type agentTranscript struct {
	id        string
	name      string
	items     []*components.ChatItem
	thinking  *components.ChatItem
	assistant *components.ChatItem
	toolByID  map[string]*components.ChatItem
	started   bool
	done      bool
	err       string
}

func newTranscript(id, name string) *agentTranscript {
	return &agentTranscript{id: id, name: name, toolByID: make(map[string]*components.ChatItem)}
}

func (a *App) transcriptFor(id, name string) *agentTranscript {
	if a.agent != nil && id == a.agent.ID {
		if a.main == nil {
			a.main = newTranscript(id, name)
		}
		return a.main
	}
	if t, ok := a.subs[id]; ok {
		return t
	}
	t := newTranscript(id, name)
	a.subs[id] = t
	return t
}

func (a *App) appendItem(t *agentTranscript, it *components.ChatItem) {
	if t == nil {
		return
	}
	if a.viewID == t.id {
		a.chat.AppendItem(it)
		t.items = a.chat.Items()
	} else {
		t.items = append(t.items, it)
	}
}

func (a *App) showTranscript(t *agentTranscript) {
	a.viewID = t.id
	a.chat.SetItems(t.items)
	t.items = a.chat.Items()
	a.chat.Invalidate()
	a.requestRedraw()
}

func (a *App) showMain() {
	a.showTranscript(a.main)
	a.chat.SetBack(false)
}

func (a *App) refreshChat() {
	a.chat.Invalidate()
	a.requestRedraw()
}

func (a *App) subscribeChat(bus event.Bus) error {
	subs := []struct {
		topic string
		fn    any
	}{
		{agent.TopicAgentInput, a.onAgentInput},
		{agent.TopicAgentStarted, a.onAgentStarted},
		{agent.TopicAgentToken, a.onAgentToken},
		{agent.TopicAgentMessage, a.onAgentMessage},
		{agent.TopicAgentToolCall, a.onAgentToolCall},
		{agent.TopicAgentToolResult, a.onAgentToolResult},
		{agent.TopicAgentUsage, a.onAgentUsage},
		{agent.TopicAgentFinished, a.onAgentFinished},
		{agent.TopicAgentError, a.onAgentError},
		{session.TopicSessionAttached, a.onSessionAttached},
	}
	for _, s := range subs {
		if err := bus.Subscribe(s.topic, s.fn); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) unsubscribeChat(bus event.Bus) {
	subs := []struct {
		topic string
		fn    any
	}{
		{agent.TopicAgentInput, a.onAgentInput},
		{agent.TopicAgentStarted, a.onAgentStarted},
		{agent.TopicAgentToken, a.onAgentToken},
		{agent.TopicAgentMessage, a.onAgentMessage},
		{agent.TopicAgentToolCall, a.onAgentToolCall},
		{agent.TopicAgentToolResult, a.onAgentToolResult},
		{agent.TopicAgentUsage, a.onAgentUsage},
		{agent.TopicAgentFinished, a.onAgentFinished},
		{agent.TopicAgentError, a.onAgentError},
		{session.TopicSessionAttached, a.onSessionAttached},
	}
	for _, s := range subs {
		_ = bus.Unsubscribe(s.topic, s.fn)
	}
}

func (a *App) onAgentInput(e agent.AgentInput) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	t := a.transcriptFor(e.AgentID, e.AgentName)
	it := &components.ChatItem{Kind: components.ItemUser, ID: e.AgentID, Text: e.Input, Attachments: e.Attachments}
	a.appendItem(t, it)
	if e.AgentID != a.agent.ID {
		if sub := a.subItemByID[e.AgentID]; sub != nil {
			sub.SubagentTask = e.Input
		}
	}
	a.refreshChat()
}

func (a *App) onAgentStarted(e agent.AgentStarted) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	t := a.transcriptFor(e.AgentID, e.AgentName)
	t.started = true
	if e.AgentID == a.agent.ID {
		a.running = true
		a.refreshHomeLocked()
		return
	}
	if a.main == nil {
		a.main = newTranscript(a.agent.ID, "orchestrator")
	}
	sub := &components.ChatItem{
		Kind:           components.ItemSubagent,
		ID:             e.AgentID,
		AgentID:        e.AgentID,
		SubagentName:   e.AgentName,
		SubagentStatus: "running",
	}
	a.subItemByID[e.AgentID] = sub
	a.appendItem(a.main, sub)
	a.refreshChat()
}

func (a *App) onAgentToken(e agent.AgentToken) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	t := a.transcriptFor(e.AgentID, e.AgentName)
	if e.Reasoning != "" {
		if t.thinking == nil {
			t.thinking = &components.ChatItem{Kind: components.ItemThinking, ID: e.AgentID}
			a.appendItem(t, t.thinking)
		}
		t.thinking.Reasoning += e.Reasoning
	}
	if e.Content != "" {
		if t.assistant == nil {
			t.assistant = &components.ChatItem{Kind: components.ItemAssistant, ID: e.AgentID}
			a.appendItem(t, t.assistant)
		}
		t.assistant.Text += e.Content
	}
	a.refreshChat()
}

func (a *App) onAgentMessage(e agent.AgentMessage) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	t := a.transcriptFor(e.AgentID, e.AgentName)
	msg := e.Message

	t.thinking = nil

	if t.assistant != nil {
		t.assistant = nil
		return
	}

	if text := messageContent(msg); text != "" {
		it := &components.ChatItem{Kind: components.ItemAssistant, ID: e.AgentID, Text: text}
		a.appendItem(t, it)
	}
	a.refreshChat()
}

func (a *App) onAgentToolCall(e agent.AgentToolCall) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	t := a.transcriptFor(e.AgentID, e.AgentName)
	a.addToolItem(t, e.Call, e.AgentID)
	if e.AgentID != a.agent.ID {
		if sub := a.subItemByID[e.AgentID]; sub != nil {
			sub.SubagentActivity = formatActivity(e.Call)
		}
	}
	a.refreshChat()
}

func formatActivity(call llm.ToolCall) string {
	name := call.Function.Name
	args := call.Function.Arguments
	switch name {
	case "read":
		var p struct{ FilePath string `json:"filePath"` }
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.FilePath != "" {
			return "Reading " + p.FilePath
		}
	case "write":
		var p struct{ FilePath string `json:"filePath"` }
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.FilePath != "" {
			return "Writing " + p.FilePath
		}
	case "edit":
		var p struct{ FilePath string `json:"filePath"` }
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.FilePath != "" {
			return "Editing " + p.FilePath
		}
	case "bash":
		var p struct{ Command string `json:"command"` }
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Command != "" {
			cmd := p.Command
			if len(cmd) > 50 {
				cmd = cmd[:50] + "…"
			}
			return "Running " + cmd
		}
	case "glob":
		var p struct{ Pattern string `json:"pattern"` }
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Pattern != "" {
			return "Finding " + p.Pattern
		}
	case "grep":
		var p struct{ Pattern string `json:"pattern"` }
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Pattern != "" {
			return "Searching " + p.Pattern
		}
	case "list":
		var p struct{ Path string `json:"path"` }
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Path != "" {
			return "Listing " + p.Path
		}
	case "webfetch":
		var p struct{ URL string `json:"url"` }
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.URL != "" {
			return "Fetching " + p.URL
		}
	}
	return name
}

func (a *App) onAgentToolResult(e agent.AgentToolResult) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	t := a.transcriptFor(e.AgentID, e.AgentName)
	it, ok := t.toolByID[e.CallID]
	if !ok {
		it = &components.ChatItem{
			Kind:     components.ItemTool,
			ID:       e.CallID,
			ToolName: e.ToolName,
		}
		t.toolByID[e.CallID] = it
		a.appendItem(t, it)
	}
	if e.Err != nil {
		it.ToolErr = e.Err.Error()
	} else {
		it.ToolOutput = e.Output
	}
	a.refreshChat()
}

func (a *App) onAgentFinished(e agent.AgentFinished) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	t := a.transcriptFor(e.AgentID, e.AgentName)
	t.done = true
	t.thinking = nil
	t.assistant = nil
	if e.AgentID == a.agent.ID {
		a.running = false
		a.refreshHomeLocked()
		a.appendItem(t, &components.ChatItem{Kind: components.ItemFinished, ID: e.AgentID})
		return
	}
	if sub := a.subItemByID[e.AgentID]; sub != nil {
		sub.SubagentStatus = "finished"
		sub.SubagentOutput = e.Output
		sub.SubagentActivity = ""
		sub.SubagentUsage = e.Usage
	}
	a.appendItem(t, &components.ChatItem{Kind: components.ItemFinished, ID: e.AgentID})
	a.refreshChat()
}

func (a *App) onAgentError(e agent.AgentError) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	t := a.transcriptFor(e.AgentID, e.AgentName)
	t.done = true
	t.err = e.Err.Error()
	t.thinking = nil
	t.assistant = nil
	if e.AgentID == a.agent.ID {
		a.running = false
		a.refreshHomeLocked()
		a.appendItem(t, &components.ChatItem{Kind: components.ItemError, Text: e.Err.Error()})
		return
	}
	if sub := a.subItemByID[e.AgentID]; sub != nil {
		sub.SubagentStatus = "error"
		sub.SubagentActivity = ""
		sub.SubagentOutput = e.Err.Error()
	}
	a.appendItem(t, &components.ChatItem{Kind: components.ItemError, Text: e.Err.Error()})
	a.refreshChat()
}

func (a *App) onAgentUsage(e agent.AgentUsage) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	if a.agent == nil {
		return
	}
	if e.AgentID == a.agent.ID {
		a.usage.PromptTokens = e.Usage.PromptTokens
		a.usage.CompletionTokens = e.Usage.CompletionTokens
		a.usage.TotalTokens = e.Usage.TotalTokens
		a.usage.Cost = e.Usage.Cost
		a.refreshHomeLocked()
		return
	}
	if sub := a.subItemByID[e.AgentID]; sub != nil {
		sub.SubagentUsage = e.Usage
		a.refreshChat()
	}
}

func (a *App) onSessionAttached(e session.SessionAttached) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	if e.AgentID != a.agent.ID {
		return
	}
	if a.session == nil {
		a.session = &activeSession{info: settings.SessionInfo{ID: e.SessionID}}
		a.refreshHomeLocked()
	}
}

func (a *App) addToolItem(t *agentTranscript, call llm.ToolCall, agentID string) {
	it := &components.ChatItem{
		Kind:     components.ItemTool,
		ID:       call.ID,
		ToolName: call.Function.Name,
		ToolArgs: call.Function.Arguments,
	}

	name := call.Function.Name
	args := call.Function.Arguments

	switch name {
	case "edit":
		var p struct {
			FilePath  string `json:"filePath"`
			OldString string `json:"oldString"`
			NewString string `json:"newString"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.OldString != "" {
			it.Diff = components.ComputeDiff(p.OldString, p.NewString)
			if p.FilePath != "" {
				it.ToolName = "edit:" + p.FilePath
			}
		}
	case "read":
		var p struct {
			FilePath string `json:"filePath"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.FilePath != "" {
			it.ToolName = "read:" + p.FilePath
		}
	case "write":
		var p struct {
			FilePath string `json:"filePath"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.FilePath != "" {
			it.ToolName = "write:" + p.FilePath
		}
	case "bash":
		var p struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Command != "" {
			cmd := p.Command
			if len(cmd) > 60 {
				cmd = cmd[:60] + "…"
			}
			it.ToolName = "bash:" + cmd
		}
	case "glob":
		var p struct {
			Pattern string `json:"pattern"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Pattern != "" {
			it.ToolName = "glob:" + p.Pattern
		}
	case "grep":
		var p struct {
			Pattern string `json:"pattern"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Pattern != "" {
			it.ToolName = "grep:" + p.Pattern
		}
	case "webfetch":
		var p struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.URL != "" {
			it.ToolName = "webfetch:" + p.URL
		}
	case "list":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Path != "" {
			it.ToolName = "list:" + p.Path
		}
	}

	t.toolByID[call.ID] = it
	a.appendItem(t, it)
}

func (a *App) submitMessage(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	a.chatMu.Lock()
	attachments := make([]llm.Attachment, len(a.home.AttachmentBar().Attachments()))
	copy(attachments, a.home.AttachmentBar().Attachments())
	a.home.AttachmentBar().Clear()
	a.refreshMentionItemsLocked()
	a.refreshHomeLocked()
	a.chatMu.Unlock()

	dispatchSubmit(input)

	a.chatMu.Lock()
	if a.running || a.agent == nil {
		a.chatMu.Unlock()
		return
	}
	a.running = true
	a.refreshHomeLocked()
	a.chatMu.Unlock()

	go a.runAgentWithAttachments(input, attachments)
}

func (a *App) runAgent(input string) {
	a.chatMu.Lock()
	attachments := make([]llm.Attachment, len(a.home.AttachmentBar().Attachments()))
	copy(attachments, a.home.AttachmentBar().Attachments())
	a.home.AttachmentBar().Clear()
	a.refreshMentionItemsLocked()
	a.refreshHomeLocked()
	a.chatMu.Unlock()
	a.runAgentWithAttachments(input, attachments)
}

func (a *App) runAgentWithAttachments(input string, attachments []llm.Attachment) {
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	orch := a.agent
	a.chatMu.Lock()
	if orch.Bus == nil {
		orch.Bus = a.bus
	}
	if orch.Provider == nil || orch.Model.ID == "" {
		if prov, err := a.deps.LLM.Provider(a.model.provider); err == nil {
			orch.Provider = prov
			orch.Model = a.model.model
		}
	}
	orch.ReasoningEffort = a.reasoningEffort
	orch.Attachments = attachments
	history := a.history
	if a.session != nil {
		a.bus.Publish(session.TopicSessionResume, session.SessionResume{
			AgentID:   orch.ID,
			SessionID: a.session.info.ID,
		})
	}
	a.chatMu.Unlock()

	var result *agent.RunResult
	handler := func(agent.StreamEvent) error { return nil }
	if len(history) > 0 {
		result, _ = orch.ResumeStream(ctx, input, history, handler)
	} else {
		result, _ = orch.RunStream(ctx, input, handler)
	}

	a.chatMu.Lock()
	if result != nil {
		a.history = result.History
	}
	a.running = false
	a.chatMu.Unlock()
	a.refreshChat()
}

func (a *App) activateItem(it *components.ChatItem) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	switch it.Kind {
	case components.ItemThinking, components.ItemTool:
		it.Expanded = !it.Expanded
		a.refreshChat()
	case components.ItemSubagent:
		it.Expanded = !it.Expanded
		a.refreshChat()
	}
}

func (a *App) openSubagentHistory(agentID string) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	if t := a.subs[agentID]; t != nil {
		a.showTranscript(t)
		a.chat.SetBack(true)
	}
}

func (a *App) backFromSubagent() {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	if a.main != nil {
		a.showMain()
	}
}

func (a *App) loadMore() {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	mainID := ""
	if a.agent != nil {
		mainID = a.agent.ID
	}
	if !a.chat.HasMore() || a.session == nil || a.deps.Sessions == nil || a.viewID != mainID {
		return
	}
	msgs, err := a.deps.Sessions.Messages(a.session.info.ID)
	if err != nil {
		a.chat.SetHasMore(false)
		return
	}
	var older []session.Message
	for _, m := range msgs {
		if m.Seq < a.loadedFloor {
			older = append(older, m)
		}
	}
	if len(older) == 0 {
		a.chat.SetHasMore(false)
		return
	}
	sort.Slice(older, func(i, j int) bool { return older[i].Seq < older[j].Seq })
	const batch = 50
	moreBelow := len(older) > batch
	if moreBelow {
		older = older[len(older)-batch:]
	}
	a.loadedFloor = older[0].Seq
	a.chat.SetHasMore(moreBelow)
	items := messagesToItems(older)
	if len(items) > 0 {
		a.main.items = append(items, a.main.items...)
		a.showTranscript(a.main)
		a.chat.SetBack(false)
	}
}

func messagesToItems(msgs []session.Message) []*components.ChatItem {
	var out []*components.ChatItem
	toolByID := make(map[string]*components.ChatItem)

	for _, m := range msgs {
		switch m.Role {
		case llm.RoleUser:
			out = append(out, &components.ChatItem{Kind: components.ItemUser, Text: messageText(m)})
		case llm.RoleAssistant:
			if reasoning := messageReasoning(m); reasoning != "" {
				out = append(out, &components.ChatItem{Kind: components.ItemThinking, Reasoning: reasoning, Expanded: false})
			}
			for _, tc := range m.ToolCalls {
				it := &components.ChatItem{
					Kind:     components.ItemTool,
					ID:       tc.ID,
					ToolName: tc.Function.Name,
					ToolArgs: tc.Function.Arguments,
				}
				enrichToolItem(it)
				toolByID[tc.ID] = it
				out = append(out, it)
			}
			if text := messageText(m); text != "" {
				out = append(out, &components.ChatItem{Kind: components.ItemAssistant, Text: text})
			}
		case llm.RoleTool:
			text := messageText(m)
			if text == "" {
				text = fmt.Sprint(m.Content)
			}
			if it, ok := toolByID[m.ToolCallID]; ok {
				it.ToolOutput = text
			} else if len(out) > 0 && out[len(out)-1].Kind == components.ItemTool {
				out[len(out)-1].ToolOutput = text
			}
		}
	}
	return out
}

func messageReasoning(m session.Message) string {
	switch c := m.Reasoning.(type) {
	case string:
		return c
	}
	return ""
}

func enrichToolItem(it *components.ChatItem) {
	name := it.ToolName
	args := it.ToolArgs
	switch name {
	case "edit":
		var p struct {
			FilePath  string `json:"filePath"`
			OldString string `json:"oldString"`
			NewString string `json:"newString"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.OldString != "" {
			it.Diff = components.ComputeDiff(p.OldString, p.NewString)
			if p.FilePath != "" {
				it.ToolName = "edit:" + p.FilePath
			}
		}
	case "read":
		var p struct {
			FilePath string `json:"filePath"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.FilePath != "" {
			it.ToolName = "read:" + p.FilePath
		}
	case "write":
		var p struct {
			FilePath string `json:"filePath"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.FilePath != "" {
			it.ToolName = "write:" + p.FilePath
		}
	case "bash":
		var p struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Command != "" {
			cmd := p.Command
			if len(cmd) > 60 {
				cmd = cmd[:60] + "…"
			}
			it.ToolName = "bash:" + cmd
		}
	case "glob":
		var p struct {
			Pattern string `json:"pattern"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Pattern != "" {
			it.ToolName = "glob:" + p.Pattern
		}
	case "grep":
		var p struct {
			Pattern string `json:"pattern"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Pattern != "" {
			it.ToolName = "grep:" + p.Pattern
		}
	case "webfetch":
		var p struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.URL != "" {
			it.ToolName = "webfetch:" + p.URL
		}
	case "list":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(args), &p); err == nil && p.Path != "" {
			it.ToolName = "list:" + p.Path
		}
	}
}

func messageContent(m llm.Message) string {
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
