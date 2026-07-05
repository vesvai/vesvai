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

// FactType represents the category of a knowledge fact
type FactType string

const (
	FactTypeArchitecture  FactType = "architecture"
	FactTypeConvention    FactType = "convention"
	FactTypeDecision      FactType = "decision"
	FactTypeWarning       FactType = "warning"
	FactTypeModule        FactType = "module"
	FactTypeRelationship  FactType = "relationship"
	FactTypeDependency    FactType = "dependency"
	FactTypePattern       FactType = "pattern"
)

// Fact represents a single piece of knowledge about the codebase
type Fact struct {
	ID         string    `json:"id"`
	Type       FactType  `json:"type"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Confidence float64   `json:"confidence"` // 0.0 to 1.0
	Tags       []string  `json:"tags,omitempty"`
	Source     string    `json:"source,omitempty"` // File path or description of where this fact was learned
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Note represents additional context or observations
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkspaceMemory is the complete memory state for a workspace
type WorkspaceMemory struct {
	Facts       []Fact    `json:"facts"`
	Notes       []Note    `json:"notes"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
}

// MemoryStore defines the interface for persisting workspace memory
type MemoryStore interface {
	// Save persists the complete workspace memory
	Save(memory *WorkspaceMemory) error

	// Load retrieves the workspace memory from storage
	Load() (*WorkspaceMemory, error)

	// Exists checks if memory storage exists
	Exists() bool

	// Close releases any resources held by the store
	Close() error
}

// SearchOptions defines options for searching facts
type SearchOptions struct {
	Type       FactType  // Filter by fact type
	MinConfidence float64 // Minimum confidence threshold
	Tags       []string  // Filter by tags (AND logic)
	Limit      int       // Maximum number of results
}

// ListOptions defines options for listing facts
type ListOptions struct {
	Page     int
	PageSize int
	Type     FactType // Optional filter by type
}

// ListResult contains paginated list results
type ListResult struct {
	Facts      []Fact `json:"facts"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}
