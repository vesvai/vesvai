package agents

import "errors"

var (
	ErrNilFactory = errors.New("agents: nil factory")
	ErrNilAgent   = errors.New("agents: factory returned nil agent")
	ErrEmptyName  = errors.New("agents: empty name")
	ErrDuplicate  = errors.New("agents: already registered")
	ErrNotFound   = errors.New("agents: not found")
)
