package output_test

import (
	"strings"
	"testing"

	"github.com/jackchuka/ccli/internal/output"
)

func TestRenderDivider(t *testing.T) {
	s := output.RenderDivider("MCP Servers", true)
	if !strings.Contains(s, "MCP Servers") {
		t.Errorf("divider missing title, got: %q", s)
	}
	if !strings.Contains(s, "───") {
		t.Errorf("divider missing line chars, got: %q", s)
	}
}

func TestRenderScopeBullet(t *testing.T) {
	// With color disabled, just check the bullet character
	s := output.RenderScopeBullet("global", true)
	if !strings.Contains(s, "●") {
		t.Errorf("global bullet missing ●, got: %q", s)
	}

	s = output.RenderScopeBullet("project", true)
	if !strings.Contains(s, "○") {
		t.Errorf("project bullet missing ○, got: %q", s)
	}
}

func TestMaskEnvValue(t *testing.T) {
	tests := []struct {
		key  string
		val  string
		want string
	}{
		{"DATADOG_API_KEY", "secret123", "••••••••"},
		{"DATADOG_SITE", "us5.datadoghq.com", "us5.datadoghq.com"},
		{"SECRET_TOKEN", "abc", "••••••••"},
		{"DATABASE_URL", "postgres://...", "••••••••"},
	}
	for _, tt := range tests {
		got := output.MaskEnvValue(tt.key, tt.val)
		if got != tt.want {
			t.Errorf("MaskEnvValue(%q, %q) = %q, want %q", tt.key, tt.val, got, tt.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1395864371, "1.3 GB"},
	}
	for _, tt := range tests {
		got := output.FormatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
