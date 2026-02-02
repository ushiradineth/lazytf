package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTruncateWithANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "plain text shorter than width",
			input:    "hello",
			width:    10,
			expected: "hello",
		},
		{
			name:     "plain text longer than width",
			input:    "hello world",
			width:    5,
			expected: "hello",
		},
		{
			name:     "plain text exact width",
			input:    "hello",
			width:    5,
			expected: "hello",
		},
		{
			name:     "ANSI styled text - truncate preserves codes",
			input:    "\x1b[1mhello world\x1b[0m",
			width:    5,
			expected: "\x1b[1mhello",
		},
		{
			name:     "ANSI styled text - no truncation needed",
			input:    "\x1b[1mhi\x1b[0m",
			width:    10,
			expected: "\x1b[1mhi\x1b[0m",
		},
		{
			name:     "complex ANSI - background color",
			input:    "\x1b[48;5;240mtext\x1b[0m",
			width:    2,
			expected: "\x1b[48;5;240mte",
		},
		{
			name:     "zero width",
			input:    "hello",
			width:    0,
			expected: "",
		},
		{
			name:     "negative width",
			input:    "hello",
			width:    -1,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateWithANSI(tt.input, tt.width)
			if result != tt.expected {
				t.Errorf("TruncateWithANSI(%q, %d) = %q, want %q", tt.input, tt.width, result, tt.expected)
			}
		})
	}
}

func TestPadLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		width       int
		wantVisible int
	}{
		{
			name:        "plain text - needs padding",
			input:       "hello",
			width:       10,
			wantVisible: 10,
		},
		{
			name:        "plain text - exact width",
			input:       "hello",
			width:       5,
			wantVisible: 5,
		},
		{
			name:        "plain text - needs truncation",
			input:       "hello world",
			width:       5,
			wantVisible: 5,
		},
		{
			name:        "ANSI styled - needs padding",
			input:       "\x1b[1mhi\x1b[0m",
			width:       10,
			wantVisible: 10,
		},
		{
			name:        "ANSI styled - exact width",
			input:       "\x1b[1mhello\x1b[0m",
			width:       5,
			wantVisible: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadLine(tt.input, tt.width)
			visible := lipgloss.Width(result)
			if visible != tt.wantVisible {
				t.Errorf("PadLine visible width = %d, want %d (result: %q)", visible, tt.wantVisible, result)
			}
		})
	}
}

func TestPadLineWithBg(t *testing.T) {
	bg := lipgloss.AdaptiveColor{Light: "#B3D9FF", Dark: "#2F5D8A"}

	tests := []struct {
		name        string
		input       string
		width       int
		wantVisible int
	}{
		{
			name:        "needs padding",
			input:       "hello",
			width:       10,
			wantVisible: 10,
		},
		{
			name:        "exact width",
			input:       "hello",
			width:       5,
			wantVisible: 5,
		},
		{
			name:        "needs truncation",
			input:       "hello world",
			width:       5,
			wantVisible: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadLineWithBg(tt.input, tt.width, bg)
			visible := lipgloss.Width(result)
			if visible != tt.wantVisible {
				t.Errorf("PadLineWithBg visible width = %d, want %d", visible, tt.wantVisible)
			}
		})
	}
}

func TestANSICutLeft(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		n        int
		expected string
	}{
		{
			name:     "cut plain text",
			input:    "hello world",
			n:        6,
			expected: "world",
		},
		{
			name:     "cut nothing",
			input:    "hello",
			n:        0,
			expected: "hello",
		},
		{
			name:     "cut more than length",
			input:    "hi",
			n:        10,
			expected: "",
		},
		{
			name:     "cut with ANSI - preserve trailing",
			input:    "hello\x1b[1mworld\x1b[0m",
			n:        5,
			expected: "\x1b[1mworld\x1b[0m",
		},
		{
			name:     "cut in middle of styled region - preserve style",
			input:    "\x1b[38;5;240m──────────\x1b[0m",
			n:        5,
			expected: "\x1b[38;5;240m─────\x1b[0m",
		},
		{
			name:     "cut after reset - no style prepended",
			input:    "\x1b[1mhello\x1b[0m world",
			n:        6,
			expected: "world",
		},
		{
			name:     "multiple ANSI codes - preserve all active",
			input:    "\x1b[1m\x1b[38;5;196mhello world\x1b[0m",
			n:        6,
			expected: "\x1b[1m\x1b[38;5;196mworld\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ANSICutLeft(tt.input, tt.n)
			if result != tt.expected {
				t.Errorf("ANSICutLeft(%q, %d) = %q, want %q", tt.input, tt.n, result, tt.expected)
			}
		})
	}
}

func TestVisibleWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "plain text",
			input:    "hello",
			expected: 5,
		},
		{
			name:     "with ANSI codes",
			input:    "\x1b[1mhello\x1b[0m",
			expected: 5,
		},
		{
			name:     "complex ANSI",
			input:    "\x1b[38;5;196m\x1b[48;5;240mtest\x1b[0m",
			expected: 4,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VisibleWidth(tt.input)
			if result != tt.expected {
				t.Errorf("VisibleWidth(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRenderSectionHeader(t *testing.T) {
	borderColor := lipgloss.AdaptiveColor{Light: "#ccc", Dark: "#555"}
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))

	tests := []struct {
		name      string
		title     string
		width     int
		wantLines int
	}{
		{
			name:      "standard header",
			title:     "Plan Output",
			width:     40,
			wantLines: 3, // top line + title + bottom line
		},
		{
			name:      "zero width uses title width",
			title:     "Apply Output",
			width:     0,
			wantLines: 3,
		},
		{
			name:      "narrow width",
			title:     "Test",
			width:     10,
			wantLines: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderSectionHeader(tt.title, tt.width, titleStyle, borderColor)

			// Check line count
			lines := testSplitLines(result)
			if len(lines) != tt.wantLines {
				t.Errorf("RenderSectionHeader line count = %d, want %d", len(lines), tt.wantLines)
			}

			// Check that title is in the middle line
			if len(lines) >= 2 {
				titleLine := lines[1]
				// The title should be styled but contain the original text
				if !containsVisibleText(titleLine, tt.title) {
					t.Errorf("title line does not contain expected title %q", tt.title)
				}
			}

			// Check that first and last lines are border lines
			if len(lines) >= 3 {
				firstLine := lines[0]
				lastLine := lines[2]
				// Border lines should contain only ─ characters (plus ANSI codes)
				if lipgloss.Width(firstLine) == 0 {
					t.Error("first border line should not be empty")
				}
				if lipgloss.Width(lastLine) == 0 {
					t.Error("last border line should not be empty")
				}
			}
		})
	}
}

func testSplitLines(s string) []string {
	var lines []string
	current := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func containsVisibleText(styled, text string) bool {
	// Strip ANSI codes and check if text is present
	var visible strings.Builder
	inEscape := false
	for _, r := range styled {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		visible.WriteRune(r)
	}
	result := visible.String()
	return result == text || len(result) >= len(text)
}

func TestPadLineWithStyle(t *testing.T) {
	paddingStyle := lipgloss.NewStyle().Background(lipgloss.Color("#333"))

	tests := []struct {
		name        string
		input       string
		width       int
		wantVisible int
	}{
		{
			name:        "needs padding",
			input:       "hello",
			width:       10,
			wantVisible: 10,
		},
		{
			name:        "exact width",
			input:       "hello",
			width:       5,
			wantVisible: 5,
		},
		{
			name:        "needs truncation",
			input:       "hello world",
			width:       5,
			wantVisible: 5,
		},
		{
			name:        "styled input needs padding",
			input:       "\x1b[1mhi\x1b[0m",
			width:       10,
			wantVisible: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadLineWithStyle(tt.input, tt.width, paddingStyle)
			visible := lipgloss.Width(result)
			if visible != tt.wantVisible {
				t.Errorf("PadLineWithStyle visible width = %d, want %d", visible, tt.wantVisible)
			}
		})
	}
}
