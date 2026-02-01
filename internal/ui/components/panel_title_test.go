package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderPanelTitleLine(t *testing.T) {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#555"))

	tests := []struct {
		name        string
		width       int
		title       string
		wantOK      bool
		wantNonEmpty bool
	}{
		{
			name:         "normal width with title",
			width:        40,
			title:        "Resources",
			wantOK:       true,
			wantNonEmpty: true,
		},
		{
			name:         "zero width",
			width:        0,
			title:        "Test",
			wantOK:       false,
			wantNonEmpty: false,
		},
		{
			name:         "negative width",
			width:        -10,
			title:        "Test",
			wantOK:       false,
			wantNonEmpty: false,
		},
		{
			name:         "empty title",
			width:        20,
			title:        "",
			wantOK:       true,
			wantNonEmpty: true,
		},
		{
			name:         "title too long for width",
			width:        5,
			title:        "This is a very long title that exceeds width",
			wantOK:       true,
			wantNonEmpty: true,
		},
		{
			name:         "exact width for title",
			width:        10,
			title:        "12345678",
			wantOK:       true,
			wantNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := RenderPanelTitleLine(tt.width, borderStyle, tt.title)

			if ok != tt.wantOK {
				t.Errorf("RenderPanelTitleLine ok = %v, want %v", ok, tt.wantOK)
			}

			if tt.wantNonEmpty && result == "" {
				t.Error("expected non-empty result")
			}

			if !tt.wantNonEmpty && result != "" {
				t.Errorf("expected empty result, got %q", result)
			}
		})
	}
}

func TestRenderPanelTitleLineWithDefaultBorder(t *testing.T) {
	// Test with empty border style (uses defaults)
	borderStyle := lipgloss.NewStyle()

	result, ok := RenderPanelTitleLine(20, borderStyle, "Title")
	if !ok {
		t.Error("expected ok=true for valid width")
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestBorderLine(t *testing.T) {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTopForeground(lipgloss.Color("#FF0000"))

	result := borderLine(borderStyle)
	// Just verify it returns a valid style
	_ = result.Render("test")
}
