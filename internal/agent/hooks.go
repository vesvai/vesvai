package agent

import (
	"github.com/vesvai/vesvai/internal/core/hook"
	"github.com/vesvai/vesvai/internal/llm"
)

type MessageInput struct {
	Text  string
	Calls []llm.ToolCall
}

var messageInputHook = hook.NewHook[MessageInput]()

func OnMessageInput(fn func(MessageInput) MessageInput) {
	messageInputHook.Add(fn)
}

func expandInput(input string) MessageInput {
	return messageInputHook.Apply(MessageInput{Text: input})
}
