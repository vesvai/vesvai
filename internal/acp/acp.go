package acp

import (
	"context"
	"net/http"
	"sync"
	"time"

	json "github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/core/event"
	"github.com/vesvai/vesvai/internal/core/logger"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/vfs"
)

type ACPSession struct {
	ID        string
	Agent     *agent.Agent
	CreatedAt time.Time
	Cwd       string

	VesvaiSessionID string

	cancel context.CancelFunc
	mu     sync.Mutex

	reqID any
}

func (s *ACPSession) SetCancel(cancel context.CancelFunc, reqID any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.cancel = cancel
	s.reqID = reqID
}

func (s *ACPSession) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

type Server struct {
	cfg      *config.Config
	bus      event.Bus
	log      *logger.Logger
	sessions *session.Manager
	llmMgr   *llm.Manager
	fs       *vfs.VFS

	transport Transport
	client    *Client

	newAgentFn func() (*agent.Agent, error)

	responseHook func(id any, body json.RawMessage) error

	active map[string]*ACPSession
	mu     sync.RWMutex

	closed chan struct{}
	once   sync.Once
}

func New(
	cfg *config.Config,
	bus event.Bus,
	log *logger.Logger,
	fs *vfs.VFS,
	sessions *session.Manager,
	llmMgr *llm.Manager,
) *Server {
	s := &Server{
		cfg:      cfg,
		bus:      bus,
		log:      log,
		sessions: sessions,
		llmMgr:   llmMgr,
		fs:       fs,
		active:   make(map[string]*ACPSession),
		closed:   make(chan struct{}),
	}
	s.client = NewClient(s.sendRaw)
	s.newAgentFn = s.newOrchestratorAgent
	return s
}

func (s *Server) Serve(ctx context.Context, transport Transport) error {
	s.transport = transport
	inbound, err := transport.Start(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			transport.Close()
			return ctx.Err()
		case msg, ok := <-inbound:
			if !ok {
				transport.Close()
				return nil
			}
			s.handleFrame(ctx, msg)
		}
	}
}

func (s *Server) handleFrame(ctx context.Context, msg Message) {
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      any             `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}

	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		s.writeError(nil, ErrCodeParse, "parse error", nil)
		return
	}

	if envelope.JSONRPC != "2.0" {
		s.writeError(envelope.ID, ErrCodeInvalidReq, "invalid jsonrpc version", nil)
		return
	}

	if envelope.Method == "" && envelope.ID != nil {
		s.client.DeliverResponse(envelope.ID, msg.Body)
		return
	}

	s.dispatch(ctx, envelope.ID, envelope.Method, envelope.Params)
}

func (s *Server) dispatch(ctx context.Context, id any, method string, params json.RawMessage) {
	var result any
	var handlerErr error

	switch method {
	case "initialize":
		result, handlerErr = s.handleInitialize(ctx, params)
	case "session/new":
		result, handlerErr = s.handleSessionNew(ctx, params)
	case "session/load":
		result, handlerErr = s.handleSessionLoad(ctx, params)
	case "session/resume":
		result, handlerErr = s.handleSessionResume(ctx, params)
	case "session/prompt":
		result, handlerErr = s.handleSessionPrompt(ctx, params, id)
	case "session/delete":
		result, handlerErr = s.handleSessionDelete(ctx, params)
	case "session/close":
		result, handlerErr = s.handleSessionClose(ctx, params)
	case "session/list":
		result, handlerErr = s.handleSessionList(ctx, params)
	case "session/cancel":
		handlerErr = s.handleSessionCancel(ctx, params)
		if handlerErr == nil {
			return
		}
	case "$/cancelRequest":
		handlerErr = s.handleCancelRequest(ctx, params)
		if handlerErr == nil {
			return
		}
	default:
		s.writeError(id, ErrCodeMethodNotFound, "method not found: "+method, nil)
		return
	}

	if id == nil {
		return
	}

	if handlerErr != nil {
		if rpcErr, ok := handlerErr.(*RPCError); ok {
			s.writeError(id, rpcErr.Code, rpcErr.Message, rpcErr.Data)
		} else {
			s.writeError(id, ErrCodeInternal, handlerErr.Error(), nil)
		}
		return
	}

	s.writeResult(id, result)
}

func (s *Server) sendRaw(body json.RawMessage) error {
	if s.transport == nil {
		return nil
	}
	return s.transport.Send(Message{Body: body})
}

func (s *Server) writeResult(id any, result any) {
	if id == nil {
		return
	}
	data, err := json.Marshal(result)
	if err != nil {
		s.writeError(id, ErrCodeInternal, "failed to marshal result", nil)
		return
	}
	resp := Response{JSONRPC: "2.0", ID: id, Result: data}
	raw, _ := json.Marshal(resp)
	s.sendResponse(id, raw)
}

func (s *Server) writeError(id any, code int, message string, data any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message, Data: data},
	}
	raw, _ := json.Marshal(resp)
	s.sendResponse(id, raw)
}

func (s *Server) sendResponse(id any, body json.RawMessage) {
	if s.responseHook != nil {
		s.responseHook(id, body)
		return
	}
	s.sendRaw(body)
}

func (s *Server) notify(sessionID string, update SessionUpdate) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "session/update",
		"params": map[string]any{
			"sessionId": sessionID,
			"update":    update,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if s.transport != nil {
		s.transport.Send(Message{SessionId: sessionID, Body: raw})
	}
}

func (s *Server) HTTPHandler() http.Handler {
	t := NewHTTPTransport(s)
	s.transport = t
	return t
}

func (s *Server) addSession(acpSess *ACPSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[acpSess.ID] = acpSess
}

func (s *Server) getSession(id string) (*ACPSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acpSess, ok := s.active[id]
	return acpSess, ok
}

func (s *Server) removeSession(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if acpSess, ok := s.active[id]; ok {
		acpSess.Cancel()
	}
	delete(s.active, id)
}

func (s *Server) resolveModel(provider, model string) (llm.Provider, llm.Model, error) {
	if s.llmMgr == nil {
		return nil, llm.Model{}, ErrModelUnavailable
	}
	s.llmMgr.WaitUntilReady()

	if provider != "" && model != "" {
		return s.resolveExact(provider, model)
	}

	reply := "acp.select.reply." + uuid.NewString()
	resultCh := make(chan llm.SelectResult, 1)
	handler := func(res llm.SelectResult) {
		select {
		case resultCh <- res:
		default:
		}
	}
	_ = s.bus.SubscribeOnce(reply, handler)
	defer s.bus.Unsubscribe(reply, handler)

	s.bus.Publish(event.TopicModelSelect, llm.SelectRequest{
		Mode:       llm.SelectModePreferred,
		Provider:   provider,
		Model:      model,
		ReplyTopic: reply,
	})

	select {
	case res := <-resultCh:
		if res.Err != nil {
			return nil, llm.Model{}, res.Err
		}
		return s.resolveExact(res.Provider, res.Model.ID)
	case <-time.After(30 * time.Second):
		return nil, llm.Model{}, ErrModelTimeout
	}
}

func (s *Server) resolveExact(provider, model string) (llm.Provider, llm.Model, error) {
	if provider == "" {
		p, m, err := s.findModelAnywhere(model)
		if err != nil {
			return nil, llm.Model{}, err
		}
		return p, m, nil
	}
	p, err := s.llmMgr.Provider(provider)
	if err != nil {
		return nil, llm.Model{}, err
	}
	models, err := s.llmMgr.Models(provider)
	if err != nil {
		return nil, llm.Model{}, err
	}
	for _, m := range models {
		if m.ID == model || m.Name == model {
			return p, m, nil
		}
	}
	return nil, llm.Model{}, ErrModelNotFound
}

func (s *Server) findModelAnywhere(model string) (llm.Provider, llm.Model, error) {
	for _, p := range s.cfg.Providers {
		if p, m, err := s.resolveExact(p.Provider, model); err == nil {
			return p, m, nil
		}
	}
	return nil, llm.Model{}, ErrModelNotFound
}

type acpError struct{}

var (
	ErrModelUnavailable = &RPCError{Code: ErrCodeInternal, Message: "llm manager unavailable"}
	ErrModelTimeout     = &RPCError{Code: ErrCodeInternal, Message: "timed out selecting model"}
	ErrModelNotFound    = &RPCError{Code: ErrCodeInternal, Message: "model not found"}
	_                   = acpError{}
)
