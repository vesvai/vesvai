package middlewares

import (
	"github.com/vesvai/vesvai/internal/agent/middlewares"
)

func Create() {
	middlewares.Register("loop-detector", NewLoopDetector())
	middlewares.Register("redaction", NewRedaction())
	middlewares.Register("retry", NewRetry())
}
