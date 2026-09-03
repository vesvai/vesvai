package subagent

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/session"
)

type Status string

const (
	StatusPending     Status = "pending"
	StatusRunning     Status = "running"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
	StatusInterrupted Status = "interrupted"
)

type SubAgent struct {
	Name       string    `json:"name"`
	AgentType  string    `json:"agent"`
	TaskIDs    []string  `json:"task_id,omitempty"`
	Background bool      `json:"background"`
	Status     Status    `json:"status"`
	Output     string    `json:"output,omitempty"`
	Err        string    `json:"error,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`

	agentID  string
	done     chan struct{}
	finished bool
}

type registry struct {
	mu       sync.Mutex
	agents   map[string]*SubAgent
	initDone bool
	file     string
	subBus   event.Bus
}

var store = newRegistry()

func newRegistry() *registry {
	return &registry{agents: make(map[string]*SubAgent)}
}

func (r *registry) init() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.initDone {
		return nil
	}
	if err := config.EnsureProjectConfigDir(); err != nil {
		return fmt.Errorf("subagent: ensure config dir: %w", err)
	}
	file, err := config.GetProjectConfigPath("subagents.json")
	if err != nil {
		return fmt.Errorf("subagent: config path: %w", err)
	}
	r.file = file

	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			r.initDone = true
			return nil
		}
		return fmt.Errorf("subagent: read file: %w", err)
	}
	var list []*SubAgent
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("subagent: parse: %w", err)
	}
	changed := false
	for _, sa := range list {
		if sa.Status == StatusPending || sa.Status == StatusRunning {
			sa.Status = StatusInterrupted
			sa.Err = "interrupted: app restarted"
			sa.UpdatedAt = time.Now()
			changed = true
		}
		sa.done = make(chan struct{})
		if sa.Status == StatusCompleted || sa.Status == StatusFailed || sa.Status == StatusInterrupted {
			sa.finished = true
			close(sa.done)
		}
		r.agents[sa.Name] = sa
	}
	r.initDone = true
	if changed {
		return r.saveLocked()
	}
	return nil
}

func (r *registry) save() error {
	if err := r.init(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveLocked()
}

func (r *registry) saveLocked() error {
	list := make([]*SubAgent, 0, len(r.agents))
	for _, sa := range r.agents {
		list = append(list, sa)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("subagent: marshal: %w", err)
	}
	if err := os.WriteFile(r.file, data, 0o644); err != nil {
		return fmt.Errorf("subagent: write: %w", err)
	}
	return nil
}

func (r *registry) subscribe(bus event.Bus) {
	if bus == nil {
		return
	}
	r.mu.Lock()
	if r.subBus == bus {
		r.mu.Unlock()
		return
	}
	r.subBus = bus
	r.mu.Unlock()

	bus.Subscribe(agent.TopicAgentStarted, func(started agent.AgentStarted) {
		r.onStarted(started.AgentID)
	})
	bus.Subscribe(agent.TopicAgentFinished, func(finished agent.AgentFinished) {
		r.onFinished(finished.AgentID, finished.Output)
	})
	bus.Subscribe(agent.TopicAgentError, func(errEv agent.AgentError) {
		r.onError(errEv.AgentID, errEv.Err)
	})
	bus.Subscribe(session.TopicSessionAttached, func(attached session.SessionAttached) {
		r.onSessionAttached(attached.AgentID, attached.SessionID)
	})
}

func (r *registry) onSessionAttached(agentID, sessionID string) {
	if sessionID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sa := range r.agents {
		if sa.agentID == agentID {
			sa.SessionID = sessionID
			return
		}
	}
}

func (r *registry) byAgentID(agentID string) *SubAgent {
	if agentID == "" {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sa := range r.agents {
		if sa.agentID == agentID {
			return sa
		}
	}
	return nil
}

func IsSubagentSession(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	if err := store.init(); err != nil {
		return false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, sa := range store.agents {
		if sa.SessionID == sessionID {
			return true
		}
	}
	return false
}

func (r *registry) onStarted(agentID string) {
	if sa := r.byAgentID(agentID); sa != nil {
		r.start(sa)
	}
}

func (r *registry) onFinished(agentID, output string) {
	if sa := r.byAgentID(agentID); sa != nil {
		r.finish(sa, output, nil)
	}
}

func (r *registry) onError(agentID string, err error) {
	if sa := r.byAgentID(agentID); sa != nil {
		r.finish(sa, "", err)
	}
}

func (r *registry) spawn(name, agentType string, taskIDs []string, background bool) (*SubAgent, error) {
	if err := r.init(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	if _, ok := r.agents[name]; ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("subagent: duplicate name %q (names must be unique)", name)
	}
	now := time.Now()
	sa := &SubAgent{
		Name:       name,
		AgentType:  agentType,
		TaskIDs:    taskIDs,
		Background: background,
		Status:     StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		done:       make(chan struct{}),
	}
	r.agents[name] = sa
	r.mu.Unlock()
	if err := r.save(); err != nil {
		return nil, err
	}
	return sa, nil
}

func (r *registry) resume(name string) (*SubAgent, error) {
	if err := r.init(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	sa, ok := r.agents[name]
	if !ok {
		return nil, fmt.Errorf("subagent: no subagent named %q", name)
	}
	if !sa.finished {
		return nil, fmt.Errorf("subagent: %q is still running", name)
	}
	sa.finished = false
	sa.Status = StatusPending
	sa.done = make(chan struct{})
	sa.UpdatedAt = time.Now()
	if err := r.saveLocked(); err != nil {
		return nil, err
	}
	return sa, nil
}

func (r *registry) setAgentID(name, agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sa, ok := r.agents[name]; ok {
		sa.agentID = agentID
	}
}

func (r *registry) start(sa *SubAgent) {
	r.mu.Lock()
	sa.Status = StatusRunning
	sa.UpdatedAt = time.Now()
	r.mu.Unlock()
	_ = r.save()
}

func (r *registry) finish(sa *SubAgent, output string, err error) {
	r.mu.Lock()
	if sa.finished {
		if output != "" {
			sa.Output = output
		}
		if err != nil && sa.Err == "" {
			sa.Err = err.Error()
		}
		r.mu.Unlock()
		_ = r.save()
		return
	}
	sa.finished = true
	sa.Output = output
	sa.UpdatedAt = time.Now()
	if err != nil {
		sa.Status = StatusFailed
		sa.Err = err.Error()
	} else {
		sa.Status = StatusCompleted
	}
	close(sa.done)
	r.mu.Unlock()
	_ = r.save()
}

func (r *registry) get(name string) (SubAgent, bool) {
	if err := r.init(); err != nil {
		return SubAgent{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	sa, ok := r.agents[name]
	if !ok {
		return SubAgent{}, false
	}
	return copyOf(sa), true
}

func (r *registry) all() []SubAgent {
	if err := r.init(); err != nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]SubAgent, 0, len(r.agents))
	for _, sa := range r.agents {
		out = append(out, copyOf(sa))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (r *registry) filter(names []string) ([]SubAgent, error) {
	out := make([]SubAgent, 0, len(names))
	for _, n := range names {
		sa, ok := r.get(n)
		if !ok {
			return nil, fmt.Errorf("subagent: no subagent named %q", n)
		}
		out = append(out, sa)
	}
	return out, nil
}

func (r *registry) waitFor(ctx context.Context, names []string) ([]SubAgent, error) {
	if err := r.init(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	live := make([]*SubAgent, 0, len(names))
	for _, n := range names {
		sa, ok := r.agents[n]
		if !ok {
			r.mu.Unlock()
			return nil, fmt.Errorf("subagent: no subagent named %q", n)
		}
		live = append(live, sa)
	}
	r.mu.Unlock()

	for _, sa := range live {
		select {
		case <-sa.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	out := make([]SubAgent, 0, len(live))
	for _, sa := range live {
		out = append(out, copyOf(sa))
	}
	return out, nil
}

func copyOf(sa *SubAgent) SubAgent {
	return SubAgent{
		Name:       sa.Name,
		AgentType:  sa.AgentType,
		TaskIDs:    append([]string(nil), sa.TaskIDs...),
		Background: sa.Background,
		Status:     sa.Status,
		Output:     sa.Output,
		Err:        sa.Err,
		SessionID:  sa.SessionID,
		CreatedAt:  sa.CreatedAt,
		UpdatedAt:  sa.UpdatedAt,
	}
}

func formatStatuses(sas []SubAgent) string {
	if len(sas) == 0 {
		return "No subagents.\n"
	}
	var b strings.Builder
	for _, sa := range sas {
		fmt.Fprintf(&b, "%s %s (agent: %s, background: %v)\n", sa.Status, sa.Name, sa.AgentType, sa.Background)
		if len(sa.TaskIDs) > 0 {
			fmt.Fprintf(&b, "   task_id: %s\n", strings.Join(sa.TaskIDs, ", "))
		}
		if sa.Status == StatusCompleted {
			fmt.Fprintf(&b, "   response:\n%s\n", sa.Output)
		}
		if (sa.Status == StatusFailed || sa.Status == StatusInterrupted) && sa.Err != "" {
			fmt.Fprintf(&b, "   error: %s\n", sa.Err)
		}
	}
	return b.String()
}
