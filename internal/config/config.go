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

func GetMemoryDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, MemoryDirName), nil
}

func GetTodosDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, TodosDirName), nil
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

func AddProvider(provider LLMConfig) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	cfg.Providers = append(cfg.Providers, provider)

	return Save(cfg)
}

type ModelsCache struct {
	Providers map[string][]string `json:"providers"`
}

func GetModelsCachePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ModelsCacheFileName), nil
}

func LoadModelsCache() (*ModelsCache, error) {
	cache := &ModelsCache{
		Providers: make(map[string][]string),
	}

	cachePath, err := GetModelsCachePath()
	if err != nil {
		return cache, nil
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return nil, fmt.Errorf("failed to read models cache: %w", err)
	}

	if err := json.Unmarshal(data, cache); err != nil {
		return nil, fmt.Errorf("failed to parse models cache: %w", err)
	}

	return cache, nil
}

func SaveModelsCache(cache *ModelsCache) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	cachePath := filepath.Join(configDir, ModelsCacheFileName)

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal models cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write models cache: %w", err)
	}

	return nil
}

func SaveProviderModels(providerName string, modelIDs []string) error {
	cache, err := LoadModelsCache()
	if err != nil {
		return err
	}

	cache.Providers[providerName] = modelIDs

	return SaveModelsCache(cache)
}
