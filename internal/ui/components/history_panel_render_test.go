package components

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func TestHistoryPanel_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		hp := NewHistoryPanel(styles.DefaultStyles())
		hp.SetEntries(testutil.SampleHistory)

		result := testutil.RenderComponent(t, hp, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t).
			AssertHasBorder(t)
	})
}

func TestHistoryPanel_EmptyState(t *testing.T) {
	hp := NewHistoryPanel(styles.DefaultStyles())
	hp.SetEntries(nil)

	result := testutil.RenderComponent(t, hp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertContains(t, "No items")
}

func TestHistoryPanel_WithEntries(t *testing.T) {
	hp := NewHistoryPanel(styles.DefaultStyles())
	hp.SetEntries(testutil.SampleHistory)

	result := testutil.RenderComponent(t, hp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertHasPanelID(t, "[3]")
}

func TestHistoryPanel_StatusRendering(t *testing.T) {
	hp := NewHistoryPanel(styles.DefaultStyles())

	statuses := []struct {
		name    string
		entries []history.Entry
	}{
		{
			name: "success",
			entries: []history.Entry{
				testutil.HistoryEntry(1, history.StatusSuccess, "Success op"),
			},
		},
		{
			name: "failed",
			entries: []history.Entry{
				testutil.HistoryEntry(2, history.StatusFailed, "Failed op"),
			},
		},
		{
			name: "canceled",
			entries: []history.Entry{
				testutil.HistoryEntry(3, history.StatusCanceled, "Canceled op"),
			},
		},
	}

	for _, tc := range statuses {
		t.Run(tc.name, func(t *testing.T) {
			hp.SetEntries(tc.entries)
			result := testutil.RenderComponent(t, hp, 80, 20)

			result.
				AssertNotEmpty(t).
				AssertHasBorder(t)
		})
	}
}

func TestHistoryPanel_SelectionHighlight(t *testing.T) {
	hp := NewHistoryPanel(styles.DefaultStyles())
	hp.SetEntries(testutil.SampleHistory)
	hp.SetSelection(1, true)

	result := testutil.RenderWithFocus(t, hp, 80, 20, true)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestHistoryPanel_FocusState(t *testing.T) {
	hp := NewHistoryPanel(styles.DefaultStyles())
	hp.SetEntries(testutil.SampleHistory)

	// Focused state
	focusedResult := testutil.RenderWithFocus(t, hp, 80, 20, true)
	focusedResult.
		AssertNotEmpty(t).
		AssertHasBorder(t)

	// Unfocused state
	unfocusedResult := testutil.RenderWithFocus(t, hp, 80, 20, false)
	unfocusedResult.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestHistoryPanel_ScrollbarAppearsWhenNeeded(t *testing.T) {
	hp := NewHistoryPanel(styles.DefaultStyles())

	// Few entries - no scrollbar
	hp.SetEntries([]history.Entry{
		testutil.HistoryEntry(1, history.StatusSuccess, "One"),
	})
	result := testutil.RenderComponent(t, hp, 40, 10)
	result.AssertNoScrollbar(t)

	// Many entries - scrollbar should appear
	hp.SetEntries(testutil.ManyHistoryEntries(50))
	result = testutil.RenderComponent(t, hp, 40, 10)
	result.AssertHasScrollbar(t)
}

func TestHistoryPanel_HeightRespected(t *testing.T) {
	for _, h := range []int{5, 10, 15, 20} {
		t.Run(testutil.DimensionSet{Name: "height", Height: h}.String(), func(t *testing.T) {
			hp := NewHistoryPanel(styles.DefaultStyles())
			hp.SetEntries(testutil.ManyHistoryEntries(50))

			result := testutil.RenderComponent(t, hp, 80, h)

			result.
				AssertHeight(t, h).
				AssertNoLineOverflow(t)
		})
	}
}

func TestHistoryPanel_MinimalDimensions(t *testing.T) {
	hp := NewHistoryPanel(styles.DefaultStyles())
	hp.SetEntries(testutil.SampleHistory)

	// Very small dimensions should not panic
	result := testutil.RenderComponent(t, hp, 15, 5)
	result.AssertNotEmpty(t)
}

func TestHistoryPanel_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		hp := NewHistoryPanel(styles.DefaultStyles())
		hp.SetEntries(testutil.SampleHistory)

		result := testutil.RenderComponent(t, hp, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t)
	})
}

func TestHistoryPanel_ItemCount(t *testing.T) {
	hp := NewHistoryPanel(styles.DefaultStyles())
	hp.SetEntries(testutil.ManyHistoryEntries(29))
	hp.SetSelection(6, true)

	result := testutil.RenderComponent(t, hp, 80, 20)

	// Should show item count like "7 of 29"
	result.AssertHasItemCount(t)
}
