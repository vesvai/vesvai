package sdk

func (e *Engine) Subscribe(topic string, handler any) (func(), error) {
	if err := e.checkOpen(); err != nil {
		return nil, err
	}
	if err := e.bus.Subscribe(topic, handler); err != nil {
		return nil, err
	}
	return func() {
		_ = e.bus.Unsubscribe(topic, handler)
	}, nil
}

func (e *Engine) SubscribeOnce(topic string, handler any) (func(), error) {
	if err := e.checkOpen(); err != nil {
		return nil, err
	}
	if err := e.bus.SubscribeOnce(topic, handler); err != nil {
		return nil, err
	}
	return func() {
		_ = e.bus.Unsubscribe(topic, handler)
	}, nil
}

func (e *Engine) OnAgentToken(fn func(AgentToken)) (func(), error) {
	return e.Subscribe(TopicAgentToken, fn)
}

func (e *Engine) OnAgentToolCall(fn func(AgentToolCall)) (func(), error) {
	return e.Subscribe(TopicAgentToolCall, fn)
}

func (e *Engine) OnAgentToolResult(fn func(AgentToolResult)) (func(), error) {
	return e.Subscribe(TopicAgentToolResult, fn)
}

func (e *Engine) OnAgentFinished(fn func(AgentFinished)) (func(), error) {
	return e.Subscribe(TopicAgentFinished, fn)
}

func (e *Engine) OnAgentError(fn func(AgentError)) (func(), error) {
	return e.Subscribe(TopicAgentError, fn)
}

func (e *Engine) OnCompaction(fn func(CompactionEvent)) (func(), error) {
	return e.Subscribe(TopicCompactionFinished, fn)
}
