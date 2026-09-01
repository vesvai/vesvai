package config

import "testing"

func TestUpsertProviderAdds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := UpsertProvider(LLMConfig{Provider: "groq", APIKey: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertProvider(LLMConfig{Provider: "openai", APIKey: "b"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Providers) != 2 {
		t.Fatalf("providers = %+v, want 2", cfg.Providers)
	}
}

func TestUpsertProviderUpdates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := UpsertProvider(LLMConfig{Provider: "groq", APIKey: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertProvider(LLMConfig{Provider: "groq", APIKey: "b"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("providers = %+v, want 1", cfg.Providers)
	}
	if cfg.Providers[0].Provider != "groq" || cfg.Providers[0].APIKey != "b" {
		t.Fatalf("provider = %+v, want groq/b", cfg.Providers[0])
	}
}

func TestRemoveProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := UpsertProvider(LLMConfig{Provider: "groq", APIKey: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertProvider(LLMConfig{Provider: "openai", APIKey: "b"}); err != nil {
		t.Fatal(err)
	}

	if err := RemoveProvider("groq"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].Provider != "openai" {
		t.Fatalf("providers = %+v, want only openai", cfg.Providers)
	}

	if err := RemoveProvider("nope"); err == nil {
		t.Fatal("expected error for missing provider")
	}
}

func TestUpsertMCPServerAdds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := UpsertMCPServer("db", MCPServerConfig{Command: "npx", Args: []string{"-y"}}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertMCPServer("remote", MCPServerConfig{URL: "https://example.com/sse"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MCPServers) != 2 {
		t.Fatalf("mcp servers = %+v, want 2", cfg.MCPServers)
	}
	if cfg.MCPServers["db"].Command != "npx" {
		t.Fatalf("db = %+v", cfg.MCPServers["db"])
	}
	if cfg.MCPServers["remote"].URL != "https://example.com/sse" {
		t.Fatalf("remote = %+v", cfg.MCPServers["remote"])
	}
}

func TestUpsertMCPServerUpdates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := UpsertMCPServer("db", MCPServerConfig{Command: "npx"}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertMCPServer("db", MCPServerConfig{Command: "go", Args: []string{"run", "server"}}); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MCPServers) != 1 {
		t.Fatalf("mcp servers = %+v, want 1", cfg.MCPServers)
	}
	if cfg.MCPServers["db"].Command != "go" || len(cfg.MCPServers["db"].Args) != 2 {
		t.Fatalf("db = %+v", cfg.MCPServers["db"])
	}
}

func TestRemoveMCPServer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := UpsertMCPServer("db", MCPServerConfig{Command: "npx"}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertMCPServer("remote", MCPServerConfig{URL: "https://example.com/sse"}); err != nil {
		t.Fatal(err)
	}

	if err := RemoveMCPServer("db"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MCPServers) != 1 || cfg.MCPServers["remote"].URL == "" {
		t.Fatalf("mcp servers = %+v, want only remote", cfg.MCPServers)
	}

	if err := RemoveMCPServer("nope"); err == nil {
		t.Fatal("expected error for missing server")
	}
}
