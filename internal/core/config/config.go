package config

import (
	"fmt"
	json "github.com/goccy/go-json"
	"os"
	"path/filepath"
)

type LLMConfig struct {
	Provider   string            `json:"provider"`
	Driver     string            `json:"driver"`
	APIKey     string            `json:"api_key,omitempty"`
	BaseURL    string            `json:"base_url,omitempty"`
	Timeout    int               `json:"timeout,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type LoggerConfig struct {
	Driver      string `json:"driver"`
	MaxLogCount int    `json:"max_log_count"`
}

type CacheConfig struct {
	Driver string `json:"driver"`
}

type SessionConfig struct {
	Driver string `json:"driver"`
}

type MCPServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type LanguageServerConfig struct {
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	FileTypes   []string          `json:"filetypes,omitempty"`
	RootMarkers []string          `json:"rootMarkers,omitempty"`
	Download    string            `json:"download,omitempty"`
	Install     string            `json:"install,omitempty"`
	Version     string            `json:"version,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
}

type Config struct {
	Providers       []LLMConfig                     `json:"providers"`
	Logger          LoggerConfig                    `json:"logger"`
	Cache           CacheConfig                     `json:"cache"`
	Session         SessionConfig                   `json:"session"`
	MCPServers      map[string]MCPServerConfig      `json:"mcp_servers,omitempty"`
	LanguageServers map[string]LanguageServerConfig `json:"language_servers,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		Logger: LoggerConfig{
			Driver:      "sqlite",
			MaxLogCount: 1000,
		},
		Cache: CacheConfig{
			Driver: "sqlite",
		},
		Session: SessionConfig{
			Driver: "sqlite",
		},
		MCPServers:      make(map[string]MCPServerConfig),
		LanguageServers: make(map[string]LanguageServerConfig),
	}
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, GlobalConfigDirName), nil
}

func GetConfigPath(paths ...string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	allPaths := append([]string{configDir}, paths...)
	return filepath.Join(allPaths...), nil
}

func Load() (*Config, error) {
	config := DefaultConfig()

	configPath, err := GetConfigPath(GlobalConfigFileName)
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
	configPath, err := GetConfigPath(GlobalConfigFileName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

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
	configPath, err := GetConfigPath(GlobalConfigFileName)
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	return Save(config)
}

func UpsertProvider(provider LLMConfig) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	for i := range cfg.Providers {
		if cfg.Providers[i].Provider == provider.Provider {
			cfg.Providers[i] = provider
			return Save(cfg)
		}
	}

	cfg.Providers = append(cfg.Providers, provider)
	return Save(cfg)
}

func RemoveProvider(name string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	found := false
	providers := cfg.Providers[:0]
	for _, p := range cfg.Providers {
		if p.Provider == name {
			found = true
			continue
		}
		providers = append(providers, p)
	}
	if !found {
		return fmt.Errorf("provider %q not found", name)
	}

	cfg.Providers = providers
	return Save(cfg)
}

func UpsertMCPServer(name string, server MCPServerConfig) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]MCPServerConfig)
	}
	cfg.MCPServers[name] = server
	return Save(cfg)
}

func RemoveMCPServer(name string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.MCPServers[name]; !ok {
		return fmt.Errorf("mcp server %q not found", name)
	}

	delete(cfg.MCPServers, name)
	return Save(cfg)
}

func UpsertLanguageServer(name string, server LanguageServerConfig) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	if cfg.LanguageServers == nil {
		cfg.LanguageServers = make(map[string]LanguageServerConfig)
	}
	cfg.LanguageServers[name] = server
	return Save(cfg)
}

func RemoveLanguageServer(name string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.LanguageServers[name]; !ok {
		return fmt.Errorf("language server %q not found", name)
	}

	delete(cfg.LanguageServers, name)
	return Save(cfg)
}
