package settings

import (
	"os"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
	"github.com/vesvai/vesvai/internal/utils/query"
)

type sessionTab struct {
	settings *Settings
	index    int
}

func newSessionTab(s *Settings) *sessionTab { return &sessionTab{settings: s} }

const sessionRowCount = 4

func (t *sessionTab) rowEnabled(i int) bool {
	if i == 2 || i == 3 {
		return t.settings.active != nil
	}
	return true
}

func (t *sessionTab) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		if t.index > 0 {
			t.index--
		}
		return true
	case tcell.KeyDown:
		if t.index < sessionRowCount-1 {
			t.index++
		}
		return true
	case tcell.KeyEnter:
		if !t.rowEnabled(t.index) {
			return true
		}
		switch t.index {
		case 0:
			t.settings.openSessionList()
		case 1:
			t.settings.newSession()
		case 2:
			t.settings.openDeleteConfirm()
		case 3:
			t.settings.openTitle()
		}
		return true
	}
	return false
}

func (t *sessionTab) Draw(screen tcell.Screen, bounds layout.Region, _ bool) {
	th := styles.Current()
	rows := []struct {
		label, value string
	}{
		{"Load session", t.activeValue()},
		{"New session", ""},
		{"Delete session", ""},
		{"Change title", ""},
	}
	for i, r := range rows {
		style := th.Base().Background(th.InputBg)
		marker := "  "
		enabled := t.rowEnabled(i)
		if i == t.index {
			style = th.Base().Foreground(th.InputText).Background(th.Selection)
			marker = "> "
		} else if !enabled {
			style = th.Base().Foreground(th.Placeholder).Background(th.InputBg)
		}
		label := r.label
		if !enabled {
			label += " (no active session)"
		}
		components.DrawText(screen, bounds.Left+1, bounds.Top+i, marker+label, style)
		if r.value != "" {
			components.DrawText(screen, bounds.Right()-len(r.value)-3, bounds.Top+i, r.value, style)
		}
	}
}

func (t *sessionTab) activeValue() string {
	if t.settings.active == nil {
		return "—"
	}
	return t.settings.active.Title
}

func (s *Settings) hasSessions() bool { return s.deps.Sessions != nil }

func (s *Settings) openSessionList() {
	if !s.hasSessions() {
		s.errMsg = "session store unavailable"
		return
	}
	l := components.NewList("Load session")

	var items []components.ListItem
	sessions, _, err := s.deps.Sessions.List(query.Query{
		Page: query.Page{Number: 1, Size: 200},
		Sort: []query.Sort{{Column: "updated_at", Dir: query.Desc}},
	})
	if err == nil {
		if dir, werr := os.Getwd(); werr == nil {
			var filtered []session.Session
			for _, sess := range sessions {
				if sess.ProjectDir == dir {
					filtered = append(filtered, sess)
				}
			}
			sessions = filtered
		}
		sort.Slice(sessions, func(i, j int) bool { return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt) })
		for _, sess := range sessions {
			items = append(items, components.ListItem{
				Label:  sess.Title,
				Detail: sess.Provider + "/" + sess.Model + " · " + sess.UpdatedAt.Format("2006-01-02 15:04"),
				Data:   sess.ID,
			})
		}
	} else {
		s.errMsg = "failed to list sessions: " + err.Error()
	}
	if len(items) == 0 {
		items = append(items, components.ListItem{Label: "(no sessions in this project)"})
	}
	l.SetItems(items)
	l.SetOnSelect(func(_ int, item components.ListItem) {
		if id, ok := item.Data.(string); ok {
			s.loadSession(id)
		} else {
			s.back()
		}
	})
	s.openSub(&listModal{title: "Sessions", list: l, onBack: s.back})
}

func (s *Settings) loadSession(id string) {
	if !s.hasSessions() {
		s.errMsg = "session store unavailable"
		s.back()
		return
	}
	sess, err := s.deps.Sessions.Get(id)
	if err != nil {
		s.errMsg = "failed to load session: " + err.Error()
		s.back()
		return
	}
	msgs, err := s.deps.Sessions.Messages(id)
	if err != nil {
		s.errMsg = "failed to load messages: " + err.Error()
		s.back()
		return
	}
	info := SessionInfo{
		ID:       sess.ID,
		Title:    sess.Title,
		Provider: sess.Provider,
		Model:    sess.Model,
		Messages: msgs,
	}
	s.active = &info
	s.errMsg = ""
	if s.onSessionChange != nil {
		s.onSessionChange(info)
	}
	s.back()
	s.closeModal()
}

func (s *Settings) newSession() {
	s.active = nil
	s.errMsg = ""
	if s.onSessionClear != nil {
		s.onSessionClear()
	}
	s.back()
	s.closeModal()
}

func (s *Settings) closeModal() {
	if s.onClose != nil {
		s.onClose()
	}
}

func (s *Settings) openDeleteConfirm() {
	if s.active == nil {
		return
	}
	l := components.NewList("Delete session")
	title := s.active.Title
	if title == "" {
		title = "(untitled)"
	}
	l.SetItems([]components.ListItem{
		{Label: "Delete", Detail: "session \"" + title + "\" will be permanently removed"},
		{Label: "Cancel", Detail: "keep the session"},
	})
	l.SetOnSelect(func(i int, _ components.ListItem) {
		if i == 0 {
			s.deleteSession()
		} else {
			s.back()
		}
	})
	s.openSub(&listModal{title: "Confirm deletion", list: l, onBack: s.back})
}

func (s *Settings) deleteSession() {
	if s.active == nil || !s.hasSessions() {
		s.back()
		return
	}
	id := s.active.ID
	s.active = nil
	if err := s.deps.Sessions.Delete(id); err != nil {
		s.errMsg = "failed to delete session: " + err.Error()
	} else {
		s.errMsg = ""
	}
	if s.onSessionClear != nil {
		s.onSessionClear()
	}
	s.back()
}

func (s *Settings) openTitle() {
	if s.active == nil {
		return
	}
	f := components.NewField("Session title")
	f.SetPlaceholder("enter a title")
	f.SetText(s.active.Title)
	f.Focus()
	f.SetOnEnter(func(title string) { s.saveTitle(strings.TrimSpace(title)) })
	f.SetOnCancel(s.back)
	s.openSub(&fieldModal{title: "Change title", field: f})
}

func (s *Settings) saveTitle(title string) {
	if title == "" || s.active == nil || !s.hasSessions() {
		s.back()
		return
	}
	if err := s.deps.Sessions.SetTitle(s.active.ID, title); err != nil {
		s.errMsg = "failed to save title: " + err.Error()
		s.back()
		return
	}
	s.active.Title = title
	s.errMsg = ""
	if s.onSessionChange != nil {
		s.onSessionChange(*s.active)
	}
	s.back()
}
