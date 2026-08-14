package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agents/explorer"
	"github.com/vesvai/vesvai/internal/agents/orchestrator"
	"github.com/vesvai/vesvai/internal/agents/planner"
	"github.com/vesvai/vesvai/internal/bootstrap"
	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/skill"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/components"
)

var routeableAgents = []string{"orchestrator", "planner", "explorer"}

func stripLeadingMention(prompt string) (string, string) {
	trimmed := strings.TrimSpace(prompt)
	for _, name := range routeableAgents {
		rest, ok := strings.CutPrefix(trimmed, "@"+name)
		if !ok {
			continue
		}
		if rest != "" {
			if r := rune(rest[0]); isMentionRune(r) {
				continue
			}
		}
		return name, strings.TrimSpace(rest)
	}
	return "orchestrator", prompt
}

func isMentionRune(r rune) bool {
	return r == '/' || r == '.' || r == '-' || r == '_' ||
		(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

type AgentDriverConfig struct {
	Runner   *agent.Runner
	Store    *session.FileStore
	App      *bootstrap.App
	Approver *TUIApprover
}

type AgentDriver struct {
	mu       sync.Mutex
	cancel   context.CancelFunc
	runner   *agent.Runner
	store    *session.FileStore
	app      *bootstrap.App
	skills   *skill.Manager
	approver *TUIApprover

	model    string
	provider string
	models   []tui.ModelInfo
}

func NewAgentDriver(cfg AgentDriverConfig) *AgentDriver {
	d := &AgentDriver{
		runner:   cfg.Runner,
		store:    cfg.Store,
		app:      cfg.App,
		approver: cfg.Approver,
	}
	if cfg.App != nil {
		d.skills = cfg.App.Skills()
	}
	d.reloadModels()
	return d
}

func (d *AgentDriver) Cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cancel != nil {
		d.cancel()
	}
}

func (d *AgentDriver) setCancel(c context.CancelFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cancel = c
}

func (d *AgentDriver) clearCancel() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cancel = nil
}

func (d *AgentDriver) Run(ctx context.Context, req RunRequest, emit func(tui.StreamEvent)) {
	ctx, cancel := context.WithCancel(ctx)
	d.setCancel(cancel)
	defer d.clearCancel()

	agentName := "orchestrator"
	prompt := req.Text
	if name, rest := stripLeadingMention(prompt); name != "" {
		agentName = name
		prompt = rest
	}

	if d.skills != nil {
		prompt = d.expandSkills(prompt)
	}

	sessionID := agent.SessionIDFromContext(ctx)
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	ctx = agent.WithAgentContext(ctx, agentName, sessionID)
	if d.model != "" {
		ctx = agent.WithModelContext(ctx, d.model)
	}
	if len(req.History) > 0 {
		history := d.expandSkillHistory(req.History)
		ctx = agent.WithHistoryContext(ctx, history)
	}

	var content any = prompt
	if len(req.Attachments) > 0 {
		content = llm.ContentWithAttachments(prompt, toLLMAttachments(req.Attachments))
	}

	var target agent.Agent
	switch agentName {
	case "planner":
		target = planner.New()
	case "explorer":
		target = explorer.New()
	default:
		target = orchestrator.New(agent.WithModel(d.model))
	}

	runner := d.runner
	if r, err := d.runnerFor(); err == nil {
		runner = r
	}

	emit(tui.StreamEvent{Kind: tui.EventStart})

	usage := &tui.Usage{}
	_, err := runner.RunStreamContent(ctx, target, content, func(chunk agent.StreamChunk) error {
		d.mapChunk(chunk, usage, emit)
		return nil
	})

	if err != nil {
		if ctx.Err() != nil {
			emit(tui.StreamEvent{Kind: tui.EventError, Error: errInterrupted})
		} else {
			emit(tui.StreamEvent{Kind: tui.EventError, Error: err})
		}
	}
}

func (d *AgentDriver) mapChunk(chunk agent.StreamChunk, usage *tui.Usage, emit func(tui.StreamEvent)) {
	switch {
	case chunk.SubagentStart != nil:
		emit(tui.StreamEvent{
			Kind: tui.EventSubagentStart,
			Subagent: &tui.SubagentInfo{
				Name:   chunk.SubagentStart.Name,
				Prompt: chunk.SubagentStart.Prompt,
			},
		})

	case chunk.SubagentDone != nil:
		sd := chunk.SubagentDone
		emit(tui.StreamEvent{
			Kind:       tui.EventSubagentDone,
			SubagentID: sd.Name,
			SubagentResult: &tui.SubagentResult{
				Name:     sd.Name,
				Result:   sd.Result,
				Error:    sd.Error,
				Duration: sd.Duration,
			},
		})

	case chunk.ToolCall != nil:
		emit(tui.StreamEvent{
			Kind: tui.EventToolCall,
			ToolCall: &tui.ToolCallInfo{
				ToolName:   chunk.ToolCall.ToolName,
				Args:       chunk.ToolCall.Args,
				SubagentID: chunk.SubagentID,
			},
		})

	case chunk.ToolResult != nil:
		emit(tui.StreamEvent{
			Kind: tui.EventToolResult,
			ToolResult: &tui.ToolResultInfo{
				ToolName:   chunk.ToolResult.ToolName,
				Result:     chunk.ToolResult.Result,
				Error:      chunk.ToolResult.Error,
				Duration:   time.Duration(chunk.ToolResult.Duration) * time.Millisecond,
				SubagentID: chunk.SubagentID,
			},
		})

	case chunk.SubagentID != "" && chunk.Content != "":
		emit(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: chunk.SubagentID, Content: chunk.Content})

	case chunk.SubagentID != "" && chunk.Reasoning != "":
		emit(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: chunk.SubagentID, Content: chunk.Reasoning, SubagentReasoning: true})

	case chunk.Content != "":
		emit(tui.StreamEvent{Kind: tui.EventContent, Content: chunk.Content})

	case chunk.Reasoning != "":
		emit(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: chunk.Reasoning})
	}

	if chunk.IsDone && chunk.SubagentID == "" {
		if chunk.Usage != nil {
			total := chunk.Usage.TotalTokens
			if total == 0 {
				total = chunk.Usage.PromptTokens + chunk.Usage.CompletionTokens
			}
			usage.PromptTokens += chunk.Usage.PromptTokens
			usage.CompletionTokens += chunk.Usage.CompletionTokens
			usage.TotalTokens += total
		}
		if chunk.Final {
			emit(tui.StreamEvent{Kind: tui.EventDone, Usage: usage})
		}
	}
}

func (d *AgentDriver) expandSkillHistory(history []llm.Message) []llm.Message {
	if d.skills == nil {
		return history
	}
	out := make([]llm.Message, 0, len(history))
	for _, m := range history {
		if m.Role == llm.RoleUser {
			switch c := m.Content.(type) {
			case string:
				m.Content = d.expandSkills(c)
			case llm.Content:
				c.Text = d.expandSkills(c.Text)
				m.Content = c
			case *llm.Content:
				if c != nil {
					c.Text = d.expandSkills(c.Text)
					m.Content = c
				}
			}
		}
		out = append(out, m)
	}
	return out
}

func (d *AgentDriver) expandSkills(text string) string {
	names := map[string]bool{}
	if list, err := d.skills.List(); err == nil {
		for _, s := range list {
			names[s.Name] = true
		}
	}
	var b strings.Builder
	runes := []rune(text)
	for i := 0; i < len(runes); {
		if runes[i] != '/' {
			b.WriteRune(runes[i])
			i++
			continue
		}
		j := i + 1
		for j < len(runes) && isMentionRune(runes[j]) {
			j++
		}
		name := string(runes[i+1 : j])
		if j > i+1 && names[name] {
			if s, err := d.skills.Read(name); err == nil {
				b.WriteString(skill.StripFrontmatter(s.Content))
				i = j
				continue
			}
		}
		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}

func (d *AgentDriver) reloadModels() {
	d.models = nil
	cache, err := config.LoadModelsCache()
	if err == nil {
		for provider, ids := range cache.Providers {
			for _, id := range ids {
				d.models = append(d.models, tui.ModelInfo{
					Name:          id,
					Provider:      provider,
					Effort:        "-",
					ContextWindow: 0,
				})
			}
		}
	}
	if len(d.models) == 0 && d.app != nil {
		d.fetchModelsFromProviders()
	}
}

func (d *AgentDriver) fetchModelsFromProviders() {
	for _, p := range d.app.Config.Providers {
		provider, err := d.app.Providers.Create(p)
		if err != nil {
			continue
		}
		models, err := provider.ListModels(context.Background())
		if err != nil || len(models) == 0 {
			continue
		}
		ids := make([]string, 0, len(models))
		for _, m := range models {
			ids = append(ids, m.ID)
			d.models = append(d.models, tui.ModelInfo{
				Name:          m.ID,
				Provider:      p.Provider,
				Effort:        "-",
				ContextWindow: 0,
			})
		}
		_ = config.SaveProviderModels(p.Provider, ids)
	}
}

type Backend interface {
	ListSessions() ([]components.Session, error)
	LoadSession(id string) (*session.Session, error)
	DeleteSession(id string) error
	SaveSession(s *session.Session) error
	Models() []tui.ModelInfo
	SetModel(name string)
	NewSessionID() string
	CurrentHistory(conv *tui.Conversation) []llm.Message
	ConnectProvider(name, apiKey string) error
	SupportedProviders() []string
	MentionAgents() []components.Mention
	SlashCatalog() []components.Mention
}

func (d *AgentDriver) SupportedProviders() []string {
	if d.app == nil || d.app.Providers == nil {
		return nil
	}
	return d.app.Providers.Supported()
}

func (d *AgentDriver) Skills() *skill.Manager {
	return d.skills
}

func (d *AgentDriver) MentionAgents() []components.Mention {
	if d.app == nil || d.app.AgentRegistry == nil {
		return nil
	}
	names := make([]string, 0, len(d.app.AgentRegistry.All()))
	for name := range d.app.AgentRegistry.All() {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]components.Mention, 0, len(names))
	for _, name := range names {
		out = append(out, components.Mention{ID: name, Kind: "agent", Label: name})
	}
	return out
}

func (d *AgentDriver) SlashCatalog() []components.Mention {
	var out []components.Mention
	if d.skills != nil {
		if skills, err := d.skills.List(); err == nil {
			for _, s := range skills {
				out = append(out, components.Mention{ID: s.Name, Kind: "skill", Label: s.Name})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (d *AgentDriver) ApprovalRequests() <-chan permissionRequest {
	if d.approver == nil {
		return nil
	}
	return d.approver.reqs
}

func (d *AgentDriver) ListSessions() ([]components.Session, error) {
	if d.store == nil {
		return nil, nil
	}
	res, err := d.store.List(session.ListOptions{Page: 1, PageSize: 100, Reverse: true})
	if err != nil {
		return nil, err
	}
	out := make([]components.Session, 0, len(res.Sessions))
	for _, s := range res.Sessions {
		title := s.Title
		if title == "" {
			title = "(untitled)"
		}
		out = append(out, components.Session{
			ID:      s.ID,
			Title:   title,
			Updated: relativeTime(s.UpdatedAt),
		})
	}
	return out, nil
}

func (d *AgentDriver) LoadSession(id string) (*session.Session, error) {
	if d.store == nil {
		return nil, fmt.Errorf("session store unavailable")
	}
	return d.store.Load(id)
}

func (d *AgentDriver) DeleteSession(id string) error {
	if d.store == nil {
		return fmt.Errorf("session store unavailable")
	}
	return d.store.Delete(id)
}

func (d *AgentDriver) SaveSession(s *session.Session) error {
	if d.store == nil {
		return fmt.Errorf("session store unavailable")
	}
	return d.store.Save(s)
}

func (d *AgentDriver) NewSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

func (d *AgentDriver) Models() []tui.ModelInfo { return d.models }

func (d *AgentDriver) Model() string { return d.model }

func (d *AgentDriver) SetModel(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.model = name
}

func (d *AgentDriver) SetProvider(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.provider = name
}

func (d *AgentDriver) runnerFor() (*agent.Runner, error) {
	if d.app == nil {
		return d.runner, nil
	}

	d.mu.Lock()
	selected := d.provider
	d.mu.Unlock()

	name := selected
	if name == "" {
		name = d.app.DefaultProviderName()
	}
	if name == "" {
		return d.runner, nil
	}

	provider, err := d.app.CreateProviderByName(name)
	if err != nil {
		return nil, err
	}
	return d.app.CreateRunnerWithApprover(provider, d.approver), nil
}

func (d *AgentDriver) CurrentHistory(conv *tui.Conversation) []llm.Message {
	msgs := conv.Messages
	if len(msgs) <= 1 {
		return nil
	}
	return ConvToMessages(&tui.Conversation{Messages: msgs[:len(msgs)-1]})
}

func (d *AgentDriver) ConnectProvider(name, apiKey string) error {
	cfg := config.LLMConfig{Provider: name, APIKey: apiKey}
	provider, err := d.app.Providers.Create(cfg)
	if err != nil {
		return err
	}
	if err := config.AddProvider(cfg); err != nil {
		return err
	}
	models, err := provider.ListModels(context.Background())
	if err == nil {
		ids := make([]string, 0, len(models))
		for _, m := range models {
			ids = append(ids, m.ID)
		}
		_ = config.SaveProviderModels(name, ids)
	}
	d.reloadModels()
	if err != nil {
		return fmt.Errorf("provider saved, but model fetch failed: %w", err)
	}
	return nil
}
