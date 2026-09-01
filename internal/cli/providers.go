package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vesvai/vesvai/internal/core/cache"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/llm"
)

func maskAPIKey(key string) string {
	if key == "" {
		return "(none)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func (c *CLI) runProviders(out io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cli: load config: %w", err)
	}
	if len(cfg.Providers) == 0 {
		fmt.Fprintln(out, "no providers configured")
		return nil
	}

	store, err := cache.CacheModule(cfg.Cache)
	if err != nil {
		return fmt.Errorf("cli: open cache: %w", err)
	}
	defer store.Close()

	fmt.Fprintf(out, "%-12s %-24s %s\n", "PROVIDER", "API KEY", "MODELS")
	for _, p := range cfg.Providers {
		count := 0
		if raw, err := store.Get(p.Provider); err == nil {
			var models []llm.Model
			if json.Unmarshal(raw, &models) == nil {
				count = len(models)
			}
		}
		fmt.Fprintf(out, "%-12s %-24s %d\n", p.Provider, maskAPIKey(p.APIKey), count)
	}
	return nil
}

func (c *CLI) newProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Manage providers",
	}
	cmd.AddCommand(c.newProviderListCommand(), c.newProviderRemoveCommand(), c.newProviderRefreshCommand())
	return cmd
}

func (c *CLI) newProviderListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured providers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.runProviders(cmd.OutOrStdout())
		},
	}
}

func (c *CLI) newProviderRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a configured provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return c.runProviderRemove(args[0])
		},
	}
}

func (c *CLI) runProviderRemove(name string) error {
	if err := config.RemoveProvider(name); err != nil {
		return fmt.Errorf("cli: remove provider: %w", err)
	}

	cfg, err := config.Load()
	if err == nil {
		if store, cerr := cache.CacheModule(cfg.Cache); cerr == nil {
			_ = store.Delete(name)
			_ = store.Close()
		}
	}

	c.bus.Publish(event.TopicProviderRemoved, name)
	c.log.Finfo("provider %q removed", name)
	return nil
}

func (c *CLI) newProviderRefreshCommand() *cobra.Command {
	var provider string
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Re-fetch models for providers (bypasses cache)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.runProviderRefresh(cmd.OutOrStdout(), provider)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "only refresh this provider")
	return cmd
}

func (c *CLI) runProviderRefresh(out io.Writer, provider string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cli: load config: %w", err)
	}

	targets := configuredProviders(cfg, provider)
	if len(targets) == 0 {
		return fmt.Errorf("cli: provider %q not configured", provider)
	}

	store, err := cache.CacheModule(cfg.Cache)
	if err != nil {
		return fmt.Errorf("cli: open cache: %w", err)
	}
	defer store.Close()

	for _, p := range targets {
		if err := c.forceSyncProvider(store, p); err != nil {
			fmt.Fprintf(out, "%-12s FAIL  %v\n", p.Provider, err)
		} else {
			fmt.Fprintf(out, "%-12s OK\n", p.Provider)
		}
	}
	return nil
}

func (c *CLI) forceSyncProvider(store cache.Cache, cfg config.LLMConfig) error {
	_ = store.Delete(cfg.Provider)
	return c.syncProvider(cfg)
}

func configuredProviders(cfg *config.Config, provider string) []config.LLMConfig {
	var targets []config.LLMConfig
	for _, p := range cfg.Providers {
		if provider == "" || p.Provider == provider {
			targets = append(targets, p)
		}
	}
	return targets
}

func decodeModels(raw []byte) ([]llm.Model, bool) {
	var models []llm.Model
	if err := json.Unmarshal(raw, &models); err != nil {
		return nil, false
	}
	return models, true
}
