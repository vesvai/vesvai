package tui

import (
	"fmt"
	"time"
)

type Role int

const (
	RoleUser Role = iota
	RoleAssistant
)

func (r Role) String() string {
	if r == RoleUser {
		return "you"
	}
	return "assistant"
}

type Status int

const (
	StatusIdle Status = iota
	StatusRunning
	StatusDone
	StatusError
)

type ToolState int

const (
	ToolRunning ToolState = iota
	ToolSuccess
	ToolError
)

type ToolCall struct {
	Name     string
	Args     map[string]any
	Result   string
	Error    error
	Duration time.Duration
	State    ToolState
	Expanded bool

	spinnerFrame int
}

type PartKind int

const (
	PartThinking PartKind = iota
	PartContent
	PartTool
	PartSubagent
)

type Part struct {
	Kind     PartKind
	Thinking string
	Content  string
	Tool     *ToolCall
	Subagent *Subagent

	ThinkingExpanded bool

	renderThinking []Line
	renderContent  []Line
	thinkingDirty  bool
	contentDirty   bool
}

func (p *Part) RenderedThinking() []Line     { return p.renderThinking }
func (p *Part) SetRenderedThinking(l []Line) { p.renderThinking = l; p.thinkingDirty = false }
func (p *Part) ThinkingDirty() bool          { return p.thinkingDirty }

func (p *Part) RenderedContent() []Line     { return p.renderContent }
func (p *Part) SetRenderedContent(l []Line) { p.renderContent = l; p.contentDirty = false }
func (p *Part) ContentDirty() bool          { return p.contentDirty }

func ThinkingPart(text string) Part {
	return Part{Kind: PartThinking, Thinking: text, thinkingDirty: true}
}

func ContentPart(text string) Part {
	return Part{Kind: PartContent, Content: text, contentDirty: true}
}

type Subagent struct {
	Name     string
	Prompt   string
	Content  string
	Thinking string
	Status   Status
	Error    error
	Result   string
	Duration time.Duration

	Tools []*ToolCall
	Parts []Part
}

func (sa *Subagent) RebuildParts() {
	sa.Parts = nil
	if sa.Thinking != "" {
		sa.Parts = append(sa.Parts, Part{Kind: PartThinking, Thinking: sa.Thinking, thinkingDirty: true})
	}
	if sa.Content != "" {
		sa.Parts = append(sa.Parts, Part{Kind: PartContent, Content: sa.Content, contentDirty: true})
	}
	for _, tc := range sa.Tools {
		sa.Parts = append(sa.Parts, Part{Kind: PartTool, Tool: tc})
	}
}

type Message struct {
	ID       string
	Role     Role
	Content  string
	Thinking string
	Time     time.Time
	Status   Status
	Error    error

	Tools     []*ToolCall
	Subagents []*Subagent

	Parts []Part

	Attachments []*Attachment

	idx int

	renderContent []Line
	contentDirty  bool
}

func (m *Message) Height() int {
	h := 0
	for _, l := range m.renderContent {
		h += len(l)
	}
	for i := range m.Parts {
		p := &m.Parts[i]
		for _, l := range p.renderContent {
			h += len(l)
		}
		for _, l := range p.renderThinking {
			h += len(l)
		}
	}
	h += len(m.Tools) * 2
	return h
}

func (m *Message) IDx() int {
	if m.idx == 0 {
		return 1
	}
	return m.idx
}

func (m *Message) RenderedContent() []Line     { return m.renderContent }
func (m *Message) SetRenderedContent(l []Line) { m.renderContent = l; m.contentDirty = false }
func (m *Message) ContentDirty() bool          { return m.contentDirty }

func (m *Message) RebuildParts() {
	m.Parts = nil
	if m.Thinking != "" {
		m.Parts = append(m.Parts, Part{Kind: PartThinking, Thinking: m.Thinking, thinkingDirty: true})
	}
	if m.Content != "" {
		m.Parts = append(m.Parts, Part{Kind: PartContent, Content: m.Content, contentDirty: true})
	}
	for _, tc := range m.Tools {
		m.Parts = append(m.Parts, Part{Kind: PartTool, Tool: tc})
	}
}

func (m *Message) EnsureParts() {
	if len(m.Parts) == 0 && (m.Thinking != "" || m.Content != "" || len(m.Tools) > 0) {
		m.RebuildParts()
	}
}

type Conversation struct {
	Messages []*Message
	revision int64
	seq      int
}

func (c *Conversation) Revision() int64 { return c.revision }

func (c *Conversation) AddUser(text string) *Message {
	m := &Message{
		ID:           c.nextID("msg"),
		Role:         RoleUser,
		Content:      text,
		Time:         time.Now(),
		Status:       StatusDone,
		contentDirty: true,
	}
	c.Messages = append(c.Messages, m)
	m.idx = len(c.Messages)
	c.revision++
	return m
}

func (c *Conversation) StartAssistant() *Message {
	m := &Message{
		ID:           c.nextID("msg"),
		Role:         RoleAssistant,
		Time:         time.Now(),
		Status:       StatusRunning,
		contentDirty: true,
	}
	c.Messages = append(c.Messages, m)
	m.idx = len(c.Messages)
	c.revision++
	return m
}

func (c *Conversation) Current() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
	return c.Messages[len(c.Messages)-1]
}

func (c *Conversation) AppendContent(delta string) {
	m := c.Current()
	if m == nil {
		return
	}
	m.Content += delta
	if n := len(m.Parts); n > 0 && m.Parts[n-1].Kind == PartContent {
		p := &m.Parts[n-1]
		p.Content += delta
		p.contentDirty = true
	} else {
		m.Parts = append(m.Parts, Part{Kind: PartContent, Content: delta, contentDirty: true})
	}
	c.revision++
}

func (c *Conversation) AppendThinking(delta string) {
	m := c.Current()
	if m == nil {
		return
	}
	m.Thinking += delta
	if n := len(m.Parts); n > 0 && m.Parts[n-1].Kind == PartThinking {
		p := &m.Parts[n-1]
		p.Thinking += delta
		p.thinkingDirty = true
	} else {
		m.Parts = append(m.Parts, Part{Kind: PartThinking, Thinking: delta, thinkingDirty: true})
	}
	c.revision++
}

func (c *Conversation) AddToolCall(name string, args map[string]any) *ToolCall {
	m := c.Current()
	if m == nil {
		m = c.StartAssistant()
	}
	tc := &ToolCall{Name: name, Args: args, State: ToolRunning}
	m.Tools = append(m.Tools, tc)
	m.Parts = append(m.Parts, Part{Kind: PartTool, Tool: tc})
	c.revision++
	return tc
}

func (c *Conversation) FinishToolCall(tc *ToolCall, result string, err error, d time.Duration) {
	if tc == nil {
		return
	}
	tc.Result = result
	tc.Error = err
	tc.Duration = d
	if err != nil {
		tc.State = ToolError
	} else {
		tc.State = ToolSuccess
	}
	c.revision++
}

func (c *Conversation) AddSubagent(name, prompt string) (*Subagent, string) {
	m := c.Current()
	if m == nil {
		m = c.StartAssistant()
	}
	sa := &Subagent{Name: name, Prompt: prompt, Status: StatusRunning}
	m.Subagents = append(m.Subagents, sa)
	m.Parts = append(m.Parts, Part{Kind: PartSubagent, Subagent: sa})
	id := SubagentBlockID(m, len(m.Subagents)-1)
	c.revision++
	return sa, id
}

func (c *Conversation) SubagentByBlockID(id string) *Subagent {
	for _, m := range c.Messages {
		for i, sa := range m.Subagents {
			if SubagentBlockID(m, i) == id {
				return sa
			}
		}
	}
	return nil
}

func (c *Conversation) SubagentByID(id string) *Subagent {
	m := c.Current()
	if m == nil {
		return nil
	}
	for i, sa := range m.Subagents {
		if SubagentBlockID(m, i) == id {
			return sa
		}
	}
	for i := len(m.Subagents) - 1; i >= 0; i-- {
		if m.Subagents[i].Name == id {
			return m.Subagents[i]
		}
	}
	return nil
}

func (c *Conversation) AppendSubagentContent(sa *Subagent, delta string) {
	if sa == nil {
		return
	}
	sa.Content += delta
	if n := len(sa.Parts); n > 0 && sa.Parts[n-1].Kind == PartContent {
		p := &sa.Parts[n-1]
		p.Content += delta
		p.contentDirty = true
	} else {
		sa.Parts = append(sa.Parts, Part{Kind: PartContent, Content: delta, contentDirty: true})
	}
	c.revision++
}

func (c *Conversation) AppendSubagentThinking(sa *Subagent, delta string) {
	if sa == nil {
		return
	}
	sa.Thinking += delta
	if n := len(sa.Parts); n > 0 && sa.Parts[n-1].Kind == PartThinking {
		p := &sa.Parts[n-1]
		p.Thinking += delta
		p.thinkingDirty = true
	} else {
		sa.Parts = append(sa.Parts, Part{Kind: PartThinking, Thinking: delta, thinkingDirty: true})
	}
	c.revision++
}

func (c *Conversation) AddSubagentTool(sa *Subagent, name string, args map[string]any) *ToolCall {
	if sa == nil {
		return nil
	}
	tc := &ToolCall{Name: name, Args: args, State: ToolRunning}
	sa.Tools = append(sa.Tools, tc)
	sa.Parts = append(sa.Parts, Part{Kind: PartTool, Tool: tc})
	c.revision++
	return tc
}

func (c *Conversation) FinishSubagentTool(sa *Subagent, tc *ToolCall, result string, err error, d time.Duration) {
	if sa == nil || tc == nil {
		return
	}
	tc.Result = result
	tc.Error = err
	tc.Duration = d
	if err != nil {
		tc.State = ToolError
	} else {
		tc.State = ToolSuccess
	}
	c.revision++
}

func (c *Conversation) FinishSubagent(sa *Subagent, result string, err error, d time.Duration) {
	if sa == nil {
		return
	}
	sa.Result = result
	sa.Error = err
	sa.Duration = d
	if err != nil {
		sa.Status = StatusError
	} else {
		sa.Status = StatusDone
	}
	c.revision++
}

func (c *Conversation) MarkDone() {
	if m := c.Current(); m != nil {
		m.Status = StatusDone
		c.revision++
	}
}

func (c *Conversation) MarkError(err error) {
	if m := c.Current(); m != nil {
		m.Status = StatusError
		m.Error = err
		c.revision++
	}
}

func (c *Conversation) BumpRevision() { c.revision++ }

func (c *Conversation) Reset() {
	c.Messages = nil
	c.revision++
	c.seq = 0
}

func (c *Conversation) TruncateAfter(m *Message) {
	for i, msg := range c.Messages {
		if msg == m {
			c.Messages = c.Messages[:i+1]
			c.revision++
			return
		}
	}
}

func ThinkingPartID(m *Message, k int) string { return fmt.Sprintf("m%d:think%d", m.IDx(), k) }

func ToolPartID(m *Message, j int) string { return fmt.Sprintf("m%d:t%d", m.IDx(), j) }

func SubagentBlockID(m *Message, j int) string { return fmt.Sprintf("m%d:sa%d", m.IDx(), j) }

func (c *Conversation) TogglePartByID(id string) bool {
	for _, m := range c.Messages {
		if m.Role != RoleAssistant {
			continue
		}
		th, tj := 0, 0
		for i := range m.Parts {
			p := &m.Parts[i]
			switch p.Kind {
			case PartThinking:
				if ThinkingPartID(m, th) == id {
					p.ThinkingExpanded = !p.ThinkingExpanded
					c.revision++
					return true
				}
				th++
			case PartTool:
				if ToolPartID(m, tj) == id {
					p.Tool.Expanded = !p.Tool.Expanded
					c.revision++
					return true
				}
				tj++
			}
		}
	}
	return false
}

func (c *Conversation) nextID(prefix string) string {
	c.seq++
	return fmt.Sprintf("%s_%d", prefix, c.seq)
}

type ModelInfo struct {
	Name          string
	Provider      string
	Effort        string
	ContextWindow int
}

type Model struct {
	Conv  *Conversation
	Busy  bool
	Usage Usage
	Err   error
	Step  int
	Model string

	Provider      string
	Effort        string
	ContextWindow int

	SessionName string
	StatusMsg   string
	StatusMsgAt time.Time
}

func NewModel(model string) *Model {
	return &Model{
		Conv:  &Conversation{},
		Model: model,
	}
}

func (m *Model) UsageFraction() float64 {
	if m.ContextWindow <= 0 {
		return 0
	}
	return float64(m.Usage.TotalTokens) / float64(m.ContextWindow)
}

func (m *Model) SetStatusMsg(msg string) {
	m.StatusMsg = msg
	m.StatusMsgAt = time.Now()
}

func (m *Model) StatusMsgFresh(within time.Duration) bool {
	if m.StatusMsg == "" {
		return false
	}
	return time.Since(m.StatusMsgAt) < within
}

func (m *Model) Apply(ev StreamEvent) {
	switch ev.Kind {
	case EventStart:
		m.Busy = true
		m.Err = nil
		m.Step++
		m.Conv.StartAssistant()

	case EventReasoning:
		m.Conv.AppendThinking(ev.Reasoning)

	case EventContent:
		m.Conv.AppendContent(ev.Content)

	case EventToolCall:
		if ev.ToolCall != nil {
			if ev.ToolCall.SubagentID != "" {
				if sa := m.Conv.SubagentByID(ev.ToolCall.SubagentID); sa != nil {
					m.Conv.AddSubagentTool(sa, ev.ToolCall.ToolName, ev.ToolCall.Args)
				}
			} else {
				m.Conv.AddToolCall(ev.ToolCall.ToolName, ev.ToolCall.Args)
			}
		}

	case EventToolResult:
		if ev.ToolResult != nil {
			if ev.ToolResult.SubagentID != "" {
				if sa := m.Conv.SubagentByID(ev.ToolResult.SubagentID); sa != nil && len(sa.Tools) > 0 {
					m.Conv.FinishSubagentTool(sa, sa.Tools[len(sa.Tools)-1],
						ev.ToolResult.Result, ev.ToolResult.Error, ev.ToolResult.Duration)
				}
			} else {
				cur := m.Conv.Current()
				var tc *ToolCall
				if cur != nil && len(cur.Tools) > 0 {
					tc = cur.Tools[len(cur.Tools)-1]
				}
				m.Conv.FinishToolCall(tc, ev.ToolResult.Result, ev.ToolResult.Error, ev.ToolResult.Duration)
			}
		}

	case EventSubagentStart:
		if ev.Subagent != nil {
			m.Conv.AddSubagent(ev.Subagent.Name, ev.Subagent.Prompt)
		}

	case EventSubagentChunk:
		if sa := m.Conv.SubagentByID(ev.SubagentID); sa != nil {
			if ev.SubagentReasoning {
				m.Conv.AppendSubagentThinking(sa, ev.Content)
			} else {
				m.Conv.AppendSubagentContent(sa, ev.Content)
			}
		}

	case EventSubagentDone:
		if sa := m.Conv.SubagentByID(ev.SubagentID); sa != nil {
			res := ev.SubagentResult
			if res == nil {
				res = &SubagentResult{}
			}
			m.Conv.FinishSubagent(sa, res.Result, res.Error, res.Duration)
		}

	case EventDone:
		m.Busy = false
		if ev.Usage != nil {
			u := *ev.Usage
			if u.TotalTokens == 0 {
				u.TotalTokens = u.PromptTokens + u.CompletionTokens
			}
			m.Usage = u
		}
		m.Conv.MarkDone()

	case EventError:
		m.Busy = false
		m.Err = ev.Error
		m.Conv.MarkError(ev.Error)
	}
}
