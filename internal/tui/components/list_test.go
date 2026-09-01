package components

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func items(n int) []ListItem {
	out := make([]ListItem, n)
	for i := range out {
		out[i] = ListItem{Label: "item" + itoa(i), Detail: "detail" + itoa(i)}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [10]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestListNavigation(t *testing.T) {
	l := NewList("test")
	l.SetItems(items(5))
	if got := l.Count(); got != 5 {
		t.Fatalf("Count = %d, want 5", got)
	}
	it, _ := l.Selected()
	if it.Label != "item0" {
		t.Fatalf("first selected = %q", it.Label)
	}
	l.MoveDown()
	it, _ = l.Selected()
	if it.Label != "item1" {
		t.Fatalf("after down selected = %q", it.Label)
	}
	l.MoveUp()
	it, _ = l.Selected()
	if it.Label != "item0" {
		t.Fatalf("after up selected = %q", it.Label)
	}
	l.End()
	it, _ = l.Selected()
	if it.Label != "item4" {
		t.Fatalf("after end selected = %q", it.Label)
	}
	l.Home()
	it, _ = l.Selected()
	if it.Label != "item0" {
		t.Fatalf("after home selected = %q", it.Label)
	}
}

func TestListHandleKeyNavigation(t *testing.T) {
	l := NewList("test")
	l.SetItems(items(3))
	l.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	it, _ := l.Selected()
	if it.Label != "item1" {
		t.Errorf("key down selected = %q, want item1", it.Label)
	}
	l.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'j', 0))
	it, _ = l.Selected()
	if it.Label != "item1" {
		t.Errorf("j should not navigate, selected = %q", it.Label)
	}
}

func TestListFilter(t *testing.T) {
	l := NewList("test")
	l.SetSearchable(true)
	l.SetItems([]ListItem{
		{Label: "opencode-zen", Detail: "opencode-zen"},
		{Label: "openai", Detail: "openai"},
		{Label: "groq", Detail: "groq"},
	})
	for _, r := range []rune("open") {
		l.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, 0))
	}
	if got := l.Count(); got != 2 {
		t.Fatalf("filtered count = %d, want 2", got)
	}
	l.HandleKey(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if got := l.Count(); got != 3 {
		t.Fatalf("after esc count = %d, want 3", got)
	}
	if l.FilterText() != "" {
		t.Errorf("filter should be cleared, got %q", l.FilterText())
	}
}

func TestListFilterSelectAndEnter(t *testing.T) {
	l := NewList("test")
	l.SetSearchable(true)
	l.SetItems([]ListItem{
		{Label: "alpha", Detail: "provider-a"},
		{Label: "beta", Detail: "provider-b"},
	})
	for _, r := range []rune("beta") {
		l.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, 0))
	}
	var got ListItem
	l.SetOnSelect(func(_ int, item ListItem) { got = item })
	l.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if got.Label != "beta" {
		t.Errorf("selected = %q, want beta", got.Label)
	}
}

func TestListEnterNoItems(t *testing.T) {
	l := NewList("test")
	l.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
}

func TestListHomeEndEmpty(t *testing.T) {
	l := NewList("test")
	l.Home()
	l.End()
	l.MoveUp()
	l.MoveDown()
}
