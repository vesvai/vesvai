package session

import (
	"fmt"
	"os"
	"sync"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/core/logger"
	"github.com/vesvai/vesvai/internal/llm"
)

const chunksPerCommit = 10

type Recorder struct {
	mgr      *Manager
	log      *logger.Logger
	mu       sync.Mutex
	sessions map[string]string
	resumed  map[string]string
	pending  map[string]*pendingMessage
}

type pendingMessage struct {
	content   string
	reasoning string
	chunks    int
}

func NewRecorder(mgr *Manager, log *logger.Logger) *Recorder {
	return &Recorder{
		mgr:      mgr,
		log:      log,
		sessions: make(map[string]string),
		resumed:  make(map[string]string),
		pending:  make(map[string]*pendingMessage),
	}
}

func (r *Recorder) Start(bus event.Bus) error {
	subs := []struct {
		topic string
		fn    any
	}{
		{agent.TopicAgentStarted, r.handleStarted},
		{agent.TopicAgentInput, r.handleInput},
		{agent.TopicAgentToken, r.handleToken},
		{agent.TopicAgentMessage, r.handleMessage},
		{agent.TopicAgentToolResult, r.handleToolResult},
		{agent.TopicAgentFinished, r.handleFinished},
		{agent.TopicAgentError, r.handleError},
		{TopicSessionResume, r.handleResume},
	}
	for _, s := range subs {
		if err := bus.Subscribe(s.topic, s.fn); err != nil {
			return fmt.Errorf("session recorder: subscribe %s: %w", s.topic, err)
		}
	}
	return nil
}

func (r *Recorder) Stop(bus event.Bus) error {
	subs := []struct {
		topic string
		fn    any
	}{
		{agent.TopicAgentStarted, r.handleStarted},
		{agent.TopicAgentInput, r.handleInput},
		{agent.TopicAgentToken, r.handleToken},
		{agent.TopicAgentMessage, r.handleMessage},
		{agent.TopicAgentToolResult, r.handleToolResult},
		{agent.TopicAgentFinished, r.handleFinished},
		{agent.TopicAgentError, r.handleError},
		{TopicSessionResume, r.handleResume},
	}
	var firstErr error
	for _, s := range subs {
		if err := bus.Unsubscribe(s.topic, s.fn); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("session recorder: unsubscribe %s: %w", s.topic, err)
		}
	}
	return firstErr
}

func (r *Recorder) sessionID(agentID string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.sessions[agentID]
	return id, ok
}

func (r *Recorder) handleResume(e SessionResume) {
	if e.AgentID == "" || e.SessionID == "" {
		return
	}
	r.mu.Lock()
	r.resumed[e.AgentID] = e.SessionID
	r.mu.Unlock()
}

func (r *Recorder) handleStarted(e agent.AgentStarted) {
	r.mu.Lock()
	if _, ok := r.sessions[e.AgentID]; ok {
		r.mu.Unlock()
		return
	}
	sessID := r.resumed[e.AgentID]
	r.mu.Unlock()

	if sessID == "" {
		opts := CreateOptions{Title: e.AgentName}
		if e.Provider != nil {
			opts.Provider = e.Provider.Name()
		}
		opts.Model = e.Model.ID
		if dir, err := os.Getwd(); err == nil {
			opts.ProjectDir = dir
		}

		s, err := r.mgr.Create(opts)
		if err != nil {
			r.log.Fdebug("session recorder: create session for agent %q: %v", e.AgentName, err)
			return
		}
		sessID = s.ID
	}

	r.mu.Lock()
	r.sessions[e.AgentID] = sessID
	r.mu.Unlock()

	r.mgr.publish(TopicSessionAttached, SessionAttached{
		AgentID:   e.AgentID,
		SessionID: sessID,
	})
}

func (r *Recorder) handleInput(e agent.AgentInput) {
	if id, ok := r.sessionID(e.AgentID); ok {
		if _, err := r.mgr.AppendMessage(id, llm.UserMessage(e.Input)); err != nil {
			r.log.Fdebug("session recorder: append input: %v", err)
		}
	}
}

func (r *Recorder) handleToken(e agent.AgentToken) {
	r.mu.Lock()
	sessID, ok := r.sessions[e.AgentID]
	p := r.pending[e.AgentID]
	if p == nil {
		p = &pendingMessage{}
		r.pending[e.AgentID] = p
	}
	p.content += e.Content
	p.reasoning += e.Reasoning
	p.chunks++
	commit := p.chunks >= chunksPerCommit
	r.mu.Unlock()

	if !ok {
		return
	}
	if commit {
		r.commitPending(e.AgentID, sessID)
	}
}

func (r *Recorder) handleMessage(e agent.AgentMessage) {
	id, ok := r.sessionID(e.AgentID)
	if !ok {
		return
	}
	r.mu.Lock()
	streaming := r.pending[e.AgentID] != nil && r.pending[e.AgentID].chunks > 0
	r.mu.Unlock()
	if streaming {
		r.commitPending(e.AgentID, id)
		return
	}
	if _, err := r.mgr.AppendMessage(id, e.Message); err != nil {
		r.log.Fdebug("session recorder: append message: %v", err)
	}
}

func (r *Recorder) handleError(e agent.AgentError) {
	if id, ok := r.sessionID(e.AgentID); ok {
		r.commitPending(e.AgentID, id)
	}
}

func (r *Recorder) commitPending(agentID, sessID string) {
	r.mu.Lock()
	p, ok := r.pending[agentID]
	if !ok || p.chunks == 0 {
		r.mu.Unlock()
		return
	}
	msg := llm.AssistantMessage(p.content)
	if p.reasoning != "" {
		msg.Reasoning = p.reasoning
	}
	p.content = ""
	p.reasoning = ""
	p.chunks = 0
	r.mu.Unlock()

	if msg.Content == "" && msg.Reasoning == "" {
		return
	}
	if _, err := r.mgr.AppendMessage(sessID, msg); err != nil {
		r.log.Fdebug("session recorder: append tokens: %v", err)
	}
}

func (r *Recorder) handleToolResult(e agent.AgentToolResult) {
	if id, ok := r.sessionID(e.AgentID); ok {
		if _, err := r.mgr.AppendMessage(id, llm.ToolMessage(e.Output, e.CallID)); err != nil {
			r.log.Fdebug("session recorder: append tool result: %v", err)
		}
	}
}

func (r *Recorder) handleFinished(e agent.AgentFinished) {
	if id, ok := r.sessionID(e.AgentID); ok {
		if err := r.mgr.AccumulateUsage(id, e.Usage); err != nil {
			r.log.Fdebug("session recorder: accumulate usage: %v", err)
		}
	}
}
