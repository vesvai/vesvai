package subagent

import (
	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/session"
)

type HistoryReader interface {
	Messages(sessionID string) ([]session.Message, error)
}

var sessionReader HistoryReader

func SubAgentTools(reader HistoryReader) {
	sessionReader = reader
	tools.Register(subAgentTool())
	tools.Register(subAgentsStatusTool())
}
