package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/vesvai/vesvai/internal/event"
)

type Client struct {
	transport  Transport
	serverName string

	mu           sync.RWMutex
	initialized  bool
	serverInfo   Implementation
	capabilities ServerCapabilities
	instructions string
	eventBus     event.EventBus
}

func NewClient(transport Transport, opts ...ClientOption) *Client {
	c := &Client{
		transport: transport,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type ClientOption func(*Client)

func WithEventBus(bus event.EventBus) ClientOption {
	return func(c *Client) {
		c.eventBus = bus
	}
}

func WithServerName(name string) ClientOption {
	return func(c *Client) {
		c.serverName = name
	}
}

func (c *Client) Connect(ctx context.Context) error {
	if err := c.transport.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	return c.initialize(ctx)
}

func (c *Client) initialize(ctx context.Context) error {
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{
				ListChanged: true,
			},
		},
		ClientInfo: Implementation{
			Name:    ClientName,
			Version: ClientVersion,
		},
	}

	resp, err := c.transport.SendRequest(ctx, MethodInitialize, params)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	var result InitializeResult
	if err := decodeResult(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to decode initialize result: %w", err)
	}

	if err := c.transport.SendNotification(ctx, MethodInitialized, nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.mu.Lock()
	c.initialized = true
	c.serverInfo = result.ServerInfo
	c.capabilities = result.Capabilities
	c.instructions = result.Instructions
	c.mu.Unlock()

	if c.eventBus != nil {
		c.eventBus.Publish(ctx, NewMCPEvent(EventMCPConnect, ConnectEventData{
			ServerName:    c.serverInfo.Name,
			ServerVersion: c.serverInfo.Version,
			Instructions:  result.Instructions,
		}))
	}

	return nil
}

func (c *Client) ServerInfo() Implementation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

func (c *Client) Capabilities() ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

func (c *Client) Instructions() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.instructions
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.transport.SendRequest(ctx, MethodToolsList, nil)
	if err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	var result ListToolsResult
	if err := decodeResult(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode tools/list result: %w", err)
	}

	if c.eventBus != nil {
		c.eventBus.Publish(ctx, NewMCPEvent(EventMCPToolsList, ToolsListEventData{
			ServerName: c.serverInfo.Name,
			ToolCount:  len(result.Tools),
		}))
	}

	return result.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*CallToolResult, error) {
	params := CallToolParams{
		Name:      name,
		Arguments: args,
	}

	if c.eventBus != nil {
		c.eventBus.Publish(ctx, NewMCPEvent(EventMCPToolCall, ToolCallEventData{
			ServerName: c.serverInfo.Name,
			ToolName:   name,
			Arguments:  args,
		}))
	}

	resp, err := c.transport.SendRequest(ctx, MethodToolsCall, params)
	if err != nil {
		if c.eventBus != nil {
			c.eventBus.Publish(ctx, NewMCPEvent(EventMCPToolResult, ToolResultEventData{
				ServerName: c.serverInfo.Name,
				ToolName:   name,
				IsError:    true,
			}))
		}
		return nil, fmt.Errorf("tools/call failed: %w", err)
	}

	if resp.Error != nil {
		if c.eventBus != nil {
			c.eventBus.Publish(ctx, NewMCPEvent(EventMCPToolResult, ToolResultEventData{
				ServerName: c.serverInfo.Name,
				ToolName:   name,
				IsError:    true,
			}))
		}
		return nil, fmt.Errorf("tools/call error: %s", resp.Error.Message)
	}

	var result CallToolResult
	if err := decodeResult(resp.Result, &result); err != nil {
		if c.eventBus != nil {
			c.eventBus.Publish(ctx, NewMCPEvent(EventMCPToolResult, ToolResultEventData{
				ServerName: c.serverInfo.Name,
				ToolName:   name,
				IsError:    true,
			}))
		}
		return nil, fmt.Errorf("failed to decode tools/call result: %w", err)
	}

	if result.IsError {
		if c.eventBus != nil {
			c.eventBus.Publish(ctx, NewMCPEvent(EventMCPToolResult, ToolResultEventData{
				ServerName: c.serverInfo.Name,
				ToolName:   name,
				IsError:    true,
			}))
		}
		var texts []string
		for _, c := range result.Content {
			if c.Type == "text" && c.Text != "" {
				texts = append(texts, c.Text)
			}
		}
		if len(texts) > 0 {
			return nil, fmt.Errorf("tool error: %s", strings.Join(texts, "\n"))
		}
		return nil, fmt.Errorf("tool returned error")
	}

	if c.eventBus != nil {
		c.eventBus.Publish(ctx, NewMCPEvent(EventMCPToolResult, ToolResultEventData{
			ServerName: c.serverInfo.Name,
			ToolName:   name,
			IsError:    false,
		}))
	}

	return &result, nil
}

func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	resp, err := c.transport.SendRequest(ctx, MethodResourcesList, nil)
	if err != nil {
		return nil, fmt.Errorf("resources/list failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("resources/list error: %s", resp.Error.Message)
	}

	var result ListResourcesResult
	if err := decodeResult(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode resources/list result: %w", err)
	}

	if c.eventBus != nil {
		c.eventBus.Publish(ctx, NewMCPEvent(EventMCPResourcesList, ResourcesListEventData{
			ServerName:    c.serverInfo.Name,
			ResourceCount: len(result.Resources),
		}))
	}

	return result.Resources, nil
}

func (c *Client) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	params := ReadResourceParams{
		URI: uri,
	}

	resp, err := c.transport.SendRequest(ctx, MethodResourcesRead, params)
	if err != nil {
		return nil, fmt.Errorf("resources/read failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("resources/read error: %s", resp.Error.Message)
	}

	var result ReadResourceResult
	if err := decodeResult(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode resources/read result: %w", err)
	}

	if c.eventBus != nil {
		c.eventBus.Publish(ctx, NewMCPEvent(EventMCPResourceRead, ResourceReadEventData{
			ServerName: c.serverInfo.Name,
			URI:        uri,
		}))
	}

	return &result, nil
}

func (c *Client) ListPrompts(ctx context.Context) ([]Prompt, error) {
	resp, err := c.transport.SendRequest(ctx, MethodPromptsList, nil)
	if err != nil {
		return nil, fmt.Errorf("prompts/list failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("prompts/list error: %s", resp.Error.Message)
	}

	var result ListPromptsResult
	if err := decodeResult(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode prompts/list result: %w", err)
	}

	if c.eventBus != nil {
		c.eventBus.Publish(ctx, NewMCPEvent(EventMCPPromptsList, PromptsListEventData{
			ServerName:  c.serverInfo.Name,
			PromptCount: len(result.Prompts),
		}))
	}

	return result.Prompts, nil
}

func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]string) (*GetPromptResult, error) {
	params := GetPromptParams{
		Name:      name,
		Arguments: args,
	}

	resp, err := c.transport.SendRequest(ctx, MethodPromptsGet, params)
	if err != nil {
		return nil, fmt.Errorf("prompts/get failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("prompts/get error: %s", resp.Error.Message)
	}

	var result GetPromptResult
	if err := decodeResult(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode prompts/get result: %w", err)
	}

	if c.eventBus != nil {
		c.eventBus.Publish(ctx, NewMCPEvent(EventMCPPromptGet, PromptGetEventData{
			ServerName: c.serverInfo.Name,
			PromptName: name,
		}))
	}

	return &result, nil
}

func (c *Client) SendNotification(ctx context.Context, method string, params any) error {
	return c.transport.SendNotification(ctx, method, params)
}

func (c *Client) SetNotificationHandler(handler func(notification JSONRPCNotification)) {
	c.transport.SetNotificationHandler(handler)
}

func (c *Client) Close() error {
	serverName := c.serverInfo.Name
	if serverName == "" {
		serverName = c.serverName
	}

	err := c.transport.Close()

	if c.eventBus != nil {
		c.eventBus.Publish(context.Background(), NewMCPEvent(EventMCPDisconnect, DisconnectEventData{
			ServerName: serverName,
		}))
	}

	return err
}

func decodeResult(raw any, target any) error {
	if raw == nil {
		return nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return json.Unmarshal(data, target)
}
