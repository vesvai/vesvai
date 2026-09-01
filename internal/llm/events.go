package llm

type SelectMode string

const (
	SelectModeExact     SelectMode = "exact"
	SelectModePreferred SelectMode = "preferred"
)

type SelectRequest struct {
	Provider   string
	Model      string
	Mode       SelectMode
	ReplyTopic string
}

type SelectResult struct {
	Provider string
	Model    Model
	Err      error
}

type ModelsLoaded struct {
	Provider string
	Count    int
	Err      error
}
