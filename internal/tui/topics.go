package tui

const (
	TopicSubmit = "tui.message.submitted"
)

type SubmitEvent struct {
	Message string
}
