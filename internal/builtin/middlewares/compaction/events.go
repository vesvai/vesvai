package compaction

import "github.com/vesvai/vesvai/internal/llm"

const (
	TopicCompactionStarted  = "agent.compaction.started"
	TopicCompactionFinished = "agent.compaction.finished"
)

type Event struct {
	AgentID           string
	AgentName         string
	Strategy          string
	Messages          int
	Tokens            int
	CompactedMessages []llm.Message
}
