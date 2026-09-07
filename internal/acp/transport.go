package acp

import (
	"context"

	json "github.com/goccy/go-json"
)

type Message struct {
	SessionId string
	Body      json.RawMessage
}

type OutboundFunc func(sessionID string, body json.RawMessage) error

type Transport interface {
	Start(ctx context.Context) (<-chan Message, error)
	Send(msg Message) error
	Close() error
}
