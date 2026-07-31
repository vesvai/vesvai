package llm

import "testing"

func TestRoleValues(t *testing.T) {
	tests := []struct {
		role     Role
		expected string
	}{
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
		{RoleSystem, "system"},
		{RoleTool, "tool"},
	}
	for _, tt := range tests {
		if string(tt.role) != tt.expected {
			t.Errorf("Role %v = %q, want %q", tt.role, string(tt.role), tt.expected)
		}
	}
}

func TestFinishReasonValues(t *testing.T) {
	tests := []struct {
		reason   FinishReason
		expected string
	}{
		{FinishReasonStop, "stop"},
		{FinishReasonLength, "length"},
		{FinishReasonContentFilter, "content_filter"},
		{FinishReasonToolCalls, "tool_calls"},
		{FinishReasonNull, "null"},
	}
	for _, tt := range tests {
		if string(tt.reason) != tt.expected {
			t.Errorf("FinishReason %v = %q, want %q", tt.reason, string(tt.reason), tt.expected)
		}
	}
}
