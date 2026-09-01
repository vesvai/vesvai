package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func bashTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"bash",
		"Execute a shell command on the local machine. The command runs with a 40-second timeout by default; pass 'timeout' (in seconds) to override. Use this tool to run build scripts, tests, linters, git operations, or any other command-line task. The working directory defaults to the workspace root; use 'workdir' to run in a subdirectory (relative to workspace root). Both stdout and stderr are captured and returned. Exit code is included in the output. Prefer this tool over the file tools when you need to run CLI programs rather than manipulate files directly.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Shell command to execute. Use standard shell syntax. Examples: 'go test ./...', 'npm run build', 'git status', 'ls -la', 'cargo check'.",
				},
				"workdir": map[string]any{
					"type":        "string",
					"description": "Working directory relative to workspace root. Omit or set to empty string to use the workspace root. Example: 'src/myapp', 'internal/core'.",
				},
				"timeout": map[string]any{
					"type":        "integer",
					"description": "Timeout in seconds. Defaults to 40. Use a larger value for long-running commands (e.g. builds, test suites), a smaller one for quick checks.",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Optional short explanation of what the command does and why. It's usefull for explaining to user what commands do.",
				},
			},
			"required": []string{"command"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				Command     string `json:"command"`
				Workdir     string `json:"workdir"`
				Timeout     int    `json:"timeout"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("bash: invalid arguments: %w", err)
			}
			if params.Command == "" {
				return "", fmt.Errorf("bash: command is required")
			}
			timeout := 40 * time.Second
			if params.Timeout > 0 {
				timeout = time.Duration(params.Timeout) * time.Second
			}

			workdir := fs.Root()
			if params.Workdir != "" {
				resolved, err := fs.Resolve(params.Workdir)
				if err != nil {
					return "", fmt.Errorf("bash: resolve workdir: %w", err)
				}
				workdir = resolved
			}

			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)
			cmd.Dir = workdir

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			exitCode := 0
			if err != nil {
				if ctx.Err() != nil {
					return fmt.Sprintf("Command timed out after %s.\nstdout:\n%s\nstderr:\n%s\n", timeout, stdout.String(), stderr.String()), nil
				}
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					return "", fmt.Errorf("bash: %w", err)
				}
			}

			result := fmt.Sprintf("Exit code: %d\n", exitCode)
			if stdout.Len() > 0 {
				result += fmt.Sprintf("stdout:\n%s\n", stdout.String())
			}
			if stderr.Len() > 0 {
				result += fmt.Sprintf("stderr:\n%s\n", stderr.String())
			}
			return result, nil
		},
	)
}
