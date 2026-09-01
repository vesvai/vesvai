package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/llm"
)

func (c *CLI) newLoginCommand() *cobra.Command {
	var provider string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Add or update a provider with an API key",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return c.runLogin(provider, apiKey, c.selectProvider, c.promptAPIKey)
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "provider name to log in to")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "provider API key")

	return cmd
}

func (c *CLI) runLogin(provider, apiKey string, selectProvider func() (string, error), promptAPIKey func() (string, error)) error {
	if provider == "" {
		p, err := selectProvider()
		if err != nil {
			return err
		}
		provider = p
	} else if !llm.HasProvider(provider) {
		return fmt.Errorf("cli: unknown provider %q (available: %v)", provider, llm.ListProviders())
	}

	if apiKey == "" {
		key, err := promptAPIKey()
		if err != nil {
			return err
		}
		apiKey = key
	}

	cfg := config.LLMConfig{
		Provider: provider,
		APIKey:   apiKey,
	}

	if err := c.syncProvider(cfg); err != nil {
		return err
	}

	if err := config.UpsertProvider(cfg); err != nil {
		return fmt.Errorf("cli: save provider %q: %w", provider, err)
	}

	c.log.Finfo("provider %q saved", provider)

	return nil
}

func (c *CLI) syncProvider(cfg config.LLMConfig) error {
	loaded := make(chan llm.ModelsLoaded, 1)
	handler := func(ml llm.ModelsLoaded) {
		if ml.Provider == cfg.Provider {
			select {
			case loaded <- ml:
			default:
			}
		}
	}
	if err := c.bus.Subscribe(event.TopicModelsLoaded, handler); err != nil {
		return fmt.Errorf("cli: subscribe models loaded: %w", err)
	}
	defer c.bus.Unsubscribe(event.TopicModelsLoaded, handler)

	c.bus.Publish(event.TopicProviderAdded, cfg)

	select {
	case ml := <-loaded:
		if ml.Err != nil {
			return fmt.Errorf("cli: provider %q: %w", cfg.Provider, ml.Err)
		}
		return nil
	case <-time.After(35 * time.Second):
		return fmt.Errorf("cli: timed out syncing provider %q", cfg.Provider)
	}
}

func (c *CLI) selectProvider() (string, error) {
	names := llm.ListProviders()
	if len(names) == 0 {
		return "", errors.New("cli: no providers registered")
	}

	p := promptui.Select{
		Label: "Select provider",
		Items: names,
		Size:  10,
		Searcher: func(input string, index int) bool {
			provider := strings.ToLower(names[index])
			searchValue := strings.ToLower(input)
			return strings.Contains(provider, searchValue)
		},
	}

	_, result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("cli: select provider: %w", err)
	}
	return result, nil
}

func (c *CLI) promptAPIKey() (string, error) {
	p := promptui.Prompt{
		Label: "API key (optional)",
		Mask:  '*',
	}

	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("cli: prompt api key: %w", err)
	}
	return result, nil
}
