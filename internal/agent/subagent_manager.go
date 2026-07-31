package agent

import (
	"fmt"
	"sync"
	"time"
)

type SubagentStatus string

const (
	SubagentStatusRunning   SubagentStatus = "running"
	SubagentStatusCompleted SubagentStatus = "completed"
	SubagentStatusFailed    SubagentStatus = "failed"
	SubagentStatusCancelled SubagentStatus = "cancelled"
)

type SubagentRun struct {
	ID        string
	Name      string
	Prompt    string
	ParentID  string
	Status    SubagentStatus
	StartedAt time.Time
	EndedAt   time.Time

	Content   string
	Reasoning string

	ToolCalls []SubagentToolEvent

	Result string
	Error  error

	mu sync.RWMutex
}

type SubagentToolEvent struct {
	ToolName string
	Args     map[string]any
	Result   string
	Error    error
	Duration int64
}

func (r *SubagentRun) AppendContent(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Content += s
}

func (r *SubagentRun) AppendReasoning(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Reasoning += s
}

func (r *SubagentRun) AppendToolCall(ev SubagentToolEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ToolCalls = append(r.ToolCalls, ev)
}

func (r *SubagentRun) SetStatus(s SubagentStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Status = s
	if s != SubagentStatusRunning {
		r.EndedAt = time.Now()
	}
}

func (r *SubagentRun) SetResult(result string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Result = result
	r.Error = err
	if err != nil {
		r.Status = SubagentStatusFailed
	} else {
		r.Status = SubagentStatusCompleted
	}
	r.EndedAt = time.Now()
}

func (r *SubagentRun) Snapshot() SubagentRunSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]SubagentToolEvent, len(r.ToolCalls))
	copy(tools, r.ToolCalls)
	return SubagentRunSnapshot{
		ID:        r.ID,
		Name:      r.Name,
		Prompt:    r.Prompt,
		ParentID:  r.ParentID,
		Status:    r.Status,
		StartedAt: r.StartedAt,
		EndedAt:   r.EndedAt,
		Content:   r.Content,
		Reasoning: r.Reasoning,
		ToolCalls: tools,
		Result:    r.Result,
	}
}

type SubagentRunSnapshot struct {
	ID        string
	Name      string
	Prompt    string
	ParentID  string
	Status    SubagentStatus
	StartedAt time.Time
	EndedAt   time.Time
	Content   string
	Reasoning string
	ToolCalls []SubagentToolEvent
	Result    string
}

type SubagentManager struct {
	mu   sync.RWMutex
	runs map[string]*SubagentRun
}

func NewSubagentManager() *SubagentManager {
	return &SubagentManager{
		runs: make(map[string]*SubagentRun),
	}
}

func (m *SubagentManager) Start(id, name, prompt, parentID string) *SubagentRun {
	run := &SubagentRun{
		ID:        id,
		Name:      name,
		Prompt:    prompt,
		ParentID:  parentID,
		Status:    SubagentStatusRunning,
		StartedAt: time.Now(),
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[id] = run
	return run
}

func (m *SubagentManager) Get(id string) (*SubagentRun, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runs[id]
	return r, ok
}

func (m *SubagentManager) Active() []*SubagentRun {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var active []*SubagentRun
	for _, r := range m.runs {
		if r.Status == SubagentStatusRunning {
			active = append(active, r)
		}
	}
	return active
}

func (m *SubagentManager) All() []SubagentRunSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snapshots := make([]SubagentRunSnapshot, 0, len(m.runs))
	for _, r := range m.runs {
		snapshots = append(snapshots, r.Snapshot())
	}
	return snapshots
}

func (m *SubagentManager) Cancel(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok || r.Status != SubagentStatusRunning {
		return false
	}
	r.Status = SubagentStatusCancelled
	r.EndedAt = time.Now()
	return true
}

func (m *SubagentManager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.runs, id)
}

type Mailbox struct {
	mu       sync.Mutex
	messages map[string][]MailMessage
}

type MailMessage struct {
	From    string
	To      string
	Content string
	Sent    time.Time
}

func NewMailbox() *Mailbox {
	return &Mailbox{
		messages: make(map[string][]MailMessage),
	}
}

func (mb *Mailbox) Post(from, to, content string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.messages[to] = append(mb.messages[to], MailMessage{
		From:    from,
		To:      to,
		Content: content,
		Sent:    time.Now(),
	})
}

func (mb *Mailbox) Collect(name string) []MailMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	msgs := mb.messages[name]
	delete(mb.messages, name)
	return msgs
}

func (mb *Mailbox) HasMessages(name string) bool {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	return len(mb.messages[name]) > 0
}

func generateRunID(name string) string {
	return fmt.Sprintf("subagent-%s-%d", name, time.Now().UnixNano())
}
