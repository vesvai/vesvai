package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "VESVAI_PLUGIN",
	MagicCookieValue: "vesvai",
}

var PluginMap = map[string]plugin.Plugin{
	"vesvai": &VesvaiPluginRPC{},
}

type VesvaiPluginRPC struct {
	Impl Plugin
}

func (p *VesvaiPluginRPC) Server(*plugin.MuxBroker) (interface{}, error) {
	return &VesvaiPluginRPCServer{impl: p.Impl}, nil
}

func (p *VesvaiPluginRPC) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &VesvaiPluginRPCClient{client: c}, nil
}

type VesvaiPluginRPCServer struct {
	impl Plugin
}

func (s *VesvaiPluginRPCServer) Name(args interface{}, resp *string) error {
	*resp = s.impl.Name()
	return nil
}

func (s *VesvaiPluginRPCServer) Version(args interface{}, resp *string) error {
	*resp = s.impl.Version()
	return nil
}

func (s *VesvaiPluginRPCServer) Description(args interface{}, resp *string) error {
	*resp = s.impl.Description()
	return nil
}

func (s *VesvaiPluginRPCServer) Boot(args *Deps, resp *interface{}) error {
	return s.impl.Boot(*args)
}

type VesvaiPluginRPCClient struct {
	client *rpc.Client
}

func (c *VesvaiPluginRPCClient) Name() string {
	var resp string
	err := c.client.Call("Plugin.Name", new(interface{}), &resp)
	if err != nil {
		return ""
	}
	return resp
}

func (c *VesvaiPluginRPCClient) Version() string {
	var resp string
	err := c.client.Call("Plugin.Version", new(interface{}), &resp)
	if err != nil {
		return ""
	}
	return resp
}

func (c *VesvaiPluginRPCClient) Description() string {
	var resp string
	err := c.client.Call("Plugin.Description", new(interface{}), &resp)
	if err != nil {
		return ""
	}
	return resp
}

func (c *VesvaiPluginRPCClient) Boot(deps Deps) error {
	var resp interface{}
	return c.client.Call("Plugin.Boot", &deps, &resp)
}

type PluginError struct {
	Message string
}

func (e *PluginError) Error() string {
	return e.Message
}
