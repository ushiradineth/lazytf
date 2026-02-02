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

func TestPanelFrame_SetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)

	newStyles := styles.DefaultStyles()
	frame.SetStyles(newStyles)

	if frame.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestNewPanelFrameNilStyles(t *testing.T) {
	// NewPanelFrame should use DefaultStyles when passed nil
	frame := NewPanelFrame(nil)
	if frame == nil {
		t.Fatal("expected non-nil frame")
	}
	if frame.styles == nil {
		t.Error("expected styles to be set to DefaultStyles")
	}
}

func TestPanelFrame_ContentWidthMinimum(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	// Set very small size
	frame.SetSize(1, 1)
	frame.SetConfig(PanelFrameConfig{
		ShowScrollbar: true,
	})

	// ContentWidth should return at least 1
	if frame.ContentWidth() < 1 {
		t.Errorf("ContentWidth should be at least 1, got %d", frame.ContentWidth())
	}
}

func TestPanelFrame_ContentHeightMinimum(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	// Set very small size
	frame.SetSize(1, 1)
	frame.SetConfig(PanelFrameConfig{
		FooterText: "footer",
	})

	// ContentHeight should return at least 1
	if frame.ContentHeight() < 1 {
		t.Errorf("ContentHeight should be at least 1, got %d", frame.ContentHeight())
	}
}

func TestCalculateScrollParamsZeroVisibleHeight(t *testing.T) {
	// When visibleHeight is 0, should return defaults
	pos, size := CalculateScrollParams(0, 0, 10)
	if pos != 0 || size != 1.0 {
		t.Errorf("expected (0, 1.0) for zero visible height, got (%v, %v)", pos, size)
	}
}

func TestCalculateScrollParamsNegativeVisibleHeight(t *testing.T) {
	// When visibleHeight is negative, should return defaults
	pos, size := CalculateScrollParams(0, -5, 10)
	if pos != 0 || size != 1.0 {
		t.Errorf("expected (0, 1.0) for negative visible height, got (%v, %v)", pos, size)
	}
}

func TestCalculateScrollParamsExcessiveScrollOffset(t *testing.T) {
	// When scrollOffset exceeds scroll range, scrollPos should be capped at 1.0
	pos, size := CalculateScrollParams(100, 5, 20)
	if pos != 1.0 {
		t.Errorf("expected scrollPos capped at 1.0, got %v", pos)
	}
	if size != 0.25 {
		t.Errorf("expected thumbSize 0.25, got %v", size)
	}
}

func TestCalculateScrollParamsEqualItemsAndHeight(t *testing.T) {
	// When totalItems equals visibleHeight
	pos, size := CalculateScrollParams(0, 10, 10)
	if pos != 0 || size != 1.0 {
		t.Errorf("expected (0, 1.0) when items equal height, got (%v, %v)", pos, size)
	}
}

func TestPanelFrame_RenderWithEmptyBorderChars(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 10)
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		ShowScrollbar: true,
		ScrollPos:     0.5,
		ThumbSize:     0.3,
	})

	// Render should work with default styles (which may have empty border chars)
	view := frame.RenderWithContent([]string{"line1", "line2"})
	if view == "" {
		t.Error("expected non-empty view")
	}
	// View should contain border characters
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Error("expected default border characters to be used")
	}
}

func TestPanelFrame_RenderScrollbarAtMiddle(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 8)
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		ShowScrollbar: true,
		ScrollPos:     0.5,
		ThumbSize:     0.2,
	})

	view := frame.RenderWithContent([]string{"line1", "line2", "line3", "line4", "line5", "line6"})
	// Should contain scrollbar thumb
	if !strings.Contains(view, "▐") {
		t.Error("expected scrollbar thumb in middle position")
	}
}

func TestPanelFrame_RenderScrollbarAtBottom(t *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)
	frame.SetSize(40, 6)
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		ShowScrollbar: true,
		ScrollPos:     1.0,
		ThumbSize:     0.25,
	})

	view := frame.RenderWithContent([]string{"line1", "line2", "line3", "line4"})
	// Should render without error
	if view == "" {
		t.Error("expected non-empty view")
	}
}
