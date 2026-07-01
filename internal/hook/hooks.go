package hook

import "context"

const (
	HookSystemInit     = "system:init"
	HookSystemReady    = "system:ready"
	HookSystemShutdown = "system:shutdown"

	HookError         = "error"
	HookErrorRecovery = "error:recovery"
)

type HookContext struct {
	Context context.Context
	Session *SessionInfo
	Data    map[string]interface{}
}

type SessionInfo struct {
	ID   string
	Name string
}

func NewHookContext(ctx context.Context) *HookContext {
	return &HookContext{
		Context: ctx,
		Data:    make(map[string]interface{}),
	}
}

func (hc *HookContext) WithSession(id, name string) *HookContext {
	hc.Session = &SessionInfo{ID: id, Name: name}
	return hc
}

func (hc *HookContext) Set(key string, value interface{}) {
	hc.Data[key] = value
}

func (hc *HookContext) Get(key string) (interface{}, bool) {
	val, ok := hc.Data[key]
	return val, ok
}
