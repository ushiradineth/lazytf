package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestHistoryViewRendersContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(60, 10)
	view.SetTitle("Apply details")
	view.SetContent("line one\nline two")

	out := view.View()
	if !strings.Contains(out, "Apply details") {
		t.Fatalf("expected title in output")
	}
	if !strings.Contains(out, "line one") || !strings.Contains(out, "line two") {
		t.Fatalf("expected content in output")
	}
}

func TestHistoryViewUpdateHandlesKeys(_ *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(40, 6)
	view.SetTitle("Title")
	view.SetContent("line")

	_, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
}

func TestHistoryViewSetStyles(t *testing.T) {
	view := NewHistoryView(styles.DefaultStyles())
	view.SetSize(60, 20)

	newStyles := styles.DefaultStyles()
	view.SetStyles(newStyles)

	if view.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestHistoryViewViewContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 20)
	view.SetContent("line one\nline two\nline three")

	content := view.ViewContent()
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestHistoryViewViewContentNilStyles(t *testing.T) {
	view := &HistoryView{}
	content := view.ViewContent()
	if content != "" {
		t.Error("expected empty content for nil styles")
	}
}

func TestHistoryViewViewContentZeroWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(0, 20)
	view.SetContent("test content")

	content := view.ViewContent()
	// Should still return something even with zero width
	_ = content
}

func TestHistoryViewGetTitle(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)

	// Initially empty
	if view.GetTitle() != "" {
		t.Error("expected empty title initially")
	}

	view.SetTitle("Apply Output")
	if view.GetTitle() != "Apply Output" {
		t.Errorf("expected 'Apply Output', got %q", view.GetTitle())
	}
}

func TestHistoryViewViewWithStatusContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)
	view.SetTitle("Operation Details")

	// Content with status metadata that exercises colorizeMetadataLine
	content := `Status:       Success
Started:      2024-01-15
Duration:     5m30s
Command:      terraform apply

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.`

	view.SetContent(content)
	out := view.View()

	if !strings.Contains(out, "Operation Details") {
		t.Error("expected title in output")
	}
}

func TestHistoryViewViewWithFailedStatus(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)
	view.SetTitle("Failed Operation")

	// Content with failed status
	content := `Status:       Failed
Started:      2024-01-15
Duration:     2m15s
Command:      terraform apply

Error: Error creating resource`

	view.SetContent(content)
	out := view.View()

	if !strings.Contains(out, "Failed Operation") {
		t.Error("expected title in output")
	}
}

func TestHistoryViewViewWithRunningStatus(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)
	view.SetTitle("Running Operation")

	// Content with running status
	content := `Status:       Running
Started:      2024-01-15
Duration:     0s
Command:      terraform plan

Planning...`

	view.SetContent(content)
	out := view.View()

	if !strings.Contains(out, "Running Operation") {
		t.Error("expected title in output")
	}
}

func TestColorizeLineDiffPrefixes(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)

	tests := []struct {
		name  string
		input string
	}{
		{"replace prefix", "-/+ resource.test"},
		{"add prefix", "+ added_field = value"},
		{"remove prefix", "- removed_field = value"},
		{"change prefix", "~ changed_field = value"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := view.colorizeLine(tc.input)
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestColorizeLineRemoveWithNullSuffix(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)

	// Line with "-> null" suffix should be colored specially
	result := view.colorizeLine("- old_field = value -> null")
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestColorizeLineWithLeadingWhitespace(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)

	// Line with indentation
	result := view.colorizeLine("    + indented_field = value")
	if result == "" {
		t.Error("expected non-empty result")
	}
	// Should preserve indentation
	if !strings.HasPrefix(result, "    ") {
		t.Error("expected leading whitespace to be preserved")
	}
}

func TestColorizeLineSeparatorLine(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)

	// Separator line should return empty string
	result := view.colorizeLine("─────────────────────")
	if result != "" {
		t.Errorf("expected empty string for separator, got %q", result)
	}
}

func TestColorizeLineSectionTitles(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)

	titles := []string{"Details", "Plan Output", "Apply Output"}
	for _, title := range titles {
		t.Run(title, func(t *testing.T) {
			result := view.colorizeLine(title)
			if result == "" {
				t.Errorf("expected non-empty result for section title %q", title)
			}
		})
	}
}

func TestColorizeLineMetadataLines(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)

	lines := []string{
		"Status: Success",
		"Time: 2024-01-15 10:00:00",
		"Environment: production",
		"Directory: /path/to/project",
	}

	for _, line := range lines {
		t.Run(line, func(t *testing.T) {
			result := view.colorizeLine(line)
			if result == "" {
				t.Errorf("expected non-empty result for metadata line %q", line)
			}
		})
	}
}

func TestColorizeLineDefaultCase(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)

	// A regular line without any special prefix
	input := "regular line without special prefix"
	result := view.colorizeLine(input)
	if result != input {
		t.Errorf("expected line to be unchanged, got %q", result)
	}
}

func TestIsSeparatorLine(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"───────", true},
		{"─", true},
		{"normal text", false},
		{"─text", false},
		{"  ─  ", false}, // has spaces
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := isSeparatorLine(tc.input)
			if result != tc.expected {
				t.Errorf("isSeparatorLine(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestIsSectionTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Details", true},
		{"Plan Output", true},
		{"Apply Output", true},
		{"Other Title", false},
		{"", false},
		{"details", false}, // case sensitive
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := isSectionTitle(tc.input)
			if result != tc.expected {
				t.Errorf("isSectionTitle(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestIsMetadataLine(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Status: Success", true},
		{"Time: 10:00:00", true},
		{"Environment: prod", true},
		{"Directory: /path", true},
		{"Other: value", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := isMetadataLine(tc.input)
			if result != tc.expected {
				t.Errorf("isMetadataLine(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestColorizeStatusValue(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)

	tests := []struct {
		name  string
		input string
	}{
		{"success with icon", "  ● Success"},
		{"success text", "  Success"},
		{"failed with icon", "  ✗ Failed"},
		{"failed text", "  Failed"},
		{"canceled with icon", "  ○ Canceled"},
		{"canceled text", "  Canceled"},
		{"other value", "  Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := view.colorizeStatusValue(tc.input)
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestColorizeMetadataLineNoColon(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)

	// Line without colon should return unchanged
	input := "no colon here"
	result := view.colorizeMetadataLine(input, input)
	if result != input {
		t.Errorf("expected line unchanged, got %q", result)
	}
}

func TestColorizeOutputNilStyles(t *testing.T) {
	view := &HistoryView{}
	input := "test content"
	result := view.colorizeOutput(input)
	if result != input {
		t.Errorf("expected content unchanged with nil styles, got %q", result)
	}
}

func TestHistoryViewViewDefaultTitle(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 20)
	view.SetContent("content")

	// Don't set title - should use default
	out := view.View()
	if !strings.Contains(out, "Apply details") {
		t.Error("expected default title 'Apply details'")
	}
}

func TestHistoryViewViewNilStyles(t *testing.T) {
	view := &HistoryView{}
	out := view.View()
	if out != "" {
		t.Error("expected empty output for nil styles")
	}
}

func TestHistoryViewViewZeroWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(0, 20)
	view.SetContent("content")

	// Should still render even with zero width
	out := view.View()
	if out == "" {
		t.Error("expected non-empty output")
	}
}
