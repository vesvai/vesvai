package components

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestPickerFilter(t *testing.T) {
	p := NewPicker("test")
	p.SetAll([]ListItem{
		{Label: "go-development", Detail: "write Go code"},
		{Label: "websearch", Detail: "search the web"},
		{Label: "testing", Detail: "run tests"},
	})
	p.Update("go")
	if p.Count() != 1 {
		t.Fatalf("Count = %d, want 1", p.Count())
	}
	it, ok := p.Selected()
	if !ok || it.Label != "go-development" {
		t.Errorf("selected = %+v, want go-development", it)
	}
	p.Update("")
	if p.Count() != 3 {
		t.Errorf("Count after empty query = %d, want 3", p.Count())
	}
}

func TestPickerNavigation(t *testing.T) {
	p := NewPicker("test")
	p.SetAll([]ListItem{
		{Label: "a"},
		{Label: "b"},
		{Label: "c"},
	})
	p.Update("")
	p.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if it, _ := p.Selected(); it.Label != "b" {
		t.Errorf("after down selected = %q, want b", it.Label)
	}
	p.HandleKey(tcell.NewEventKey(tcell.KeyUp, 0, 0))
	if it, _ := p.Selected(); it.Label != "a" {
		t.Errorf("after up selected = %q, want a", it.Label)
	}
}

func TestPickerEmptyResult(t *testing.T) {
	p := NewPicker("test")
	p.SetAll([]ListItem{{Label: "go-development"}})
	p.Update("zzz")
	if p.Count() != 0 {
		t.Errorf("Count = %d, want 0", p.Count())
	}
	if _, ok := p.Selected(); ok {
		t.Error("no selection when empty")
	}
}
