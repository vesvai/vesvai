package permission

import (
	"testing"
)

func TestBashAllowed(t *testing.T) {
	cases := []struct {
		name string
		args string
		want bool
	}{
		{"ls", `{"command": "ls -la"}`, true},
		{"go test", `{"command": "go test ./..."}`, true},
		{"npm run build", `{"command": "npm run build"}`, true},
		{"pwd", `{"command": "pwd"}`, true},
		{"node script", `{"command": "node index.js"}`, true},
		{"git status", `{"command": "git status"}`, true},
		{"cat", `{"command": "cat /etc/passwd"}`, false},
		{"rm", `{"command": "rm -rf ."}`, false},
		{"rm with whitespace", `{"command": "  rm -rf .  "}`, false},
		{"semicolon", `{"command": "ls; rm -rf ."}`, false},
		{"pipe", `{"command": "ls | grep x"}`, false},
		{"redirection", `{"command": "ls > /tmp/out"}`, false},
		{"dollar expansion", `{"command": "ls $HOME"}`, false},
		{"command substitution", "{\"command\": \"ls `pwd`\"}", false},
		{"andand", `{"command": "ls && rm -rf ."}`, false},
		{"newline", "{\"command\": \"ls\\nrm -rf .\"}", false},
		{"absolute path", `{"command": "/bin/ls"}`, false},
		{"leading dir", `{"command": "./ls"}`, false},
		{"empty", `{"command": ""}`, false},
		{"invalid json", `not json`, false},
	}
	for _, tc := range cases {
		if got := bashAllowed(tc.args); got != tc.want {
			t.Errorf("%s: bashAllowed = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestHasShellMetachar(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"ls -la", false},
		{"a;b", true},
		{"a|b", true},
		{"a&b", true},
		{"a>b", true},
		{"a<b", true},
		{"a$b", true},
		{"a`b", true},
		{"a\nb", true},
		{"a\rb", true},
		{"a&&b", true},
		{"a||b", true},
	}
	for _, tc := range cases {
		if got := hasShellMetachar(tc.in); got != tc.want {
			t.Errorf("hasShellMetachar(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
