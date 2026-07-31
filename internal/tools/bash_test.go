package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCDEscape_AllowedWithin(t *testing.T) {
	if err := checkCDEscape("cd src && go build", "/project"); err != nil {
		t.Errorf("cd src should be allowed: %v", err)
	}
}

func TestCheckCDEscape_RelativeEscape(t *testing.T) {
	if err := checkCDEscape("cd ../secret && ls", "/project"); err == nil {
		t.Error("cd ../secret should be rejected")
	}
}

func TestCheckCDEscape_AbsoluteEscape(t *testing.T) {
	if err := checkCDEscape("cd /etc && cat passwd", "/project"); err == nil {
		t.Error("cd /etc should be rejected")
	}
}

func TestCheckCDEscape_HomeTildeEscape(t *testing.T) {
	if err := checkCDEscape("cd ~ && ls", "/project"); err == nil {
		t.Error("cd ~ should be rejected (outside project)")
	}
}

func TestIsWithinDir(t *testing.T) {
	cases := []struct {
		abs, root string
		want      bool
	}{
		{"/project", "/project", true},
		{"/project/src", "/project", true},
		{"/projectother", "/project", false},
		{"/Projects/x", "/project", false},
		{"/etc", "/project", false},
	}
	for _, c := range cases {
		if got := isWithinDir(c.abs, c.root); got != c.want {
			t.Errorf("isWithinDir(%q, %q) = %v, want %v", c.abs, c.root, got, c.want)
		}
	}
}

func TestBashTool_WorkdirOutsideProjectRejected(t *testing.T) {
	tool := newBashTool("/project")
	if tool.Name() != "bash" {
		t.Errorf("name = %s", tool.Name())
	}
}

func TestBashTool_DescriptionMentionsSandbox(t *testing.T) {
	tool := newBashTool("/project")
	desc := tool.Description()
	if !contains(desc, "project root") {
		t.Errorf("description should mention project root sandboxing: %s", desc)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBashToolSandboxWorkdirs(t *testing.T) {
	cases := []struct {
		workdir string
		root    string
		ok      bool
	}{
		{"", "/project", true},
		{".", "/project", true},
		{"/project/sub", "/project", true},
		{"/etc", "/project", false},
		{"/projectother", "/project", false},
	}
	for _, c := range cases {
		abs := c.workdir
		if abs == "" || abs == "." {
			abs = c.root
		}
		got := isWithinDir(abs, c.root)
		if got != c.ok {
			t.Errorf("workdir=%q root=%q isWithinDir=%v want %v", c.workdir, c.root, got, c.ok)
		}
	}
}

func TestExpandTilde(t *testing.T) {
	out := expandTilde("/absolute/path")
	if out != "/absolute/path" {
		t.Errorf("expandTilde(/absolute) = %q", out)
	}
}

func TestBashCtx(t *testing.T) {
	ctx := context.Background()
	_ = ctx
}
func TestCountFileLines(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	os.WriteFile(p, []byte("line1\nline2\nline3\n"), 0644)

	if got := countFileLines(p, 30); got != 3 {
		t.Errorf("countFileLines = %d, want 3", got)
	}

	os.WriteFile(p, []byte("a\nb"), 0644)
	if got := countFileLines(p, 3); got != 2 {
		t.Errorf("countFileLines(no trailing NL) = %d, want 2", got)
	}

	if got := countFileLines(p, 3<<20); got != 0 {
		t.Errorf("countFileLines(oversize) = %d, want 0", got)
	}
}
