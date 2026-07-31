package memory

import (
	"errors"
	"time"
)

var (
	ErrFactNotFound = errors.New("fact not found")
	ErrNoteNotFound = errors.New("note not found")
	ErrStoreClosed  = errors.New("store is closed")
	ErrInvalidFact  = errors.New("invalid fact: title and content are required")
	ErrInvalidNote  = errors.New("invalid note: title and content are required")
)

type FactType string

const (
	FactTypeArchitecture FactType = "architecture"
	FactTypeConvention   FactType = "convention"
	FactTypeDecision     FactType = "decision"
	FactTypeWarning      FactType = "warning"
	FactTypeModule       FactType = "module"
	FactTypeRelationship FactType = "relationship"
	FactTypeDependency   FactType = "dependency"
	FactTypePattern      FactType = "pattern"
)

type Fact struct {
	ID         string    `json:"id"`
	Type       FactType  `json:"type"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Confidence float64   `json:"confidence"`
	Tags       []string  `json:"tags,omitempty"`
	Source     string    `json:"source,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkspaceMemory struct {
	Facts     []Fact    `json:"facts"`
	Notes     []Note    `json:"notes"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

type MemoryStore interface {
	Save(memory *WorkspaceMemory) error

	Load() (*WorkspaceMemory, error)

	Exists() bool

	Close() error
}

type SearchOptions struct {
	Type          FactType
	MinConfidence float64
	Tags          []string
	Limit         int
}

type ListOptions struct {
	Page     int
	PageSize int
	Type     FactType
}

type ListResult struct {
	Facts      []Fact `json:"facts"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}
