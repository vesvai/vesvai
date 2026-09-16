package reminder

import (
	"fmt"
	"strings"
)

type Reminder struct {
	Tag     string
	Attrs   map[string]string
	Content string
}

func New(tag, content string, kvPairs ...string) Reminder {
	attrs := make(map[string]string, len(kvPairs)/2)
	for i := 0; i+1 < len(kvPairs); i += 2 {
		attrs[kvPairs[i]] = kvPairs[i+1]
	}
	return Reminder{Tag: tag, Attrs: attrs, Content: content}
}

func (r Reminder) Format() string {
	var b strings.Builder
	b.WriteString("<system-reminder")
	b.WriteString(fmt.Sprintf(" tag=%q", r.Tag))
	for k, v := range r.Attrs {
		b.WriteString(fmt.Sprintf(" %s=%q", k, v))
	}
	b.WriteString(">\n")
	b.WriteString(r.Content)
	b.WriteString("\n</system-reminder>")
	return b.String()
}

func FormatAll(reminders []Reminder) string {
	if len(reminders) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<system-reminders>\n")
	for i, r := range reminders {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(r.Format())
	}
	b.WriteString("\n</system-reminders>")
	return b.String()
}

func SubAgentDone(name string, taskIDs []string, output string) Reminder {
	taskStr := ""
	if len(taskIDs) > 0 {
		taskStr = strings.Join(taskIDs, ",")
	}
	content := fmt.Sprintf("Subagent %q finished.\nResponse:\n%s", name, output)
	return New("subagent", content, "task", taskStr, "agent", name)
}

func SubAgentFailed(name string, taskIDs []string, errMsg string) Reminder {
	taskStr := ""
	if len(taskIDs) > 0 {
		taskStr = strings.Join(taskIDs, ",")
	}
	content := fmt.Sprintf("Subagent %q failed: %s", name, errMsg)
	return New("subagent", content, "task", taskStr, "agent", name)
}
