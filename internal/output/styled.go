package output

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Sensitive key patterns for masking.
var sensitivePatterns = []string{"KEY", "SECRET", "TOKEN", "PASSWORD", "CREDENTIAL", "URL"}

var (
	dimColor    = lipgloss.Color("#666666")
	titleColor  = lipgloss.Color("#5B9BD5")
	dimStyle    = lipgloss.NewStyle().Foreground(dimColor)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(titleColor)
	headerStyle = lipgloss.NewStyle().Foreground(dimColor)

	scopeStyles = map[string]struct {
		bullet string
		style  lipgloss.Style
	}{
		"global":   {"●", lipgloss.NewStyle().Foreground(lipgloss.Color("#5B9BD5"))},
		"project":  {"○", lipgloss.NewStyle().Foreground(lipgloss.Color("#6BBF6B"))},
		"personal": {"◇", lipgloss.NewStyle().Foreground(lipgloss.Color("#D4A843"))},
		"plugin":   {"◆", lipgloss.NewStyle().Foreground(lipgloss.Color("#4DCFCF"))},
	}
	defaultScopeStyle = struct {
		bullet string
		style  lipgloss.Style
	}{"·", lipgloss.NewStyle().Foreground(dimColor)}
)

// RenderDivider creates a section divider with title.
func RenderDivider(title string, noColor bool) string {
	if title == "" {
		line := strings.Repeat("─", 40)
		if noColor {
			return fmt.Sprintf("  %s", line)
		}
		return fmt.Sprintf("  %s", dimStyle.Render(line))
	}
	line := strings.Repeat("─", max(2, 40-len(title)-1))
	if noColor {
		return fmt.Sprintf("  %s %s", title, line)
	}
	return fmt.Sprintf("  %s %s", titleStyle.Render(title), dimStyle.Render(line))
}

// RenderTitle renders a top-level title with color.
func RenderTitle(title string, noColor bool) string {
	if noColor {
		return fmt.Sprintf("  %s", title)
	}
	return fmt.Sprintf("  %s", titleStyle.Render(title))
}

// RenderHeader renders a column header row in gray.
func RenderHeader(text string, noColor bool) string {
	if noColor {
		return text
	}
	return headerStyle.Render(text)
}

// RenderDim renders text in dim gray.
func RenderDim(text string, noColor bool) string {
	if noColor {
		return text
	}
	return dimStyle.Render(text)
}

// RenderScopeBullet returns the bullet character for a scope.
func RenderScopeBullet(scope string, noColor bool) string {
	s, ok := scopeStyles[scope]
	if !ok {
		s = defaultScopeStyle
	}
	if noColor {
		return s.bullet
	}
	return s.style.Render(s.bullet)
}

// MaskEnvValue masks sensitive environment variable values.
func MaskEnvValue(key, value string) string {
	upper := strings.ToUpper(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(upper, pattern) {
			return "••••••••"
		}
	}
	return value
}

// FormatBytes formats byte count as human-readable string.
func FormatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// FormatCount formats a large integer with human-readable suffixes (k, M).
func FormatCount(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// ShortModelName abbreviates a Claude model identifier for display.
// "claude-sonnet-4-6-20250514" → "sonnet-4.6"
func ShortModelName(model string) string {
	// Strip "claude-" prefix
	short, ok := strings.CutPrefix(model, "claude-")
	if !ok {
		return model
	}
	// Strip trailing date segment (8+ digits)
	if idx := strings.LastIndex(short, "-"); idx > 0 {
		suffix := short[idx+1:]
		if len(suffix) >= 8 && isAllDigits(suffix) {
			short = short[:idx]
		}
	}
	// Replace final single-digit segment separator with dot: "sonnet-4-6" → "sonnet-4.6"
	if idx := strings.LastIndex(short, "-"); idx > 0 {
		suffix := short[idx+1:]
		if len(suffix) == 1 && suffix[0] >= '0' && suffix[0] <= '9' {
			short = short[:idx] + "." + suffix
		}
	}
	return short
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
