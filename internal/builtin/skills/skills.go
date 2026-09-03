package skills

import (
	"github.com/vesvai/vesvai/internal/agent/prompt"
)

func All() map[string]*prompt.Prompt {
	return map[string]*prompt.Prompt{
		"batch":  BatchSkill(),
		"review": ReviewSkill(),
		"init":   InitSkill(),
	}
}
