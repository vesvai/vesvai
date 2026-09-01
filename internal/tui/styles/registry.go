package styles

import (
	"sort"
	"sync"

	"github.com/vesvai/vesvai/internal/core/hook"
)

var (
	registryMu  sync.RWMutex
	registry    = map[string]Theme{}
	currentName string
	current     Theme
	onChange    = hook.NewHook[Theme]()
	defaults    sync.Once
)

func Register(name string, t Theme) {
	registryMu.Lock()
	defer registryMu.Unlock()
	t.Name = name
	registry[name] = t
}

func Get(name string) (Theme, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	t, ok := registry[name]
	return t, ok
}

func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func RegisterDefaults() {
	defaults.Do(func() {
		Register("dark", Dark())
		Register("light", Light())
		Register("dracula", Dracula())
	})
}

func Set(name string) bool {
	RegisterDefaults()
	registryMu.Lock()
	t, ok := registry[name]
	if ok {
		currentName = name
		current = onChange.Apply(t)
	}
	registryMu.Unlock()
	return ok
}

func Current() Theme {
	RegisterDefaults()
	registryMu.RLock()
	t := current
	name := currentName
	registryMu.RUnlock()
	if name == "" {
		Set("dark")
		return Current()
	}
	return t
}

func Name() string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if currentName == "" {
		return "dark"
	}
	return currentName
}

func Next() Theme {
	names := Names()
	if len(names) == 0 {
		return Current()
	}
	cur := Name()
	idx := 0
	for i, n := range names {
		if n == cur {
			idx = i
			break
		}
	}
	Set(names[(idx+1)%len(names)])
	return Current()
}

func OnChange(fn func(Theme) Theme) {
	onChange.Add(fn)
}
