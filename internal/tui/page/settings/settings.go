package settings

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/mcp"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

type tabKind int

const (
	tabGeneral tabKind = iota
	tabSession
	tabMCP
	tabSkills
)

var tabNames = []string{"General", "Session", "MCP", "Skills"}

type selectedModel struct {
	provider string
	model    llm.Model
}

type SessionInfo struct {
	ID              string
	Title           string
	Provider        string
	Model           string
	ReasoningEffort string
	Messages        []session.Message
}

type Settings struct {
	deps Deps

	tab     tabKind
	general *generalTab
	mcp     *mcpTab
	skills  *skillsTab
	session *sessionTab

	active *SessionInfo

	sub    components.Component
	model  selectedModel
	errMsg string

	reasoningEffort string

	onClose                 func()
	onModelChange           func(provider string, model llm.Model)
	onReasoningEffortChange func(effort string)
	onSessionChange         func(info SessionInfo)
	onSessionClear          func()
}

func New(deps Deps) *Settings {
	s := &Settings{deps: deps}
	s.general = newGeneral(s)
	s.mcp = newMCP(s)
	s.skills = newSkills(s)
	s.session = newSessionTab(s)
	return s
}

func (s *Settings) SetOnClose(fn func()) { s.onClose = fn }

func (s *Settings) SetSelectedModel(provider string, model llm.Model) {
	s.model = selectedModel{provider: provider, model: model}
}

func (s *Settings) SetOnModelChange(fn func(provider string, model llm.Model)) {
	s.onModelChange = fn
}

func (s *Settings) SetReasoningEffort(effort string) {
	s.reasoningEffort = effort
}

func (s *Settings) SetOnReasoningEffortChange(fn func(effort string)) {
	s.onReasoningEffortChange = fn
}

func (s *Settings) ModelDisplay() string { return s.modelDisplay() }

func (s *Settings) modelSupportsReasoning() bool {
	if s.model.model.Config == nil {
		return false
	}
	return len(s.model.model.Config.ReasoningOptions) > 0
}

func (s *Settings) modelConfigSummary() string {
	if s.model.model.Config == nil {
		return "Config=nil"
	}
	return fmt.Sprintf("Config.ReasoningOptions=%d", len(s.model.model.Config.ReasoningOptions))
}

func (s *Settings) openReasoning() {
	l := components.NewList("Reasoning effort")
	var items []components.ListItem
	items = append(items, components.ListItem{Label: "default", Detail: "use provider default", Marked: s.reasoningEffort == ""})
	if s.model.model.Config != nil {
		for _, opt := range s.model.model.Config.ReasoningOptions {
			if opt.Type == "effort" {
				for _, v := range opt.Values {
					items = append(items, components.ListItem{Label: v, Marked: s.reasoningEffort == v})
				}
			}
		}
	}
	l.SetItems(items)
	l.SetOnSelect(func(_ int, item components.ListItem) {
		if item.Label == "default" {
			s.reasoningEffort = ""
		} else {
			s.reasoningEffort = item.Label
		}
		if s.onReasoningEffortChange != nil {
			s.onReasoningEffortChange(s.reasoningEffort)
		}
		s.back()
	})
	s.openSub(&listModal{title: "Reasoning Effort", list: l, onBack: s.back})
}

func (s *Settings) HasSub() bool { return s.sub != nil }

func (s *Settings) SetActiveSession(info *SessionInfo) { s.active = info }

func (s *Settings) SetOnSessionChange(fn func(info SessionInfo)) {
	s.onSessionChange = fn
}

func (s *Settings) SetOnSessionClear(fn func()) { s.onSessionClear = fn }

func (s *Settings) CloseRequested() bool { return false }

func (s *Settings) nextTab() {
	s.tab = tabKind((int(s.tab) + 1) % len(tabNames))
}

func (s *Settings) prevTab() {
	s.tab = tabKind((int(s.tab) - 1 + len(tabNames)) % len(tabNames))
}

func (s *Settings) back() { s.sub = nil }

func (s *Settings) openSub(c components.Component) { s.sub = c }

func (s *Settings) openTools(server string) {
	l := components.NewList("Tools: " + server)
	var items []components.ListItem
	for _, tool := range mcp.ToolsForServer(server) {
		items = append(items, components.ListItem{Label: tool.Name, Detail: tool.Description})
	}
	if len(items) == 0 {
		items = append(items, components.ListItem{Label: "(no tools registered)"})
	}
	l.SetItems(items)
	s.openSub(&listModal{title: "MCP tools", list: l, onBack: s.back})
}

func (s *Settings) HandleKey(ev *tcell.EventKey) bool {
	if s.sub != nil {
		if s.sub.HandleKey(ev) {
			return true
		}
		return true
	}
	switch ev.Key() {
	case tcell.KeyEsc:
		if s.onClose != nil {
			s.onClose()
		}
		return true
	case tcell.KeyTab, tcell.KeyRight:
		s.nextTab()
		return true
	case tcell.KeyLeft:
		s.prevTab()
		return true
	}
	switch s.tab {
	case tabGeneral:
		return s.general.HandleKey(ev)
	case tabSession:
		return s.session.HandleKey(ev)
	case tabMCP:
		return s.mcp.HandleKey(ev)
	case tabSkills:
		return s.skills.HandleKey(ev)
	}
	return false
}

func (s *Settings) Draw(screen tcell.Screen, bounds layout.Region, focused bool) {
	th := styles.Current()
	components.DrawModalBackdrop(screen, bounds)

	w, h := bounds.Width-6, bounds.Height-6
	if w > 78 {
		w = 78
	}
	if h > 30 {
		h = 30
	}
	inner := components.DrawCenteredBox(screen, bounds, w, h, "Settings")

	x := inner.Left + 1
	for i, name := range tabNames {
		style := th.Base().Foreground(th.Hint).Background(th.InputBg)
		if tabKind(i) == s.tab {
			style = th.Base().Foreground(th.InputBg).Background(th.Accent)
		}
		components.DrawText(screen, x, inner.Top, " "+name+" ", style)
		x += len(name) + 3
	}

	if s.sub != nil {
		s.sub.Draw(screen, inner, true)
		components.DrawFooter(screen, inner, "↑/↓ navigate  Enter select  Esc back")
		return
	}

	content := layout.Region{Left: inner.Left, Top: inner.Top + 2, Width: inner.Width, Height: inner.Height - 3}
	switch s.tab {
	case tabGeneral:
		s.general.Draw(screen, content, focused)
	case tabSession:
		s.session.Draw(screen, content, focused)
	case tabMCP:
		s.mcp.Draw(screen, content, focused)
	case tabSkills:
		s.skills.Draw(screen, content, focused)
	}

	if s.errMsg != "" {
		components.DrawText(screen, inner.Left+1, content.Bottom(), components.TruncateTo(s.errMsg, inner.Width-2),
			th.Base().Foreground(tcell.ColorRed).Background(th.InputBg))
	}
	components.DrawFooter(screen, inner, "Tab / ← → switch tab  Esc close")
}
