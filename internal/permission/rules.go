package permission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vesvai/vesvai/internal/config"
)

type Action string

const (
	ActionAllow  Action = "allow"
	ActionDeny   Action = "deny"
	ActionPrompt Action = "prompt"
)

type Rule struct {
	Tool     string   `json:"tool"`
	Paths    []string `json:"paths,omitempty"`
	Commands []string `json:"commands,omitempty"`
	Action   Action   `json:"action"`
}

type Rules struct {
	Default Action `json:"default"`
	Rules   []Rule `json:"rules"`
}

type Resolution struct {
	Action  Action
	Reason  string
	Matched *Rule
}

const allowByDefault Action = ActionPrompt

func DefaultRules() *Rules {
	return &Rules{
		Default: ActionPrompt,
		Rules: []Rule{
			{Tool: "read", Action: ActionAllow},
			{Tool: "list", Action: ActionAllow},
			{Tool: "glob", Action: ActionAllow},
			{Tool: "grep", Action: ActionAllow},
			{Tool: "get-todo", Action: ActionAllow},
			{Tool: "list-todos", Action: ActionAllow},
			{Tool: "set-todo", Action: ActionAllow},
			{Tool: "update-todo", Action: ActionAllow},
			{Tool: "delete-todo", Action: ActionAllow},
			{Tool: "get-fact", Action: ActionAllow},
			{Tool: "list-facts", Action: ActionAllow},
			{Tool: "search-facts", Action: ActionAllow},
			{Tool: "get-note", Action: ActionAllow},
			{Tool: "get-stats", Action: ActionAllow},
			{Tool: "planner", Action: ActionAllow},
			{Tool: "explorer", Action: ActionAllow},
			{Tool: "orchestrator", Action: ActionAllow},
			{Tool: "message", Action: ActionAllow},
			{Tool: "collect-messages", Action: ActionAllow},
			{
				Tool:     "bash",
				Commands: dangerousBashPrefixes(),
				Action:   ActionDeny,
			},
		},
	}
}

func dangerousBashPrefixes() []string {
	return []string{
		"rm -rf /",
		"rm -rf ~",
		"rm -rf /*",
		"rm -rf $HOME",
		"sudo rm",
		":(){:|:&};:",
		"mkfs",
		"dd if=/dev/zero of=/dev/",
		"chmod -R 000 /",
		"| sh",
		"| bash",
	}
}

func (r *Rules) Check(toolName string, params map[string]any, projectDir string) Resolution {
	if esc, p := pathEscapesProject(params, projectDir); esc {
		if res, has := r.findExplicitAllow(toolName, params, p); has {
			return res
		}
		return Resolution{
			Action: ActionPrompt,
			Reason: fmt.Sprintf("path %s is outside the project directory", p),
		}
	}

	for i := range r.Rules {
		rule := &r.Rules[i]
		if ruleMatches(rule, toolName, params) {
			return Resolution{Action: rule.Action, Reason: ruleReason(rule), Matched: rule}
		}
	}

	return Resolution{Action: r.effectiveDefault(), Reason: "no matching rule (default)", Matched: nil}
}

func (r *Rules) effectiveDefault() Action {
	if r.Default == "" {
		return allowByDefault
	}
	return r.Default
}

func (r *Rules) findExplicitAllow(toolName string, params map[string]any, escapedPath string) (Resolution, bool) {
	for i := range r.Rules {
		rule := &r.Rules[i]
		if rule.Action != ActionAllow || len(rule.Paths) == 0 {
			continue
		}
		if !ruleMatches(rule, toolName, params) {
			continue
		}
		for _, glob := range rule.Paths {
			if pathAllowed(glob, escapedPath) {
				return Resolution{Action: ActionAllow, Reason: "explicitly allowed", Matched: rule}, true
			}
		}
	}
	return Resolution{}, false
}

func ruleMatches(rule *Rule, toolName string, params map[string]any) bool {
	if rule.Tool != "*" && rule.Tool != toolName {
		return false
	}
	if len(rule.Commands) > 0 {
		cmd := extractCommand(params)
		if cmd == "" {
			return false
		}
		if !commandMatchesAny(rule.Commands, cmd) {
			return false
		}
	}
	if len(rule.Paths) > 0 {
		paths := extractPaths(params)
		if len(paths) == 0 {
			return false
		}
		matched := false
		for _, p := range paths {
			for _, glob := range rule.Paths {
				if pathAllowed(glob, p) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func pathAllowed(glob, path string) bool {
	if glob == "" || path == "" {
		return false
	}
	if strings.ContainsAny(glob, "*?[") {
		ok, _ := filepath.Match(glob, path)
		return ok
	}
	glob = filepath.Clean(glob)
	path = filepath.Clean(path)
	if path == glob {
		return true
	}
	if fi, err := os.Stat(glob); err == nil && !fi.IsDir() {
		return false
	}
	return strings.HasPrefix(path, glob+string(filepath.Separator))
}

func ruleReason(rule *Rule) string {
	parts := []string{fmt.Sprintf("tool=%s action=%s", rule.Tool, rule.Action)}
	if len(rule.Paths) > 0 {
		parts = append(parts, "paths="+strings.Join(rule.Paths, ","))
	}
	if len(rule.Commands) > 0 {
		parts = append(parts, "commands="+strings.Join(rule.Commands, ","))
	}
	return strings.Join(parts, " ")
}

func extractCommand(params map[string]any) string {
	if c, ok := params["command"].(string); ok {
		return c
	}
	return ""
}

func extractPaths(params map[string]any) []string {
	var paths []string
	for _, key := range []string{"path", "file_path", "file", "workdir", "dir", "directory", "pattern"} {
		if s, ok := params[key].(string); ok && s != "" {
			paths = append(paths, s)
		}
	}
	return paths
}

func pathEscapesProject(params map[string]any, projectDir string) (bool, string) {
	if projectDir == "" {
		return false, ""
	}
	projectDir = filepath.Clean(projectDir)
	for _, p := range extractPaths(params) {
		if p == "" {
			continue
		}
		abs := resolvePathForCheck(p, projectDir)
		if !isWithinOrEqual(abs, projectDir) {
			return true, abs
		}
	}
	return false, ""
}

func resolvePathForCheck(p, projectDir string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Clean(filepath.Join(home, p[2:]))
		}
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(projectDir, p))
}

func isWithinOrEqual(abs, root string) bool {
	if abs == root {
		return true
	}
	return strings.HasPrefix(abs, root+string(filepath.Separator))
}

func commandMatchesAny(patterns []string, cmd string) bool {
	c := strings.TrimSpace(cmd)
	for _, p := range patterns {
		if commandMatchPattern(p, c) {
			return true
		}
	}
	return false
}

func commandMatchPattern(pattern, cmd string) bool {
	if !strings.Contains(pattern, "*") {
		return strings.Contains(cmd, pattern)
	}
	parts := strings.SplitN(pattern, "*", 2)
	prefix, suffix := parts[0], parts[1]
	if !strings.HasPrefix(cmd, prefix) {
		return false
	}
	if suffix == "" {
		return true
	}
	return strings.HasSuffix(cmd, suffix)
}

type Manager struct {
	mu         sync.RWMutex
	global     *Rules
	project    *Rules
	projectDir string
}

func NewManager(projectDir string) *Manager {
	return &Manager{
		global:     loadRules(globalRulesPath()),
		project:    loadRules(filepath.Join(projectDir, ".vesvai", "permissions.json")),
		projectDir: projectDir,
	}
}

func (m *Manager) Rules() *Rules {
	m.mu.RLock()
	defer m.mu.RUnlock()

	builtin := DefaultRules()
	merged := &Rules{
		Default: builtin.Default,
		Rules:   append([]Rule{}, builtin.Rules...),
	}
	if m.project != nil && len(m.project.Rules) > 0 {
		merged.Rules = append(append([]Rule{}, m.project.Rules...), merged.Rules...)
	}
	if m.global != nil && len(m.global.Rules) > 0 {
		merged.Rules = append(append([]Rule{}, m.global.Rules...), merged.Rules...)
	}
	if m.global != nil && m.global.Default != "" {
		merged.Default = m.global.Default
	}
	if m.project != nil && m.project.Default != "" {
		merged.Default = m.project.Default
	}
	if merged.Default == "" {
		merged.Default = allowByDefault
	}
	return merged
}

func (m *Manager) Check(toolName string, params map[string]any) Resolution {
	return m.Rules().Check(toolName, params, m.projectDir)
}

func (m *Manager) AddRule(rule Rule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.project == nil {
		m.project = &Rules{Default: "", Rules: nil}
	}
	m.project.Rules = append(m.project.Rules, rule)
	return saveRules(m.projectDir, m.project)
}

func (m *Manager) AddAllowRule(toolName string, params map[string]any) error {
	rule := Rule{Tool: toolName, Action: ActionAllow}
	if c := extractCommand(params); c != "" {
		rule.Commands = []string{commandPrefix(c)}
	}
	if paths := extractPaths(params); len(paths) > 0 {
		rule.Paths = append(rule.Paths, paths...)
	}
	return m.AddRule(rule)
}

func commandPrefix(cmd string) string {
	c := strings.TrimSpace(cmd)
	if i := strings.IndexAny(c, "&|;"); i > 0 {
		c = strings.TrimSpace(c[:i])
	}
	fields := strings.Fields(c)
	if len(fields) > 2 {
		fields = fields[:2]
	}
	return strings.Join(fields, " ") + "*"
}

func globalRulesPath() string {
	dir, err := config.GetConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "permissions.json")
}

func loadRules(path string) *Rules {
	if path == "" {
		return &Rules{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Rules{}
		}
		return &Rules{}
	}
	var r Rules
	if err := json.Unmarshal(data, &r); err != nil {
		return &Rules{}
	}
	return &r
}

func saveRules(projectDir string, r *Rules) error {
	dir := filepath.Join(projectDir, ".vesvai")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create .vesvai: %w", err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "permissions.json"), data, 0644)
}

func SaveGlobal(r *Rules) error {
	dir, err := config.GetConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "permissions.json"), data, 0644)
}
