package permission

import (
	"path/filepath"
	"strings"

	json "github.com/goccy/go-json"
)

var commandWhitelist = map[string]bool{
	"go": true, "npm": true, "npx": true, "node": true,
	"yarn": true, "pnpm": true, "git": true, "ls": true,
	"pwd": true, "cat": true, "grep": true, "find": true,
	"mkdir": true, "touch": true, "echo": true, "tail": true,
	"head": true, "tree": true,
	"dir": true, "type": true, "findstr": true,
}

var commandBlacklist = map[string]bool{
	"rm": true, "mv": true, "cp": true, "chmod": true, "chown": true,
	"sudo": true, "su": true, "bash": true, "sh": true, "zsh": true,
	"wget": true, "curl": true, "nc": true, "kill": true, "pkill": true,
	"reboot": true, "halt": true, "poweroff": true, "dd": true, "mkfs": true,
	"docker": true, "kubectl": true,
	"del": true, "erase": true, "rmdir": true, "rd": true,
	"move": true, "ren": true, "rename": true,
	"cmd": true, "powershell": true, "pwsh": true,
	"reg": true, "regedit": true,
	"taskkill": true, "schtasks": true,
	"wmic": true, "certutil": true, "vssadmin": true,
	"format": true, "diskpart": true, "net": true, "netstat": true,
}

var dangerousArgs = []string{
	"-rf", "--force",
	"../", "..\\",
	"/etc", "/var", "/root", "/bin", "/sbin", "/usr",
	".ssh", ".gnupg",
	"c:\\windows", "\\system32", "\\syswow64",
	"hklm\\", "hkcu\\",
}

func bashAllowed(args string) bool {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return false
	}

	cmd := strings.TrimSpace(params.Command)
	if cmd == "" {
		return false
	}

	if hasShellMetachar(cmd) {
		return false
	}

	tokens := strings.Fields(cmd)
	if len(tokens) == 0 {
		return false
	}

	binPath := tokens[0]
	baseCmd := filepath.Base(binPath)
	baseCmd = strings.ToLower(baseCmd)

	baseCmd = strings.TrimSuffix(baseCmd, ".exe")
	baseCmd = strings.TrimSuffix(baseCmd, ".bat")
	baseCmd = strings.TrimSuffix(baseCmd, ".cmd")
	baseCmd = strings.TrimSuffix(baseCmd, ".ps1")
	baseCmd = strings.TrimSuffix(baseCmd, ".vbs")

	if commandBlacklist[baseCmd] {
		return false
	}

	if !commandWhitelist[baseCmd] {
		return false
	}

	if hasDangerousArgs(tokens[1:]) {
		return false
	}

	return true
}

func hasShellMetachar(s string) bool {
	return strings.ContainsAny(s, ";|&><$`\n\r(){}\\")
}

func hasDangerousArgs(args []string) bool {
	for _, arg := range args {
		lowerArg := strings.ToLower(arg)

		if lowerArg == "-rf" || lowerArg == "-f" || lowerArg == "--force" {
			return true
		}

		for _, danger := range dangerousArgs {
			if strings.Contains(lowerArg, danger) {
				return true
			}
		}
	}
	return false
}
