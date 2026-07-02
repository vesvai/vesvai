package llm

type ResponseFormat struct {
	Type   string `json:"type"`
	Schema any    `json:"schema,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type Request struct {
	Model             string          `json:"model"`
	Messages          []Message       `json:"messages"`
	Temperature       float64         `json:"temperature,omitempty"`
	TopP              float64         `json:"top_p,omitempty"`
	MaxTokens         int             `json:"max_tokens,omitempty"`
	Stream            bool            `json:"stream,omitempty"`
	Tools             []Tool          `json:"tools,omitempty"`
	ToolChoice        any             `json:"tool_choice,omitempty"`
	ResponseFormat    *ResponseFormat `json:"response_format,omitempty"`
	ParallelToolCalls bool            `json:"parallel_tool_calls,omitempty"`
	N                 int             `json:"n,omitempty"`
	PresencePenalty   float64         `json:"presence_penalty,omitempty"`
	FrequencyPenalty  float64         `json:"frequency_penalty,omitempty"`
	User              string          `json:"user,omitempty"`
}

func NewRequest(model string, messages []Message) *Request {
	return &Request{
		Model:    model,
		Messages: messages,
	}
}

func (r *Request) WithTemperature(temp float64) *Request {
	r.Temperature = temp
	return r
}

func (r *Request) WithMaxTokens(maxTokens int) *Request {
	r.MaxTokens = maxTokens
	return r
}

func (r *Request) WithTopP(topP float64) *Request {
	r.TopP = topP
	return r
}

func (r *Request) WithTools(tools []Tool) *Request {
	r.Tools = tools
	return r
}

func (r *Request) WithToolChoice(choice any) *Request {
	r.ToolChoice = choice
	return r
}

func (r *Request) WithResponseFormat(format *ResponseFormat) *Request {
	r.ResponseFormat = format
	return r
}

func (r *Request) WithStream(stream bool) *Request {
	r.Stream = stream
	return r
}

func (r *Request) WithN(n int) *Request {
	r.N = n
	return r
}

func (r *Request) WithPresencePenalty(penalty float64) *Request {
	r.PresencePenalty = penalty
	return r
}

func (r *Request) WithFrequencyPenalty(penalty float64) *Request {
	r.FrequencyPenalty = penalty
	return r
}

func (r *Request) WithUser(user string) *Request {
	r.User = user
	return r
}
