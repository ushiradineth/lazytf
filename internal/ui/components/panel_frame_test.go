package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestPanelFrame_Basic(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 10)
	frame.SetConfig(PanelFrameConfig{
		PanelID: "[2]",
	})

	content := []string{"line 1", "line 2", "line 3"}
	view := frame.RenderWithContent(content)

	if !strings.Contains(view, "[2]") {
		t.Error("Frame should show panel ID")
	}
	if !strings.Contains(view, "line 1") {
		t.Error("Frame should show content")
	}
}

func TestPanelFrame_WithTabs(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 10)
	frame.SetConfig(PanelFrameConfig{
		PanelID:   "[2]",
		Tabs:      []string{"Tab1", "Tab2"},
		ActiveTab: 0,
	})

	view := frame.RenderWithContent(nil)
	if !strings.Contains(view, "Tab1") {
		t.Error("Frame should show first tab")
	}
	if !strings.Contains(view, "Tab2") {
		t.Error("Frame should show second tab")
	}
}

func TestPanelFrame_WithFooter(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 10)
	frame.SetConfig(PanelFrameConfig{
		PanelID:    "[2]",
		FooterText: " 5 of 10 ",
	})

	view := frame.RenderWithContent(nil)
	if !strings.Contains(view, "5 of 10") {
		t.Error("Frame should show footer text")
	}
}

func TestPanelFrame_WithScrollbar(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 5)
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[2]",
		ShowScrollbar: true,
		ScrollPos:     0.0,
		ThumbSize:     0.3,
	})

	view := frame.RenderWithContent([]string{"line 1", "line 2", "line 3"})
	if !strings.Contains(view, "▐") {
		t.Error("Frame should show scrollbar thumb")
	}
}

func TestPanelFrame_FocusedState(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 10)

	// Unfocused - should render without error
	frame.SetConfig(PanelFrameConfig{
		PanelID: "[2]",
		Focused: false,
	})
	unfocusedView := frame.RenderWithContent(nil)
	if unfocusedView == "" {
		t.Error("Unfocused view should not be empty")
	}
	if !strings.Contains(unfocusedView, "[2]") {
		t.Error("Unfocused view should contain panel ID")
	}

	// Focused - should render without error
	frame.SetConfig(PanelFrameConfig{
		PanelID: "[2]",
		Focused: true,
	})
	focusedView := frame.RenderWithContent(nil)
	if focusedView == "" {
		t.Error("Focused view should not be empty")
	}
	if !strings.Contains(focusedView, "[2]") {
		t.Error("Focused view should contain panel ID")
	}
}

func TestPanelFrame_ContentDimensions(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 10)
	frame.SetConfig(PanelFrameConfig{
		ShowScrollbar: false,
	})

	// Without scrollbar
	if frame.ContentWidth() != 38 { // 40 - 2 borders
		t.Errorf("ContentWidth should be 38, got %d", frame.ContentWidth())
	}
	if frame.ContentHeight() != 8 { // 10 - 2 borders
		t.Errorf("ContentHeight should be 8, got %d", frame.ContentHeight())
	}

	// With scrollbar
	frame.SetConfig(PanelFrameConfig{
		ShowScrollbar: true,
	})
	if frame.ContentWidth() != 37 { // 40 - 2 borders - 1 scrollbar
		t.Errorf("ContentWidth with scrollbar should be 37, got %d", frame.ContentWidth())
	}
}

func TestPanelFrame_SingleTab(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 10)
	frame.SetConfig(PanelFrameConfig{
		PanelID: "[2]",
		Tabs:    []string{"Resources"},
	})

	view := frame.RenderWithContent(nil)
	if !strings.Contains(view, "[2] Resources") {
		t.Error("Frame with single tab should show '[2] TabName'")
	}
}

func TestFormatItemCount(t *testing.T) {
	tests := []struct {
		current  int
		total    int
		expected string
	}{
		{1, 10, " 1 of 10 "},
		{5, 5, " 5 of 5 "},
		{0, 0, ""},
		{100, 1000, " 100 of 1000 "},
	}

	for _, tt := range tests {
		result := FormatItemCount(tt.current, tt.total)
		if result != tt.expected {
			t.Errorf("FormatItemCount(%d, %d) = %q, want %q", tt.current, tt.total, result, tt.expected)
		}
	}
}

func TestCalculateScrollParams(t *testing.T) {
	tests := []struct {
		name          string
		scrollOffset  int
		visibleHeight int
		totalItems    int
		wantPos       float64
		wantSize      float64
	}{
		{
			name:          "all items visible",
			scrollOffset:  0,
			visibleHeight: 10,
			totalItems:    5,
			wantPos:       0,
			wantSize:      1.0,
		},
		{
			name:          "at top",
			scrollOffset:  0,
			visibleHeight: 5,
			totalItems:    20,
			wantPos:       0,
			wantSize:      0.25,
		},
		{
			name:          "at bottom",
			scrollOffset:  15,
			visibleHeight: 5,
			totalItems:    20,
			wantPos:       1.0,
			wantSize:      0.25,
		},
		{
			name:          "middle",
			scrollOffset:  7,
			visibleHeight: 5,
			totalItems:    20,
			wantPos:       0.4666666666666667,
			wantSize:      0.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, size := CalculateScrollParams(tt.scrollOffset, tt.visibleHeight, tt.totalItems)
			if pos != tt.wantPos {
				t.Errorf("scrollPos = %v, want %v", pos, tt.wantPos)
			}
			if size != tt.wantSize {
				t.Errorf("thumbSize = %v, want %v", size, tt.wantSize)
			}
		})
	}
}
