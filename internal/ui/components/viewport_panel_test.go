package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestViewportPanel_Basic(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewViewportPanel("[0]", s)
	panel.SetSize(40, 10)

	view := panel.View()
	if !strings.Contains(view, "[0]") {
		t.Error("Panel should show panel ID")
	}
}

func TestViewportPanel_WithContent(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewViewportPanel("[0]", s)
	panel.SetSize(40, 10)
	panel.SetContent("Line 1\nLine 2\nLine 3")

	view := panel.View()
	if !strings.Contains(view, "Line 1") {
		t.Error("Panel should show content")
	}
}

func TestViewportPanel_WithTabs(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewViewportPanel("[0]", s)
	panel.SetTabs([]string{"Diff", "Logs"})
	panel.SetSize(40, 10)

	view := panel.View()
	if !strings.Contains(view, "Diff") {
		t.Error("Panel should show first tab")
	}
}

func TestViewportPanel_Scrollbar(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewViewportPanel("[0]", s)
	panel.SetSize(40, 5) // Small height

	// Create content that exceeds panel height
	lines := make([]string, 0, 20)
	for range 20 {
		lines = append(lines, "Content line")
	}
	panel.SetContent(strings.Join(lines, "\n"))

	view := panel.View()
	// Scrollbar should appear (▐ character)
	if !strings.Contains(view, "▐") {
		t.Error("Panel should show scrollbar when content exceeds height")
	}
}

func TestViewportPanel_FocusState(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewViewportPanel("[0]", s)
	panel.SetSize(40, 10)

	if panel.IsFocused() {
		t.Error("Panel should not be focused initially")
	}

	panel.SetFocused(true)
	if !panel.IsFocused() {
		t.Error("Panel should be focused after SetFocused(true)")
	}
}

func TestViewportPanel_Navigation(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewViewportPanel("[0]", s)
	panel.SetSize(40, 5)

	// Create scrollable content
	lines := make([]string, 0, 20)
	for range 20 {
		lines = append(lines, "Content line")
	}
	panel.SetContent(strings.Join(lines, "\n"))

	// Test navigation methods don't panic
	panel.GotoTop()
	panel.ScrollDown()
	panel.ScrollUp()
	panel.PageDown()
	panel.PageUp()
	panel.GotoBottom()

	// Should still render without error
	view := panel.View()
	if view == "" {
		t.Error("Panel should render after navigation")
	}
}
