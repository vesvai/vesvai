package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/vesvai/vesvai/internal/agent"
)

func newBashTool() agent.Tool {
	return agent.NewFuncTool(
		"bash",
		"Execute a shell command and return its output. Use this to run build commands, tests, git operations, or any system command. Captures both stdout and stderr. Output is truncated at 100KB to prevent context overflow. Supports timeout (default 2 minutes) and custom working directory. Returns exit code on failure.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The shell command to execute. Runs in bash. Use && for chaining, | for pipes, > for redirection.",
				},
				"workdir": map[string]any{
					"type":        "string",
					"description": "Working directory for the command. Defaults to current directory. Use absolute paths.",
				},
				"timeout": map[string]any{
					"type":        "integer",
					"description": "Timeout in milliseconds. Defaults to 120000 (2 minutes). Increase for long-running builds or tests.",
				},
			},
			"required": []string{"command"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			command := asString(params, "command")
			if command == "" {
				return "", fmt.Errorf("command is required")
			}

			workdir := asString(params, "workdir")
			if workdir != "" {
				abs, err := resolvePath(workdir)
				if err != nil {
					return "", err
				}
				workdir = abs
			} else {
				workdir, _ = os.Getwd()
			}

			timeoutMs := asInt(params, "timeout")
			if timeoutMs <= 0 {
				timeoutMs = 120000
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx, "bash", "-c", command)
			cmd.Dir = workdir

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			const maxOutputLen = 100000
			if len(outputStr) > maxOutputLen {
				outputStr = outputStr[:maxOutputLen] + "\n\n... (output truncated, exceeded 100KB)"
			}

			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return outputStr, fmt.Errorf("command timed out after %v", timeout)
				}
				exitCode := -1
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
				return outputStr, fmt.Errorf("command failed with exit code %d: %w", exitCode, err)
			}

			if outputStr == "" {
				outputStr = "(no output)\n"
			}

			return outputStr, nil
		},
	)
}
