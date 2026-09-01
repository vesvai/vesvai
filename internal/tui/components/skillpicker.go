package components

import (
	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/layout"
)

type SkillPicker struct {
	p *Picker
}

func NewSkillPicker() *SkillPicker {
	return &SkillPicker{p: NewPicker("Skills — type to filter")}
}

func (sp *SkillPicker) SetAll(items []ListItem) { sp.p.SetAll(items) }

func (sp *SkillPicker) All() []ListItem { return sp.p.All() }

func (sp *SkillPicker) Count() int { return sp.p.Count() }

func (sp *SkillPicker) Update(query string) { sp.p.Update(query) }

func (sp *SkillPicker) Selected() (ListItem, bool) { return sp.p.Selected() }

func (sp *SkillPicker) MoveUp() { sp.p.MoveUp() }

func (sp *SkillPicker) MoveDown() { sp.p.MoveDown() }

func (sp *SkillPicker) HandleKey(ev *tcell.EventKey) bool { return sp.p.HandleKey(ev) }

func (sp *SkillPicker) Draw(s tcell.Screen, bounds layout.Region) { sp.p.Draw(s, bounds) }
