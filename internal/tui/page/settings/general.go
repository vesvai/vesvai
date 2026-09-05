package settings

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

type generalTab struct {
	settings *Settings
	index    int
}

func newGeneral(s *Settings) *generalTab { return &generalTab{settings: s} }

const generalRowCount = 4

func (g *generalTab) rowEnabled(i int) bool {
	if i == 3 {
		return g.settings.modelSupportsReasoning()
	}
	return true
}

func (g *generalTab) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		if g.index > 0 {
			g.index--
		}
		return true
	case tcell.KeyDown:
		if g.index < generalRowCount-1 {
			g.index++
		}
		return true
	case tcell.KeyEnter:
		if !g.rowEnabled(g.index) {
			return true
		}
		g.rows()[g.index].action()
		return true
	}
	return false
}

type genRow struct {
	label  string
	value  string
	action func()
}

func (g *generalTab) rows() []genRow {
	s := g.settings
	effort := s.reasoningEffort
	if effort == "" {
		effort = "default"
	}
	return []genRow{
		{label: "Provider", value: s.providerValue(), action: s.openProviders},
		{label: "Model", value: s.modelDisplay(), action: s.openModels},
		{label: "Theme", value: styles.Name(), action: s.openThemes},
		{label: "Reasoning", value: effort, action: s.openReasoning},
	}
}

func (g *generalTab) Draw(screen tcell.Screen, bounds layout.Region, _ bool) {
	th := styles.Current()
	rows := g.rows()
	for i, r := range rows {
		y := bounds.Top + i
		style := th.Base().Background(th.InputBg)
		marker := "  "
		enabled := g.rowEnabled(i)
		if i == g.index {
			style = th.Base().Foreground(th.InputText).Background(th.Selection)
			marker = "> "
		} else if !enabled {
			style = th.Base().Foreground(th.Placeholder).Background(th.InputBg)
		}
		label := r.label
		if !enabled {
			label += " (not supported)"
		}
		components.DrawText(screen, bounds.Left+1, y, marker+label, style)
		if enabled || i == g.index {
			components.DrawText(screen, bounds.Right()-len(r.value)-3, y, r.value, style)
		}
	}
}

func (s *Settings) configured(name string) (config.LLMConfig, bool) {
	if s.deps.Config == nil {
		return config.LLMConfig{}, false
	}
	for _, p := range s.deps.Config.Providers {
		if p.Provider == name {
			return p, true
		}
	}
	return config.LLMConfig{}, false
}

func (s *Settings) providerValue() string {
	if s.deps.Config == nil || len(s.deps.Config.Providers) == 0 {
		return "none configured"
	}
	first := s.deps.Config.Providers[0].Provider
	if len(s.deps.Config.Providers) > 1 {
		return fmt.Sprintf("%s +%d", first, len(s.deps.Config.Providers)-1)
	}
	return first
}

func (s *Settings) openProviders() {
	l := components.NewList("Select provider")
	var items []components.ListItem
	for _, name := range llm.ListProviders() {
		detail, marked := "not configured", false
		if _, ok := s.configured(name); ok {
			detail, marked = "configured", true
		}
		items = append(items, components.ListItem{Label: name, Detail: detail, Marked: marked, Data: name})
	}
	l.SetItems(items)
	l.SetOnSelect(func(_ int, item components.ListItem) {
		name, _ := item.Data.(string)
		if _, ok := s.configured(name); ok {
			s.openProviderChoice(name)
		} else {
			s.openAPIKey(name)
		}
	})
	s.openSub(&listModal{title: "Providers", list: l, onBack: s.back})
}

func (s *Settings) openProviderChoice(name string) {
	l := components.NewList("Provider: " + name)
	l.SetItems([]components.ListItem{
		{Label: "Use existing configuration", Detail: "keep the current API key"},
		{Label: "Configure again", Detail: "re-enter the API key"},
	})
	l.SetOnSelect(func(i int, _ components.ListItem) {
		if i == 1 {
			s.openAPIKey(name)
		} else {
			s.back()
		}
	})
	s.openSub(&listModal{title: "Already configured", list: l, onBack: s.back})
}

func (s *Settings) openAPIKey(name string) {
	f := components.NewField("API key for " + name)
	f.SetMask('*')
	f.SetPlaceholder("paste your API key here")
	f.Focus()
	f.SetOnEnter(func(key string) { s.saveProvider(name, strings.TrimSpace(key)) })
	f.SetOnCancel(s.back)
	s.openSub(&fieldModal{title: "Configure provider", field: f})
}

func (s *Settings) saveProvider(name, apiKey string) {
	cfg, ok := s.configured(name)
	if !ok {
		cfg = config.LLMConfig{Provider: name}
	}
	cfg.APIKey = apiKey

	if err := config.UpsertProvider(cfg); err != nil {
		s.errMsg = "failed to save provider: " + err.Error()
		s.back()
		return
	}
	s.errMsg = ""

	if fresh, err := config.Load(); err == nil {
		s.deps.Config = fresh
	}
	if s.deps.Bus != nil {
		s.deps.Bus.Publish(event.TopicProviderAdded, cfg)
	}
	s.back()
}

func (s *Settings) modelDisplay() string {
	if s.model.provider == "" {
		return "—"
	}
	name := s.model.model.ID
	if s.model.model.Name != "" {
		name = s.model.model.Name
	}
	return s.model.provider + "/" + name
}

type modelData struct {
	provider string
	model    llm.Model
}

func (s *Settings) openModels() {
	l := components.NewList("Select model")
	l.SetSearchable(true)

	var items []components.ListItem
	if s.deps.Config != nil && s.deps.LLM != nil {
		for _, p := range s.deps.Config.Providers {
			models, err := s.deps.LLM.Models(p.Provider)
			if err != nil {
				continue
			}
			for _, m := range models {
				label := m.ID
				if m.Name != "" {
					label = m.Name
				}
				marked := s.model.provider == p.Provider && s.model.model.ID == m.ID
				items = append(items, components.ListItem{
					Label:  label,
					Detail: p.Provider,
					Marked: marked,
					Data:   modelData{provider: p.Provider, model: m},
				})
			}
		}
	}
	l.SetItems(items)
	l.SetOnSelect(func(_ int, item components.ListItem) {
		if md, ok := item.Data.(modelData); ok {
			s.model = selectedModel{provider: md.provider, model: md.model}
			if s.onModelChange != nil {
				s.onModelChange(md.provider, md.model)
			}
		}
		s.back()
	})
	s.openSub(&listModal{title: "Models (type to search)", list: l, onBack: s.back})
}

func (s *Settings) openThemes() {
	l := components.NewList("Select theme")
	var items []components.ListItem
	for _, name := range styles.Names() {
		items = append(items, components.ListItem{Label: name, Marked: name == styles.Name()})
	}
	l.SetItems(items)
	l.SetOnSelect(func(_ int, item components.ListItem) {
		styles.Set(item.Label)
		if s.onThemeChange != nil {
			s.onThemeChange()
		}
		s.back()
	})
	s.openSub(&listModal{title: "Theme", list: l, onBack: s.back})
}
