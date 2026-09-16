package logging

import (
	"testing"
)

func TestLogCommandStartEndNilLogger(t *testing.T) {
	t.Parallel()
	// Should not panic
	LogCommandStart(nil, "exec-1", "doc", "list", "https://mcp.example.com", "1.0.0", false, 0)
	LogCommandEnd(nil, "exec-1", "doc", "list", true, 0, "", "")
}

func TestRedactEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"https://mcp.dingtalk.com/api", "https://mcp.dingtalk.com/api"},
		{"https://mcp.dingtalk.com/api?token=abc", "https://mcp.dingtalk.com/api"},
		{"https://mcp.dingtalk.com/api#section", "https://mcp.dingtalk.com/api"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := redactEndpoint(tt.input); got != tt.want {
				t.Errorf("redactEndpoint(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
