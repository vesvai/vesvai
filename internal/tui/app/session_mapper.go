package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui"
)

func toolCallID(msgIdx, toolIdx int) string {
	return fmt.Sprintf("call_%d_%d", msgIdx, toolIdx)
}

func ConvToMessages(conv *tui.Conversation) []llm.Message {
	var out []llm.Message
	for i, m := range conv.Messages {
		switch m.Role {
		case tui.RoleUser:
			var content any = m.ContentText()
			if len(m.Attachments) > 0 {
				content = llm.ContentWithAttachments(m.ContentText(), toLLMAttachments(m.Attachments))
			}
			out = append(out, llm.NewMessage(llm.RoleUser, content))

		case tui.RoleAssistant:
			msg := llm.Message{
				Role:      llm.RoleAssistant,
				Content:   m.ContentText(),
				ToolCalls: nil,
			}
			if m.ThinkingText() != "" {
				msg.Reasoning = m.ThinkingText()
			}
			if len(m.Tools) > 0 {
				toolCalls := make([]llm.ToolCall, 0, len(m.Tools))
				for j, tc := range m.Tools {
					argsJSON := "{}"
					if b, err := json.Marshal(tc.Args); err == nil {
						argsJSON = string(b)
					}
					toolCalls = append(toolCalls, llm.ToolCall{
						Type: "function",
						ID:   toolCallID(i, j),
						Function: llm.Function{
							Name:      tc.Name,
							Arguments: argsJSON,
						},
					})
				}
				msg.ToolCalls = toolCalls
			}
			out = append(out, msg)

			for j, tc := range m.Tools {
				result := tc.Result
				if tc.Error != nil {
					result = "Error: " + tc.Error.Error()
				}
				out = append(out, llm.ToolMessage(result, toolCallID(i, j)))
			}
		}
	}
	return out
}

func toLLMAttachments(atts []*tui.Attachment) []llm.Attachment {
	if len(atts) == 0 {
		return nil
	}
	out := make([]llm.Attachment, 0, len(atts))
	for _, a := range atts {
		data, err := os.ReadFile(a.Path)
		if err != nil {
			continue
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		mediaType := mime.TypeByExtension(filepath.Ext(a.Path))
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		switch a.Kind {
		case "image":
			out = append(out, llm.NewImageAttachmentFromBase64(mediaType, encoded))
		default:
			out = append(out, llm.NewFileAttachmentFromBase64(mediaType, encoded, a.Name))
		}
	}
	return out
}

func MessagesToConv(conv *tui.Conversation, msgs []llm.Message) {
	conv.Reset()

	var pendingTools map[string]*tui.ToolCall

	for _, m := range msgs {
		switch m.Role {
		case llm.RoleUser:
			ta := contentTextAndAttachments(m.Content)
			msg := conv.AddUser(ta.text)
			msg.Attachments = ta.atts
			msg.Time = time.Now()

		case llm.RoleAssistant:
			ta := contentTextAndAttachments(m.Content)
			tm := conv.StartAssistant()
			tm.Content = ta.text
			tm.Status = tui.StatusDone
			if r, ok := m.Reasoning.(string); ok && r != "" {
				tm.Thinking = r
			}
			pendingTools = map[string]*tui.ToolCall{}
			for _, tc := range m.ToolCalls {
				args := map[string]any{}
				if tc.Function.Arguments != "" {
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				}
				t := conv.AddToolCall(tc.Function.Name, args)
				if tc.ID != "" {
					pendingTools[tc.ID] = t
				}
			}
			tm.RebuildParts()

		case llm.RoleTool:
			target := pendingTools[m.ToolCallID]
			if target == nil {
				target = lastToolOf(conv)
			}
			if target != nil && target.State == tui.ToolRunning {
				conv.FinishToolCall(target, toolMessageText(m.Content), nil, 0)
			}
		}
	}
}

func toolMessageText(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case llm.Content:
		return c.Text
	case *llm.Content:
		if c != nil {
			return c.Text
		}
	case []any:
		var sb strings.Builder
		for _, part := range c {
			if mm, ok := part.(map[string]any); ok {
				if text, ok := mm["text"].(string); ok {
					sb.WriteString(text)
				}
			}
		}
		return sb.String()
	}
	return ""
}

func ApplySessionParts(conv *tui.Conversation, mparts []session.MessageParts) {
	if len(mparts) != len(conv.Messages) {
		return
	}
	for mi, m := range conv.Messages {
		if m.Role != tui.RoleAssistant {
			continue
		}
		records := mparts[mi].Parts
		if len(records) == 0 {
			continue
		}
		m.Parts = nil
		m.Subagents = nil

		thinkRunes := []rune(m.ThinkingText())
		contentRunes := []rune(m.ContentText())
		thPos, tcPos := 0, 0
		for _, rec := range records {
			switch rec.Kind {
			case session.PartThinking:
				end := clampEnd(thPos, rec.TextLen, len(thinkRunes))
				m.Parts = append(m.Parts, tui.ThinkingPart(string(thinkRunes[thPos:end])))
				thPos = end
			case session.PartContent:
				end := clampEnd(tcPos, rec.TextLen, len(contentRunes))
				m.Parts = append(m.Parts, tui.ContentPart(string(contentRunes[tcPos:end])))
				tcPos = end
			case session.PartTool:
				if rec.ToolIdx >= 0 && rec.ToolIdx < len(m.Tools) {
					m.Parts = append(m.Parts, tui.Part{Kind: tui.PartTool, Tool: m.Tools[rec.ToolIdx]})
				}
			case session.PartSubagent:
				if rec.Subagent != nil {
					sa := recordToSubagent(rec.Subagent)
					m.Subagents = append(m.Subagents, sa)
					m.Parts = append(m.Parts, tui.Part{Kind: tui.PartSubagent, Subagent: sa})
				}
			}
		}
	}
}

func clampEnd(pos, length, max int) int {
	end := pos + length
	if end > max {
		return max
	}
	return end
}

func ConvToSessionParts(conv *tui.Conversation) []session.MessageParts {
	out := make([]session.MessageParts, 0, len(conv.Messages))
	for _, m := range conv.Messages {
		mp := session.MessageParts{}
		if m.Role == tui.RoleAssistant {
			tj := 0
			for i := range m.Parts {
				p := &m.Parts[i]
				rec := session.PartRecord{}
				switch p.Kind {
				case tui.PartThinking:
					rec.Kind = session.PartThinking
					rec.TextLen = utf8.RuneCountInString(p.ThinkingText())
				case tui.PartContent:
					rec.Kind = session.PartContent
					rec.TextLen = utf8.RuneCountInString(p.ContentText())
				case tui.PartTool:
					rec.Kind = session.PartTool
					rec.ToolIdx = tj
					tj++
				case tui.PartSubagent:
					rec.Kind = session.PartSubagent
					rec.Subagent = subagentToRecord(p.Subagent)
				}
				mp.Parts = append(mp.Parts, rec)
			}
		}
		out = append(out, mp)
	}
	return out
}

func subagentToRecord(sa *tui.Subagent) *session.SubagentRecord {
	if sa == nil {
		return nil
	}
	rec := &session.SubagentRecord{
		Name:       sa.Name,
		Prompt:     sa.Prompt,
		Thinking:   sa.ThinkingText(),
		Content:    sa.ContentText(),
		Result:     sa.Result,
		DurationMs: sa.Duration.Milliseconds(),
	}
	if sa.Error != nil {
		rec.Error = sa.Error.Error()
	}
	for _, tc := range sa.Tools {
		tr := session.SubagentToolRecord{
			Name:       tc.Name,
			Args:       tc.Args,
			Result:     tc.Result,
			DurationMs: tc.Duration.Milliseconds(),
		}
		if tc.Error != nil {
			tr.Error = tc.Error.Error()
		}
		rec.Tools = append(rec.Tools, tr)
	}
	tj := 0
	for i := range sa.Parts {
		p := &sa.Parts[i]
		pr := session.PartRecord{}
		switch p.Kind {
		case tui.PartThinking:
			pr.Kind = session.PartThinking
			pr.TextLen = utf8.RuneCountInString(p.ThinkingText())
		case tui.PartContent:
			pr.Kind = session.PartContent
			pr.TextLen = utf8.RuneCountInString(p.ContentText())
		case tui.PartTool:
			pr.Kind = session.PartTool
			pr.ToolIdx = tj
			tj++
		}
		rec.Parts = append(rec.Parts, pr)
	}
	return rec
}

func recordToSubagent(rec *session.SubagentRecord) *tui.Subagent {
	sa := &tui.Subagent{
		Name:     rec.Name,
		Prompt:   rec.Prompt,
		Thinking: rec.Thinking,
		Content:  rec.Content,
		Result:   rec.Result,
		Duration: time.Duration(rec.DurationMs) * time.Millisecond,
		Status:   tui.StatusDone,
	}
	if rec.Error != "" {
		sa.Error = errors.New(rec.Error)
		sa.Status = tui.StatusError
	}
	for _, tr := range rec.Tools {
		tc := &tui.ToolCall{
			Name:     tr.Name,
			Args:     tr.Args,
			Result:   tr.Result,
			Duration: time.Duration(tr.DurationMs) * time.Millisecond,
			State:    tui.ToolSuccess,
		}
		if tr.Error != "" {
			tc.Error = errors.New(tr.Error)
			tc.State = tui.ToolError
		}
		sa.Tools = append(sa.Tools, tc)
	}
	thinkRunes := []rune(rec.Thinking)
	contentRunes := []rune(rec.Content)
	thPos, tcPos := 0, 0
	for _, pr := range rec.Parts {
		switch pr.Kind {
		case session.PartThinking:
			end := clampEnd(thPos, pr.TextLen, len(thinkRunes))
			sa.Parts = append(sa.Parts, tui.ThinkingPart(string(thinkRunes[thPos:end])))
			thPos = end
		case session.PartContent:
			end := clampEnd(tcPos, pr.TextLen, len(contentRunes))
			sa.Parts = append(sa.Parts, tui.ContentPart(string(contentRunes[tcPos:end])))
			tcPos = end
		case session.PartTool:
			if pr.ToolIdx >= 0 && pr.ToolIdx < len(sa.Tools) {
				sa.Parts = append(sa.Parts, tui.Part{Kind: tui.PartTool, Tool: sa.Tools[pr.ToolIdx]})
			}
		}
	}
	return sa
}

type textAndAttachments struct {
	text string
	atts []*tui.Attachment
}

func contentTextAndAttachments(content any) textAndAttachments {
	out := textAndAttachments{}
	switch c := content.(type) {
	case string:
		out.text = c
	case llm.Content:
		out.text = c.Text
		for _, a := range c.Attachments {
			if a.Type == llm.AttachmentTypeImage {
				out.atts = append(out.atts, &tui.Attachment{
					Path: a.FileName, Name: a.FileName, Kind: "image",
				})
			} else {
				name := a.FileName
				if name == "" {
					name = "attached file"
				}
				out.atts = append(out.atts, &tui.Attachment{Path: name, Name: name, Kind: "file"})
			}
		}
	case *llm.Content:
		if c != nil {
			out = contentTextAndAttachments(*c)
		}
	}
	return out
}

func lastToolOf(conv *tui.Conversation) *tui.ToolCall {
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		m := conv.Messages[i]
		if len(m.Tools) > 0 {
			return m.Tools[len(m.Tools)-1]
		}
	}
	return nil
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func titleFromText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	if len(text) > 60 {
		text = text[:57] + "..."
	}
	if text == "" {
		return "(untitled)"
	}
	return text
}
