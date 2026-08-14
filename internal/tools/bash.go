package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vesvai/vesvai/internal/agent"
)

const maxOutputLen = 100000

type cappedBuffer struct {
	mu    sync.Mutex
	buf   []byte
	limit int
	full  bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.full && len(b.buf) < b.limit {
		room := b.limit - len(b.buf)
		if len(p) > room {
			b.buf = append(b.buf, p[:room]...)
			b.full = true
		} else {
			b.buf = append(b.buf, p...)
		}
	}

	return len(p), nil
}

func (b *cappedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	s := string(b.buf)
	if b.full {
		s += "\n\n... (output truncated, exceeded 100KB)"
	}
	return s
}

func newBashTool(projectRoot string) agent.Tool {
	return agent.NewFuncTool(
		"bash",
		"Execute a shell command and return its output. Use this to run build commands, tests, git operations, or any system command. Captures both stdout and stderr. Output is truncated at 100KB to prevent context overflow. Supports timeout (default 2 minutes) and custom working directory. Returns exit code on failure. Working directory is confined to the project root; commands that try to cd outside the project will be rejected.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The shell command to execute. Runs in bash. Use && for chaining, | for pipes, > for redirection.",
				},
				"workdir": map[string]any{
					"type":        "string",
					"description": "Working directory for the command. Must be within the project directory. Defaults to project root. Use absolute paths.",
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
				abs, err := expandHome(workdir)
				if err != nil {
					return "", err
				}
				workdir = abs
			} else {
				workdir = projectRoot
			}

			if projectRoot != "" {
				workdir = filepath.Clean(workdir)
				if !isWithinDir(workdir, projectRoot) {
					return "", fmt.Errorf("working directory %s is outside the project root %s", workdir, projectRoot)
				}

				if err := checkCDEscape(command, projectRoot); err != nil {
					return "", err
				}
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

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return "", fmt.Errorf("failed to capture stdout: %w", err)
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				return "", fmt.Errorf("failed to capture stderr: %w", err)
			}

			if err := cmd.Start(); err != nil {
				return "", fmt.Errorf("failed to start command: %w", err)
			}

			out := &cappedBuffer{limit: maxOutputLen}
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				io.Copy(out, stdout)
			}()
			go func() {
				defer wg.Done()
				io.Copy(out, stderr)
			}()

			err = cmd.Wait()
			wg.Wait()

			outputStr := out.String()

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

func isWithinDir(abs, root string) bool {
	abs = filepath.Clean(abs)
	root = filepath.Clean(root)
	if abs == root {
		return true
	}
	return strings.HasPrefix(abs, root+string(filepath.Separator))
}

func checkCDEscape(command, projectRoot string) error {
	tokens := strings.Fields(command)
	for i, tok := range tokens {
		if tok != "cd" || i+1 >= len(tokens) {
			continue
		}
		target := strings.TrimRight(tokens[i+1], ";")
		target = strings.TrimSuffix(target, "&&")
		target = strings.TrimSpace(target)
		if target == "" || target == "." {
			continue
		}
		abs := expandTilde(target)
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(projectRoot, abs)
		}
		abs = filepath.Clean(abs)
		if !isWithinDir(abs, projectRoot) {
			return fmt.Errorf("cd %s would escape the project root %s", target, projectRoot)
		}
	}
	return nil
}

func expandTilde(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
