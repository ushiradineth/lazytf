package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

// testListItem implements ListPanelItem for testing.
type testListItem struct {
	text string
}

func (t testListItem) Render(st *styles.Styles, width int, selected bool) string {
	line := t.text
	if len(line) > width {
		line = line[:width]
	}
	prefix := "  "
	if selected {
		prefix = "> "
	}
	result := prefix + line
	if len(result) < width {
		result = result + strings.Repeat(" ", width-len(result))
	}
	return result
}

func createTestItems(n int) []ListPanelItem {
	items := make([]ListPanelItem, n)
	for i := range n {
		items[i] = testListItem{text: "Item " + testutil.IntToString(i)}
	}
	return items
}

func TestListPanel_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		lp := NewListPanel("[0]", styles.DefaultStyles())
		lp.SetItems(createTestItems(5))

		result := testutil.RenderComponent(t, lp, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t).
			AssertHasBorder(t)
	})
}

func TestListPanel_EmptyState(t *testing.T) {
	lp := NewListPanel("[0]", styles.DefaultStyles())
	lp.SetItems(nil)

	result := testutil.RenderComponent(t, lp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertHasPanelID(t, "[0]")
}

func TestListPanel_RenderWithItems(t *testing.T) {
	lp := NewListPanel("[0]", styles.DefaultStyles())
	lp.SetTabs([]string{"Items"})
	lp.SetItems(createTestItems(10))

	result := testutil.RenderComponent(t, lp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertHasPanelID(t, "[0]")
}

func TestListPanel_RenderFocusState(t *testing.T) {
	lp := NewListPanel("[0]", styles.DefaultStyles())
	lp.SetItems(createTestItems(5))

	// Focused state
	focusedResult := testutil.RenderWithFocus(t, lp, 80, 20, true)
	focusedResult.
		AssertNotEmpty(t).
		AssertHasBorder(t)

	// Unfocused state
	unfocusedResult := testutil.RenderWithFocus(t, lp, 80, 20, false)
	unfocusedResult.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestListPanel_HeightRespected(t *testing.T) {
	for _, h := range []int{5, 10, 15, 20, 30} {
		t.Run(testutil.DimensionSet{Name: "height", Height: h}.String(), func(t *testing.T) {
			lp := NewListPanel("[0]", styles.DefaultStyles())
			lp.SetItems(createTestItems(50))

			result := testutil.RenderComponent(t, lp, 80, h)

			result.
				AssertHeight(t, h).
				AssertNoLineOverflow(t)
		})
	}
}

func TestListPanel_ScrollbarAppearsWhenNeeded(t *testing.T) {
	lp := NewListPanel("[0]", styles.DefaultStyles())

	// Few items - no scrollbar
	lp.SetItems(createTestItems(3))
	result := testutil.RenderComponent(t, lp, 40, 15)
	result.AssertNoScrollbar(t)

	// Many items - scrollbar should appear
	lp.SetItems(createTestItems(50))
	result = testutil.RenderComponent(t, lp, 40, 10)
	result.AssertHasScrollbar(t)
}

func TestListPanel_MinimalDimensions(t *testing.T) {
	lp := NewListPanel("[0]", styles.DefaultStyles())
	lp.SetItems(createTestItems(5))

	// Very small dimensions should not panic
	result := testutil.RenderComponent(t, lp, 15, 5)
	result.AssertNotEmpty(t)
}

func TestListPanel_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		lp := NewListPanel("[0]", styles.DefaultStyles())
		lp.SetItems(createTestItems(10))

		result := testutil.RenderComponent(t, lp, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t)
	})
}

func TestListPanel_RenderItemCount(t *testing.T) {
	lp := NewListPanel("[0]", styles.DefaultStyles())
	lp.SetItems(createTestItems(29))
	lp.SetFocused(true)
	lp.SetSelectedIndex(6)

	result := testutil.RenderComponent(t, lp, 80, 20)

	// Should show item count like "7 of 29"
	result.AssertHasItemCount(t)
}

func TestListPanel_Selection(t *testing.T) {
	lp := NewListPanel("[0]", styles.DefaultStyles())
	lp.SetItems(createTestItems(10))
	lp.SetFocused(true)

	// Select different items
	lp.SetSelectedIndex(0)
	result0 := testutil.RenderComponent(t, lp, 80, 20)
	result0.AssertNotEmpty(t)

	lp.SetSelectedIndex(5)
	result5 := testutil.RenderComponent(t, lp, 80, 20)
	result5.AssertNotEmpty(t)
}

func TestListPanel_RenderTabs(t *testing.T) {
	lp := NewListPanel("[0]", styles.DefaultStyles())
	lp.SetTabs([]string{"Tab1", "Tab2", "Tab3"})
	lp.SetActiveTab(1)
	lp.SetItems(createTestItems(5))

	result := testutil.RenderComponent(t, lp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}
