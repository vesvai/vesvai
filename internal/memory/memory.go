package memory

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vesvai/vesvai/internal/filesystem"
)

type Memory struct {
	store  MemoryStore
	memory *WorkspaceMemory
	fs     *filesystem.FileSystem
}

func New(store MemoryStore, fs *filesystem.FileSystem) *Memory {
	return &Memory{
		store: store,
		fs:    fs,
	}
}

func (m *Memory) Load() error {
	memory, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load memory: %w", err)
	}
	m.memory = memory
	return nil
}

func (m *Memory) Save() error {
	if m.memory == nil {
		return fmt.Errorf("memory not loaded")
	}
	return m.store.Save(m.memory)
}

func (m *Memory) Initialize(workspacePath string) error {
	if m.store.Exists() {
		return m.Load()
	}

	analyzer := NewAnalyzer(workspacePath, m.fs)
	memory, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("failed to analyze workspace: %w", err)
	}

	m.memory = memory
	return m.store.Save(m.memory)
}

func (m *Memory) AddFact(factType FactType, title, content string, confidence float64, tags []string, source string) (*Fact, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	if title == "" || content == "" {
		return nil, ErrInvalidFact
	}

	fact := Fact{
		ID:         uuid.New().String(),
		Type:       factType,
		Title:      title,
		Content:    content,
		Confidence: confidence,
		Tags:       tags,
		Source:     source,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	m.memory.Facts = append(m.memory.Facts, fact)
	return &fact, nil
}

func (m *Memory) GetFact(id string) (*Fact, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	for i := range m.memory.Facts {
		if m.memory.Facts[i].ID == id {
			return &m.memory.Facts[i], nil
		}
	}

	return nil, ErrFactNotFound
}

func (m *Memory) UpdateFact(id string, updates map[string]interface{}) (*Fact, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	for i := range m.memory.Facts {
		if m.memory.Facts[i].ID == id {
			fact := &m.memory.Facts[i]

			if title, ok := updates["title"].(string); ok {
				fact.Title = title
			}
			if content, ok := updates["content"].(string); ok {
				fact.Content = content
			}
			if confidence, ok := updates["confidence"].(float64); ok {
				fact.Confidence = confidence
			}
			if tags, ok := updates["tags"].([]string); ok {
				fact.Tags = tags
			}
			if source, ok := updates["source"].(string); ok {
				fact.Source = source
			}

			fact.UpdatedAt = time.Now()
			return fact, nil
		}
	}

	return nil, ErrFactNotFound
}

func (m *Memory) DeleteFact(id string) error {
	if m.memory == nil {
		return fmt.Errorf("memory not loaded")
	}

	for i := range m.memory.Facts {
		if m.memory.Facts[i].ID == id {
			m.memory.Facts = append(m.memory.Facts[:i], m.memory.Facts[i+1:]...)
			return nil
		}
	}

	return ErrFactNotFound
}

func (m *Memory) SearchFacts(query string, opts *SearchOptions) ([]Fact, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	results := make([]Fact, 0)

	for _, fact := range m.memory.Facts {
		if query != "" {
			queryLower := strings.ToLower(query)
			titleMatch := strings.Contains(strings.ToLower(fact.Title), queryLower)
			contentMatch := strings.Contains(strings.ToLower(fact.Content), queryLower)
			if !titleMatch && !contentMatch {
				continue
			}
		}

		if opts != nil {
			if opts.Type != "" && fact.Type != opts.Type {
				continue
			}
			if opts.MinConfidence > 0 && fact.Confidence < opts.MinConfidence {
				continue
			}
			if len(opts.Tags) > 0 {
				if !hasAllTags(fact.Tags, opts.Tags) {
					continue
				}
			}
		}

		results = append(results, fact)
	}

	if opts != nil && opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

func (m *Memory) ListFacts(opts *ListOptions) (*ListResult, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	filtered := m.memory.Facts
	if opts != nil && opts.Type != "" {
		filtered = make([]Fact, 0)
		for _, fact := range m.memory.Facts {
			if fact.Type == opts.Type {
				filtered = append(filtered, fact)
			}
		}
	}

	total := len(filtered)

	page := 1
	pageSize := 50
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PageSize > 0 {
			pageSize = opts.PageSize
		}
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return &ListResult{
		Facts:      filtered[start:end],
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}

func (m *Memory) AddNote(title, content string, tags []string) (*Note, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	if title == "" || content == "" {
		return nil, ErrInvalidNote
	}

	note := Note{
		ID:        uuid.New().String(),
		Title:     title,
		Content:   content,
		Tags:      tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.memory.Notes = append(m.memory.Notes, note)
	return &note, nil
}

func (m *Memory) GetNote(id string) (*Note, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	for i := range m.memory.Notes {
		if m.memory.Notes[i].ID == id {
			return &m.memory.Notes[i], nil
		}
	}

	return nil, ErrNoteNotFound
}

func (m *Memory) UpdateNote(id string, updates map[string]interface{}) (*Note, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	for i := range m.memory.Notes {
		if m.memory.Notes[i].ID == id {
			note := &m.memory.Notes[i]

			if title, ok := updates["title"].(string); ok {
				note.Title = title
			}
			if content, ok := updates["content"].(string); ok {
				note.Content = content
			}
			if tags, ok := updates["tags"].([]string); ok {
				note.Tags = tags
			}

			note.UpdatedAt = time.Now()
			return note, nil
		}
	}

	return nil, ErrNoteNotFound
}

func (m *Memory) DeleteNote(id string) error {
	if m.memory == nil {
		return fmt.Errorf("memory not loaded")
	}

	for i := range m.memory.Notes {
		if m.memory.Notes[i].ID == id {
			m.memory.Notes = append(m.memory.Notes[:i], m.memory.Notes[i+1:]...)
			return nil
		}
	}

	return ErrNoteNotFound
}

func (m *Memory) MergeFacts(newFact Fact) (*Fact, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	for i := range m.memory.Facts {
		existing := &m.memory.Facts[i]
		if strings.EqualFold(existing.Title, newFact.Title) && existing.Type == newFact.Type {
			existing.Content = newFact.Content
			existing.Confidence = maxFloat64(existing.Confidence, newFact.Confidence)
			existing.UpdatedAt = time.Now()

			existing.Tags = mergeTags(existing.Tags, newFact.Tags)

			return existing, nil
		}
	}

	newFact.ID = uuid.New().String()
	newFact.CreatedAt = time.Now()
	newFact.UpdatedAt = time.Now()
	m.memory.Facts = append(m.memory.Facts, newFact)
	return &newFact, nil
}

func (m *Memory) GetFactsByType(factType FactType) ([]Fact, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	results := make([]Fact, 0)
	for _, fact := range m.memory.Facts {
		if fact.Type == factType {
			results = append(results, fact)
		}
	}

	return results, nil
}

func (m *Memory) GetHighConfidenceFacts(threshold float64) ([]Fact, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("memory not loaded")
	}

	results := make([]Fact, 0)
	for _, fact := range m.memory.Facts {
		if fact.Confidence >= threshold {
			results = append(results, fact)
		}
	}

	return results, nil
}

func (m *Memory) GetStats() map[string]interface{} {
	if m.memory == nil {
		return nil
	}

	stats := map[string]interface{}{
		"total_facts": len(m.memory.Facts),
		"total_notes": len(m.memory.Notes),
		"version":     m.memory.Version,
		"updated_at":  m.memory.UpdatedAt,
	}

	typeCounts := make(map[FactType]int)
	for _, fact := range m.memory.Facts {
		typeCounts[fact.Type]++
	}
	stats["facts_by_type"] = typeCounts

	if len(m.memory.Facts) > 0 {
		totalConfidence := 0.0
		for _, fact := range m.memory.Facts {
			totalConfidence += fact.Confidence
		}
		stats["average_confidence"] = totalConfidence / float64(len(m.memory.Facts))
	}

	return stats
}

func hasAllTags(factTags, requiredTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range factTags {
		tagSet[strings.ToLower(tag)] = true
	}

	for _, required := range requiredTags {
		if !tagSet[strings.ToLower(required)] {
			return false
		}
	}

	return true
}

func mergeTags(existing, new []string) []string {
	tagSet := make(map[string]bool)
	for _, tag := range existing {
		tagSet[strings.ToLower(tag)] = true
	}

	for _, tag := range new {
		tagSet[strings.ToLower(tag)] = true
	}

	result := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		result = append(result, tag)
	}

	return result
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
