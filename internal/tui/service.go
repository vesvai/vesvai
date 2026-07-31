package tui

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agents/orchestrator"
	"github.com/vesvai/vesvai/internal/bootstrap"
	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui/components"
)

type AgentService struct {
	bootstrap    *bootstrap.App
	runner       *agent.Runner
	sessionStore *session.FileStore
	config       *config.Config
	approver     *TUIApprover

	currentSession *session.Session
	currentModel   string
	mu             sync.Mutex

	eventBus   event.EventBus
	tuiAdapter *TUIEventAdapter
	ctx        context.Context
	cancel     context.CancelFunc
	pending    bool
	pendingMu  sync.Mutex
}

func NewAgentService(cfg *config.Config) (*AgentService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	bapp := bootstrap.New(cfg)
	if err := bapp.Init(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("bootstrap init: %w", err)
	}

	provider, err := bapp.CreateProvider()
	if err != nil {
		cancel()
		bapp.Shutdown(ctx)
		return nil, fmt.Errorf("create provider: %w", err)
	}

	approver := NewTUIApprover()
	runner := bapp.CreateRunnerWithApprover(provider, approver)

	store, err := session.NewFileStore(getWorkspacePath())
	if err != nil {
		cancel()
		bapp.Shutdown(ctx)
		return nil, fmt.Errorf("session store: %w", err)
	}

	model := selectInitialModel(cfg)

	svc := &AgentService{
		bootstrap:    bapp,
		runner:       runner,
		sessionStore: store,
		config:       cfg,
		currentModel: model,
		approver:     approver,
		eventBus:     bapp.EventBus,
		ctx:          ctx,
		cancel:       cancel,
	}

	svc.tuiAdapter = NewTUIEventAdapter(bapp.EventBus)

	svc.loadOrCreateSession()

	return svc, nil
}

func (s *AgentService) TUIEventAdapter() *TUIEventAdapter {
	return s.tuiAdapter
}

func (s *AgentService) Runner() *agent.Runner {
	return s.runner
}

func (s *AgentService) ApprovalRequests() <-chan ApprovalRequest {
	return s.approver.Requests()
}

func (s *AgentService) CurrentSession() *session.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentSession
}

func (s *AgentService) Send(text string, atts []components.Attachment, streamCallback func(agent.StreamChunk)) error {
	return s.SendWithContext(s.ctx, text, atts, streamCallback)
}

func (s *AgentService) SendWithContext(ctx context.Context, text string, atts []components.Attachment, streamCallback func(agent.StreamChunk)) error {
	s.pendingMu.Lock()
	if s.pending {
		s.pendingMu.Unlock()
		return fmt.Errorf("request already in progress")
	}
	s.pending = true
	s.pendingMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentSession == nil {
		s.newSessionLocked()
	}

	llmAtts := convertAttachments(atts)
	var content any
	if len(llmAtts) > 0 {
		content = llm.ContentWithAttachments(text, llmAtts)
	} else {
		content = text
	}

	s.currentSession.Messages = append(s.currentSession.Messages, llm.UserMessage(content))
	s.currentSession.Metadata.MessageCount = len(s.currentSession.Messages)
	if err := s.sessionStore.Save(s.currentSession); err != nil {
		s.pendingMu.Lock()
		s.pending = false
		s.pendingMu.Unlock()
		return fmt.Errorf("save session: %w", err)
	}

	go s.runAgentWithContext(ctx, text, streamCallback)

	return nil
}

func (s *AgentService) CancelPending() {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	s.pending = false
}

func (s *AgentService) runAgentWithContext(ctx context.Context, userMessage string, callback func(agent.StreamChunk)) {
	defer func() {
		s.pendingMu.Lock()
		s.pending = false
		s.pendingMu.Unlock()
	}()

	orchAgent := orchestrator.New(agent.WithModel(s.currentModel))

	var accumulatedReasoning string

	resp, err := s.runner.RunStream(ctx, orchAgent, userMessage, func(chunk agent.StreamChunk) error {
		if chunk.Reasoning != "" {
			accumulatedReasoning += chunk.Reasoning
		}
		if callback != nil {
			callback(chunk)
		}
		return nil
	})

	if err != nil {
		if callback != nil {
			callback(agent.StreamChunk{
				Content: agent.FormatError(err),
				IsDone:  true,
			})
		}
		return
	}

	if resp != nil && resp.Content != "" {
		s.mu.Lock()
		if s.currentSession != nil {
			msg := llm.AssistantMessage(resp.Content)
			if accumulatedReasoning != "" {
				msg.Reasoning = accumulatedReasoning
			}
			s.currentSession.Messages = append(s.currentSession.Messages, msg)
			s.currentSession.Metadata.MessageCount = len(s.currentSession.Messages)
			_ = s.sessionStore.Save(s.currentSession)
		}
		s.mu.Unlock()
	}
}

func (s *AgentService) NewSession() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.newSessionLocked()
}

func (s *AgentService) newSessionLocked() {
	cwd := getWorkspacePath()
	s.currentSession = &session.Session{
		Messages: make([]llm.Message, 0),
		Metadata: session.SessionMetadata{
			Model:         s.currentModel,
			WorkspacePath: cwd,
		},
	}
}

func (s *AgentService) SaveSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentSession == nil {
		return nil
	}
	return s.sessionStore.Save(s.currentSession)
}

func (s *AgentService) LoadSession(id string) (*session.Session, error) {
	sess, err := s.sessionStore.Load(id)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.currentSession = sess
	if sess.Metadata.Model != "" {
		s.currentModel = sess.Metadata.Model
	}
	s.mu.Unlock()

	return sess, nil
}

func (s *AgentService) ListSessions() ([]session.SessionMetadataIndex, error) {
	result, err := s.sessionStore.List(session.ListOptions{
		Page:     1,
		PageSize: 50,
		Reverse:  true,
	})
	if err != nil {
		return nil, err
	}
	return result.Sessions, nil
}

func (s *AgentService) DeleteSession(id string) error {
	return s.sessionStore.Delete(id)
}

func (s *AgentService) AvailableAgents() []string {
	if s.bootstrap.AgentRegistry == nil {
		return nil
	}
	all := s.bootstrap.AgentRegistry.All()
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	return names
}

func (s *AgentService) SetModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentModel = model
	if s.currentSession != nil {
		s.currentSession.Metadata.Model = model
	}
}

func (s *AgentService) CurrentModel() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentModel
}

func (s *AgentService) AvailableModels() []string {
	cache, err := config.LoadModelsCache()
	if err != nil {
		return nil
	}
	var models []string
	for _, providerModels := range cache.Providers {
		models = append(models, providerModels...)
	}
	return models
}

func (s *AgentService) CurrentSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentSession == nil {
		return ""
	}
	return s.currentSession.ID
}

func (s *AgentService) Stop() {
	s.cancel()
	if s.sessionStore != nil {
		s.sessionStore.Close()
	}
	if s.bootstrap != nil {
		s.bootstrap.Shutdown(context.Background())
	}
}

func (s *AgentService) loadOrCreateSession() {
	result, err := s.sessionStore.List(session.ListOptions{
		Page:     1,
		PageSize: 1,
		Reverse:  true,
	})
	if err == nil && len(result.Sessions) > 0 {
		sess, err := s.sessionStore.Load(result.Sessions[0].ID)
		if err == nil {
			s.currentSession = sess
			if sess.Metadata.Model != "" {
				s.currentModel = sess.Metadata.Model
			}
			return
		}
	}
	s.newSessionLocked()
}

func selectInitialModel(cfg *config.Config) string {
	cache, err := config.LoadModelsCache()
	if err != nil || len(cache.Providers) == 0 {
		return ""
	}
	providerNames := make([]string, 0, len(cache.Providers))
	for name := range cache.Providers {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)
	for _, name := range providerNames {
		models := cache.Providers[name]
		if len(models) > 0 {
			return models[0]
		}
	}
	return ""
}

func getWorkspacePath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func convertAttachments(atts []components.Attachment) []llm.Attachment {
	if len(atts) == 0 {
		return nil
	}
	result := make([]llm.Attachment, 0, len(atts))
	for _, att := range atts {
		llmAtt := convertOneAttachment(att)
		if llmAtt != nil {
			result = append(result, *llmAtt)
		}
	}
	return result
}

func convertOneAttachment(att components.Attachment) *llm.Attachment {
	switch att.Type {
	case components.AttachmentImage:
		data, err := os.ReadFile(att.FilePath)
		if err != nil {
			return nil
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		mediaType := att.MimeType
		if mediaType == "" {
			mediaType = "image/png"
		}
		att := llm.NewImageAttachmentFromBase64(mediaType, encoded)
		return &att

	case components.AttachmentText:
		encoded := base64.StdEncoding.EncodeToString([]byte(att.Content))
		att := llm.NewFileAttachmentFromBase64("text/plain", encoded, att.Name)
		return &att

	default:
		if att.FilePath != "" {
			data, err := os.ReadFile(att.FilePath)
			if err != nil {
				return nil
			}
			encoded := base64.StdEncoding.EncodeToString(data)
			mediaType := att.MimeType
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
			att := llm.NewFileAttachmentFromBase64(mediaType, encoded, att.Name)
			return &att
		}
		return nil
	}
}
