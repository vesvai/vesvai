package lsp

import "errors"

var (
	ErrNotRunning  = errors.New("lsp: server not running")
	ErrTimeout     = errors.New("lsp: request timed out")
	ErrClosed      = errors.New("lsp: transport closed")
	ErrProtocol    = errors.New("lsp: protocol error")
	ErrNotFound    = errors.New("lsp: binary not found")
	ErrUnsupported = errors.New("lsp: unsupported configuration")
)
