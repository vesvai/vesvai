package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetProjectConfigDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return filepath.Join(cwd, ProjectConfigDirName), nil
}

func EnsureProjectConfigDir() error {
	dir, err := GetProjectConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

func GetProjectConfigPath(paths ...string) (string, error) {
	configDir, err := GetProjectConfigDir()
	if err != nil {
		return "", err
	}
	allPaths := append([]string{configDir}, paths...)
	return filepath.Join(allPaths...), nil
}
