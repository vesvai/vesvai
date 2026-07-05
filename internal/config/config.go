package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Providers []LLMConfig `json:"providers"`
	MCPs      []MCPConfig `json:"mcps"`
}

type LLMConfig struct {
	Provider   string            `json:"provider"`
	APIKey     string            `json:"api_key,omitempty"`
	Models     string            `json:"models"`
	BaseURL    string            `json:"base_url,omitempty"`
	Timeout    int               `json:"timeout,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type MCPConfig struct {
	Type string `json:"type"`

	Url     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	Command     []string `json:"command,omitempty"`
	Environment []string `json:"environment,omitempty"`

	Enabled bool `json:"enabled"`
}

func DefaultConfig() *Config {
	return &Config{
		Providers: []LLMConfig{},
	}
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ConfigDirName), nil
}

func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ConfigFileName), nil
}

func GetSessionsDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, SessionsDirName), nil
}

func GetSkillsDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, SkillsDirName), nil
}

func GetPluginsDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, PluginsDirName), nil
}

func Load() (*Config, error) {
	config := DefaultConfig()

	configPath, err := GetConfigPath()
	if err != nil {
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func Save(config *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, ConfigFileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func SaveIfNotExists(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	return Save(config)
}
