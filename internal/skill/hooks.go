package skill

import (
	"context"

	"github.com/vesvai/vesvai/internal/hook"
)

func RegisterHooks(hooks *hook.Hooks, mgr *Manager) {
	if hooks == nil || mgr == nil {
		return
	}
	hooks.AddFilter(hook.HookSkillsCollect, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		existing, _ := value.([]Skill)
		skills, err := mgr.List()
		if err != nil {
			return existing
		}
		return append(existing, skills...)
	}, 50)
}
