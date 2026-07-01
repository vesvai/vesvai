package session

import (
	"time"

	"github.com/vesvai/vesvai/internal/llm"
)

type Session struct {
	ID        string          `json:"id"`
	Title     string          `json:"title,omitempty"`
	Messages  []llm.Message   `json:"messages"`
	Metadata  SessionMetadata `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type SessionMetadata struct {
	Model         string `json:"model,omitempty"`
	MessageCount  int    `json:"message_count"`
	TotalTokens   int    `json:"total_tokens,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
}

type SessionMetadataIndex struct {
	ID           string    `json:"id"`
	Title        string    `json:"title,omitempty"`
	MessageCount int       `json:"message_count"`
	Model        string    `json:"model,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ListOptions struct {
	Page     int
	PageSize int
	SortBy   string
	Reverse  bool
}

type ListResult struct {
	Sessions   []SessionMetadataIndex `json:"sessions"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

type Store interface {
	Save(session *Session) error
	Load(id string) (*Session, error)
	Delete(id string) error
	List(opts ListOptions) (*ListResult, error)
	Exists(id string) bool
	Close() error
}
