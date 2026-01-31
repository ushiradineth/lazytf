package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func TestPanelFrame_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		frame := NewPanelFrame(styles.DefaultStyles())
		frame.SetSize(d.Width, d.Height)
		frame.SetConfig(PanelFrameConfig{
			PanelID:   "[1]",
			Tabs:      []string{"Test Panel"},
			ActiveTab: 0,
			Focused:   false,
		})

		content := make([]string, frame.ContentHeight())
		for i := range content {
			content[i] = strings.Repeat(".", frame.ContentWidth())
		}

		result := testutil.RenderCapture(t, func() string {
			return frame.RenderWithContent(content)
		}, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t).
			AssertHasBorder(t)
	})
}

func TestPanelFrame_FocusedStyle(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(80, 20)

	// Unfocused
	frame.SetConfig(PanelFrameConfig{
		PanelID:   "[1]",
		Tabs:      []string{"Panel"},
		ActiveTab: 0,
		Focused:   false,
	})
	unfocusedResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)

	// Focused
	frame.SetConfig(PanelFrameConfig{
		PanelID:   "[1]",
		Tabs:      []string{"Panel"},
		ActiveTab: 0,
		Focused:   true,
	})
	focusedResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)

	// Both should render correctly with borders
	unfocusedResult.AssertHasBorder(t).AssertNotEmpty(t)
	focusedResult.AssertHasBorder(t).AssertNotEmpty(t)

	// Both should have proper height
	unfocusedResult.AssertHeight(t, 20)
	focusedResult.AssertHeight(t, 20)
}

func TestPanelFrame_Scrollbar(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(40, 10)

	// Without scrollbar
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		Tabs:          []string{"Panel"},
		ActiveTab:     0,
		ShowScrollbar: false,
	})
	noScrollResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"line1", "line2"})
	}, 40, 10)
	noScrollResult.AssertNoScrollbar(t)

	// With scrollbar
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		Tabs:          []string{"Panel"},
		ActiveTab:     0,
		ShowScrollbar: true,
		ScrollPos:     0.5,
		ThumbSize:     0.3,
	})
	scrollResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"line1", "line2"})
	}, 40, 10)
	scrollResult.AssertHasScrollbar(t)
}

func TestPanelFrame_ScrollbarPosition(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(40, 15)

	// Scrollbar at top
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		Tabs:          []string{"Panel"},
		ShowScrollbar: true,
		ScrollPos:     0.0,
		ThumbSize:     0.2,
	})
	topResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent(make([]string, frame.ContentHeight()))
	}, 40, 15)

	// Scrollbar at bottom
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		Tabs:          []string{"Panel"},
		ShowScrollbar: true,
		ScrollPos:     1.0,
		ThumbSize:     0.2,
	})
	bottomResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent(make([]string, frame.ContentHeight()))
	}, 40, 15)

	// Scrollbar position should differ
	topResult.AssertDifferentFrom(t, bottomResult, "scrollbar at top vs bottom")
}

func TestPanelFrame_Tabs(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(80, 20)

	// Single tab
	frame.SetConfig(PanelFrameConfig{
		PanelID:   "[1]",
		Tabs:      []string{"Single"},
		ActiveTab: 0,
	})
	singleResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)
	singleResult.AssertContains(t, "Single")

	// Multiple tabs
	frame.SetConfig(PanelFrameConfig{
		PanelID:   "[2]",
		Tabs:      []string{"Tab A", "Tab B"},
		ActiveTab: 0,
	})
	multiResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)
	multiResult.AssertContainsAll(t, "Tab A", "Tab B")
}

func TestPanelFrame_ActiveTab(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(80, 20)

	// Tab A active
	frame.SetConfig(PanelFrameConfig{
		PanelID:   "[2]",
		Tabs:      []string{"Tab A", "Tab B"},
		ActiveTab: 0,
		Focused:   true,
	})
	tabAResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)

	// Tab B active
	frame.SetConfig(PanelFrameConfig{
		PanelID:   "[2]",
		Tabs:      []string{"Tab A", "Tab B"},
		ActiveTab: 1,
		Focused:   true,
	})
	tabBResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)

	// Both should contain both tab names
	tabAResult.AssertContainsAll(t, "Tab A", "Tab B")
	tabBResult.AssertContainsAll(t, "Tab A", "Tab B")

	// Both should render correctly
	tabAResult.AssertHasBorder(t).AssertNotEmpty(t)
	tabBResult.AssertHasBorder(t).AssertNotEmpty(t)
}

func TestPanelFrame_Footer(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(80, 20)

	// Without footer
	frame.SetConfig(PanelFrameConfig{
		PanelID:    "[1]",
		Tabs:       []string{"Panel"},
		FooterText: "",
	})
	noFooterResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)

	// With footer
	frame.SetConfig(PanelFrameConfig{
		PanelID:    "[1]",
		Tabs:       []string{"Panel"},
		FooterText: " 7 of 29 ",
	})
	footerResult := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)
	footerResult.AssertContains(t, "7 of 29")

	// Should be different
	noFooterResult.AssertDifferentFrom(t, footerResult, "no footer vs with footer")
}

func TestPanelFrame_PanelID(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(80, 20)

	panelIDs := []string{"[1]", "[2]", "[3]", "[4]", "[0]"}
	for _, id := range panelIDs {
		t.Run(id, func(t *testing.T) {
			frame.SetConfig(PanelFrameConfig{
				PanelID: id,
				Tabs:    []string{"Panel"},
			})
			result := testutil.RenderCapture(t, func() string {
				return frame.RenderWithContent([]string{"content"})
			}, 80, 20)
			result.AssertHasPanelID(t, id)
		})
	}
}

func TestPanelFrame_RoundedBorder(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(80, 20)
	frame.SetConfig(PanelFrameConfig{
		PanelID: "[1]",
		Tabs:    []string{"Panel"},
	})

	result := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"content"})
	}, 80, 20)

	result.AssertHasRoundedBorder(t)
}

func TestPanelFrame_MinimalDimensions(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(10, 5)
	frame.SetConfig(PanelFrameConfig{
		PanelID: "[1]",
		Tabs:    []string{"X"},
	})

	// Should not panic
	result := testutil.RenderCapture(t, func() string {
		return frame.RenderWithContent([]string{"x"})
	}, 10, 5)
	result.AssertNotEmpty(t)
}

func TestPanelFrame_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		frame := NewPanelFrame(styles.DefaultStyles())
		frame.SetSize(d.Width, d.Height)
		frame.SetConfig(PanelFrameConfig{
			PanelID: "[1]",
			Tabs:    []string{"Widescreen Panel"},
		})

		content := make([]string, frame.ContentHeight())
		for i := range content {
			content[i] = "Line " + testutil.IntToString(i)
		}

		result := testutil.RenderCapture(t, func() string {
			return frame.RenderWithContent(content)
		}, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t)
	})
}

func TestPanelFrame_ContentDimensionsCalculation(t *testing.T) {
	frame := NewPanelFrame(styles.DefaultStyles())
	frame.SetSize(80, 20)
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		Tabs:          []string{"Panel"},
		ShowScrollbar: false,
	})

	// Content width should be width - 2 (borders)
	expectedContentWidth := 78
	if frame.ContentWidth() != expectedContentWidth {
		t.Errorf("content width: got %d, want %d", frame.ContentWidth(), expectedContentWidth)
	}

	// Content height should be height - 2 (top/bottom borders)
	expectedContentHeight := 18
	if frame.ContentHeight() != expectedContentHeight {
		t.Errorf("content height: got %d, want %d", frame.ContentHeight(), expectedContentHeight)
	}

	// With scrollbar, content width should be width - 3
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		Tabs:          []string{"Panel"},
		ShowScrollbar: true,
	})
	expectedContentWidthWithScrollbar := 77
	if frame.ContentWidth() != expectedContentWidthWithScrollbar {
		t.Errorf("content width with scrollbar: got %d, want %d", frame.ContentWidth(), expectedContentWidthWithScrollbar)
	}
}
