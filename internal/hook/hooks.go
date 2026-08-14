package hook

const (
	HookSystemInit     = "system:init"
	HookSystemReady    = "system:ready"
	HookSystemShutdown = "system:shutdown"

	HookError         = "error"
	HookErrorRecovery = "error:recovery"

	HookToolsCollect     = "tools:collect"
	HookAgentsCollect    = "agents:collect"
	HookProvidersCollect = "providers:collect"
	HookSkillsCollect    = "skills:collect"
)
