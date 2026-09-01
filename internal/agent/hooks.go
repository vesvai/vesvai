package agent

import "github.com/vesvai/vesvai/internal/core/hook"

var messageInputHook = hook.NewHook[string]()

func OnMessageInput(fn func(string) string) {
	messageInputHook.Add(fn)
}

func expandInput(input string) string {
	return messageInputHook.Apply(input)
}
