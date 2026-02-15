package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestNewCommandLogPanel(t *testing.T) {
	t.Run("with styles", func(t *testing.T) {
		s := styles.DefaultStyles()
		panel := NewCommandLogPanel(s)
		if panel == nil {
			t.Fatal("expected non-nil panel")
		}
		if panel.styles != s {
			t.Error("styles not set correctly")
		}
		if !panel.visible {
			t.Error("expected panel to be visible by default")
		}
	})

	t.Run("nil styles uses default", func(t *testing.T) {
		panel := NewCommandLogPanel(nil)
		if panel == nil {
			t.Fatal("expected non-nil panel")
		}
		if panel.styles == nil {
			t.Error("expected default styles to be set")
		}
	})
}

func TestCommandLogPanelSetSize(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	if panel.height != 20 {
		t.Errorf("expected height 20, got %d", panel.height)
	}
}

func TestCommandLogPanelFocus(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	if panel.IsFocused() {
		t.Error("expected not focused by default")
	}

	panel.SetFocused(true)
	if !panel.IsFocused() {
		t.Error("expected focused after SetFocused(true)")
	}

	panel.SetFocused(false)
	if panel.IsFocused() {
		t.Error("expected not focused after SetFocused(false)")
	}
}

func TestCommandLogPanelVisibility(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	if !panel.IsVisible() {
		t.Error("expected visible by default")
	}

	panel.SetVisible(false)
	if panel.IsVisible() {
		t.Error("expected not visible after SetVisible(false)")
	}

	panel.SetVisible(true)
	if !panel.IsVisible() {
		t.Error("expected visible after SetVisible(true)")
	}
}

func TestCommandLogPanelSetDiagnostics(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	diagnostics := []terraform.Diagnostic{
		{Severity: "error", Summary: "Test error"},
		{Severity: "warning", Summary: "Test warning"},
	}

	panel.SetDiagnostics(diagnostics)
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelSetLogText(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	panel.SetLogText("test log output\nline 2")
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelUpdate(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	result, cmd := panel.Update(tea.KeyMsg{Type: tea.KeyDown})
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd // cmd may be nil
}

func TestCommandLogPanelHandleKey(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	t.Run("not focused", func(t *testing.T) {
		panel.SetFocused(false)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
		if handled {
			t.Error("expected not handled when not focused")
		}
	})

	t.Run("focused - down", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - up", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - down arrow", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - up arrow", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - pgup", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyPgUp})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - pgdown", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyPgDown})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - home", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyHome})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - end", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEnd})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - unhandled key", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		if handled {
			t.Error("expected not handled for unbound key")
		}
	})
}

func TestCommandLogPanelView(t *testing.T) {
	t.Run("nil styles", func(t *testing.T) {
		panel := &CommandLogPanel{}
		view := panel.View()
		if view != "" {
			t.Error("expected empty view for nil styles")
		}
	})

	t.Run("zero height", func(t *testing.T) {
		panel := NewCommandLogPanel(styles.DefaultStyles())
		panel.SetSize(80, 0)
		view := panel.View()
		if view != "" {
			t.Error("expected empty view for zero height")
		}
	})

	t.Run("not visible", func(t *testing.T) {
		panel := NewCommandLogPanel(styles.DefaultStyles())
		panel.SetSize(80, 20)
		panel.SetVisible(false)
		view := panel.View()
		if view != "" {
			t.Error("expected empty view when not visible")
		}
	})

	t.Run("visible with content", func(t *testing.T) {
		panel := NewCommandLogPanel(styles.DefaultStyles())
		panel.SetSize(80, 20)
		panel.SetLogText("test log content")
		view := panel.View()
		if view == "" {
			t.Error("expected non-empty view")
		}
	})

	t.Run("empty content shows placeholder", func(t *testing.T) {
		panel := NewCommandLogPanel(styles.DefaultStyles())
		panel.SetSize(80, 20)
		view := panel.View()
		if view == "" {
			t.Error("expected non-empty view even without content")
		}
	})
}

func TestCommandLogPanelCalculateThumbSize(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	tests := []struct {
		name          string
		visibleHeight int
		totalLines    int
		expected      float64
	}{
		{"content fits", 20, 10, 1.0},
		{"content equals visible", 20, 20, 1.0},
		{"content exceeds visible", 20, 40, 0.5},
		{"zero visible height", 0, 10, 1.0},
		{"negative visible height", -1, 10, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := panel.calculateThumbSize(tt.visibleHeight, tt.totalLines)
			if result != tt.expected {
				t.Errorf("calculateThumbSize(%d, %d) = %f; want %f",
					tt.visibleHeight, tt.totalLines, result, tt.expected)
			}
		})
	}
}

func TestCommandLogPanelPadLine(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	tests := []struct {
		name     string
		line     string
		width    int
		expected string
	}{
		{"short line", "hello", 10, "hello     "},
		{"exact width", "hello", 5, "hello"},
		{"truncate long", "hello world", 5, "hello"},
		{"empty line", "", 5, "     "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := panel.padLine(tt.line, tt.width)
			if result != tt.expected {
				t.Errorf("padLine(%q, %d) = %q; want %q", tt.line, tt.width, result, tt.expected)
			}
		})
	}
}

func TestCommandLogPanelGetDiagnosticsPanel(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	diag := panel.GetDiagnosticsPanel()
	if diag == nil {
		t.Error("expected non-nil diagnostics panel")
	}
}

func TestCommandLogPanelAppendSessionLog(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	panel.AppendSessionLog("Planned", "terraform plan", "Plan output here")
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelClearSessionLogs(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	panel.AppendSessionLog("Action", "command", "output")
	panel.ClearSessionLogs()
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelSetStyles(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	newStyles := styles.DefaultStyles()
	panel.SetStyles(newStyles)

	if panel.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestCommandLogPanelExpandFillsContent(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	// Add enough session logs to require scrolling
	for i := 1; i <= 20; i++ {
		panel.AppendSessionLog(
			"Operation "+string(rune('A'+i%26)),
			"terraform command "+string(rune('0'+i%10)),
			"Output line 1\nOutput line 2\nOutput line 3",
		)
	}

	// Start with a small size (like compact command log - 10 lines)
	panel.SetSize(80, 10)
	smallView := panel.View()

	// Expand to a larger size (like focused command log - full height)
	panel.SetSize(80, 40)
	expandedView := panel.View()

	// Count lines with actual content (not just whitespace or border chars)
	countContentLines := func(view string) int {
		lines := cmdLogTestSplitLines(view)
		count := 0
		for _, line := range lines {
			trimmed := cmdLogTestTrimAnsi(line)
			// Skip empty lines and lines that are just borders
			if trimmed != "" && !cmdLogTestIsBorderLine(trimmed) {
				count++
			}
		}
		return count
	}

	smallContentLines := countContentLines(smallView)
	expandedContentLines := countContentLines(expandedView)

	// The expanded view should show significantly more content
	if expandedContentLines <= smallContentLines {
		t.Errorf("Expected expanded panel to show more content than small panel.\n"+
			"Small panel content lines: %d\n"+
			"Expanded panel content lines: %d",
			smallContentLines, expandedContentLines)
	}

	// The expanded view should have content filling most of the height
	minExpectedContentLines := 25 // At least 25 lines of content in a 40-line panel
	if expandedContentLines < minExpectedContentLines {
		t.Errorf("Expected at least %d content lines in expanded panel, got %d",
			minExpectedContentLines, expandedContentLines)
	}
}

// cmdLogTestSplitLines splits a string into lines (test helper).
func cmdLogTestSplitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// cmdLogTestTrimAnsi removes ANSI escape codes and trims spaces (test helper).
func cmdLogTestTrimAnsi(s string) string {
	// Simple ANSI removal - skip escape sequences
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find the end of the escape sequence
			i += 2
			for i < len(s) && !isLetter(s[i]) {
				i++
			}
			if i < len(s) {
				i++ // Skip the final letter
			}
		} else {
			result = append(result, s[i])
			i++
		}
	}
	// Trim spaces
	str := string(result)
	start, end := 0, len(str)
	for start < end && (str[start] == ' ' || str[start] == '\t') {
		start++
	}
	for end > start && (str[end-1] == ' ' || str[end-1] == '\t') {
		end--
	}
	return str[start:end]
}

func isLetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// cmdLogTestIsBorderLine checks if a line is just border characters (test helper).
func cmdLogTestIsBorderLine(s string) bool {
	for _, r := range s {
		if r != '─' && r != '│' && r != '┌' && r != '┐' && r != '└' && r != '┘' && r != '├' && r != '┤' && r != '┬' && r != '┴' && r != '┼' && r != ' ' {
			return false
		}
	}
	return true
}

func TestCommandLogPanelUpdateNilDiagnosticsPanel(t *testing.T) {
	panel := &CommandLogPanel{}
	panel.diagnosticsPanel = nil
	panel.styles = styles.DefaultStyles()

	result, cmd := panel.Update(tea.KeyMsg{Type: tea.KeyDown})
	if result == nil {
		t.Error("expected non-nil result")
	}
	if cmd != nil {
		t.Error("expected nil cmd when diagnosticsPanel is nil")
	}
}

func TestCommandLogPanelHandleKeyPgUpPgDown(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewCommandLogPanel(s)
	panel.SetSize(80, 20)
	panel.SetFocused(true)

	// Add content
	panel.AppendSessionLog("Test", "cmd", "output")

	// Test page up
	handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if handled {
		t.Error("expected 'p' key not to be handled")
	}

	// Test pgup
	handled, _ = panel.HandleKey(tea.KeyMsg{Type: tea.KeyPgUp})
	if !handled {
		t.Error("expected pgup to be handled")
	}

	// Test pgdown
	handled, _ = panel.HandleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	if !handled {
		t.Error("expected pgdown to be handled")
	}
}

func TestCommandLogPanelHandleKeyHomeEnd(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewCommandLogPanel(s)
	panel.SetSize(80, 20)
	panel.SetFocused(true)

	// Add content
	panel.AppendSessionLog("Test", "cmd", "output")

	// Test home
	handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyHome})
	if !handled {
		t.Error("expected home to be handled")
	}

	// Test end
	handled, _ = panel.HandleKey(tea.KeyMsg{Type: tea.KeyEnd})
	if !handled {
		t.Error("expected end to be handled")
	}
}

func TestCommandLogPanelCalculateThumbSizeSmallContent(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewCommandLogPanel(s)
	panel.SetSize(80, 40)

	// Small content - thumb should be large
	thumbSize := panel.calculateThumbSize(40, 5)
	// With small content, thumb should be 1.0
	if thumbSize != 1.0 {
		t.Errorf("expected thumb size 1.0 for small content, got %f", thumbSize)
	}
}

func TestCommandLogPanelCalculateThumbSizeLargeContent(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewCommandLogPanel(s)
	panel.SetSize(80, 10)

	// Large content
	thumbSize := panel.calculateThumbSize(10, 100)
	// With large content, thumb should be small
	if thumbSize > 0.2 {
		t.Errorf("expected small thumb size for large content, got %f", thumbSize)
	}
}

func TestCommandLogPanelCalculateThumbSizeZeroHeight(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewCommandLogPanel(s)

	// Zero height should return 1.0
	thumbSize := panel.calculateThumbSize(0, 100)
	if thumbSize != 1.0 {
		t.Errorf("expected thumb size 1.0 for zero height, got %f", thumbSize)
	}
}
