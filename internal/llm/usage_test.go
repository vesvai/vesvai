package llm

import "testing"

func TestUsageContextPercent(t *testing.T) {
	u := &Usage{TotalTokens: 10000}
	if pct := u.ContextPercent(40000); pct != 25 {
		t.Fatalf("pct = %v, want 25", pct)
	}
	if pct := u.ContextPercent(0); pct != 0 {
		t.Fatalf("pct = %v, want 0 for unknown window", pct)
	}
	if pct := (&Usage{}).ContextPercent(40000); pct != 0 {
		t.Fatalf("pct = %v, want 0 for empty usage", pct)
	}
}

func TestModelMaxInputTokens(t *testing.T) {
	cases := []struct {
		name string
		m    *Model
		want int
	}{
		{"max input", &Model{Config: &ModelConfig{MaxInputTokens: 128000}}, 128000},
		{"fallback max tokens", &Model{Config: &ModelConfig{MaxTokens: 8192}}, 8192},
		{"prefer max input", &Model{Config: &ModelConfig{MaxInputTokens: 128000, MaxTokens: 8192}}, 128000},
		{"no config", &Model{}, 0},
		{"nil model", nil, 0},
	}
	for _, tc := range cases {
		if got := tc.m.MaxInputTokens(); got != tc.want {
			t.Fatalf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{500, "500"},
		{254600, "254.6K"},
		{1000000, "1.0M"},
		{12500000, "12.5M"},
	}
	for _, tc := range cases {
		if got := FormatTokens(tc.n); got != tc.want {
			t.Fatalf("FormatTokens(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestFormatContextUsage(t *testing.T) {
	if got := FormatContextUsage(Usage{TotalTokens: 254600}, 1000000); got != "254.6K (25%)" {
		t.Fatalf("got %q", got)
	}
	if got := FormatContextUsage(Usage{TotalTokens: 10000}, 40000); got != "10.0K (25%)" {
		t.Fatalf("got %q", got)
	}
	if got := FormatContextUsage(Usage{TotalTokens: 254600}, 0); got != "254.6K" {
		t.Fatalf("got %q, want tokens only when window unknown", got)
	}
}
