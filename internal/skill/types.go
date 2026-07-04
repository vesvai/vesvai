package skill

import (
	"os"
	"path/filepath"
)

type SkillLocation int

const (
	LocationGlobal SkillLocation = iota
	LocationProject
)

func (l SkillLocation) String() string {
	switch l {
	case LocationGlobal:
		return "global"
	case LocationProject:
		return "project"
	default:
		return "unknown"
	}
}

type Skill struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Location SkillLocation `json:"location"`

	Path string `json:"path"`

	Content string `json:"content,omitempty"`
}

type SkillConfig struct {
	Name string `json:"name,omitempty"`

	Description string `json:"description"`

	Triggers []string `json:"triggers,omitempty"`

	Depends []string `json:"depends,omitempty"`
}

func GlobalSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "vesvai", "skills"), nil
}

func ProjectSkillsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".vesvai", "skills")
}
