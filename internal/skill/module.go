package skill

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/core/config"
)

//go:embed skills
var embedded embed.FS

const (
	builtinDir      = "builtin"
	agentsSkillsDir = ".agents/skills"
)

func SkillModule() error {
	vesvaiSkills, err := config.GetConfigPath("skills")
	if err != nil {
		return err
	}
	if err := MaterializeTo(filepath.Join(vesvaiSkills, builtinDir)); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("skill: home dir: %w", err)
	}
	if err := LoadDirs(
		filepath.Join(home, agentsSkillsDir),
		vesvaiSkills,
	); err != nil {
		return err
	}
	agent.OnMessageInput(ExpandMessage)
	return nil
}

func MaterializeTo(root string) error {
	entries, err := fs.ReadDir(embedded, "skills")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("skill: read embedded: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := copyDir(filepath.Join("skills", e.Name()), filepath.Join(root, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(from, to string) error {
	return fs.WalkDir(embedded, from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := fs.ReadFile(embedded, path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
}
