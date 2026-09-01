package llm

type StreamChoice struct {
	Index        int           `json:"index"`
	Delta        *MessageDelta `json:"delta,omitempty"`
	FinishReason *FinishReason `json:"finish_reason,omitempty"`
}

type StreamResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
	Choices           []StreamChoice `json:"choices"`
	Usage             *Usage         `json:"usage,omitempty"`
}

type StreamChunk struct {
	Content      string
	Reasoning    string
	ToolCalls    []ToolCall
	FinishReason FinishReason
	IsDone       bool
	Usage        *Usage
}

func (s *StreamResponse) ToChunk() StreamChunk {
	chunk := StreamChunk{
		IsDone: false,
	}

	if len(s.Choices) == 0 {
		return chunk
	}

	choice := s.Choices[0]

	if choice.Delta == nil {
		return chunk
	}

	if choice.Delta.Content != nil {
		chunk.Content = *choice.Delta.Content
	}

	if choice.Delta.Reasoning != nil {
		chunk.Reasoning = *choice.Delta.Reasoning
	}

	if len(choice.Delta.ToolCalls) > 0 {
		chunk.ToolCalls = choice.Delta.ToolCalls
	}

	if choice.FinishReason != nil {
		chunk.FinishReason = *choice.FinishReason
		if *choice.FinishReason != FinishReasonNull {
			chunk.IsDone = true
		}
	}

	return chunk
}
