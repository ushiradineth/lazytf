package components

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func TestResourceList_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		rl := NewResourceList(styles.DefaultStyles())
		rl.SetResources(testutil.SampleResources)

		result := testutil.RenderComponent(t, rl, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t).
			AssertHasBorder(t)
	})
}

func TestResourceList_EmptyState(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(nil)

	result := testutil.RenderComponent(t, rl, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertHasPanelID(t, "[2]")
}

func TestResourceList_WithResources(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)

	result := testutil.RenderComponent(t, rl, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertHasPanelID(t, "[2]")
}

func TestResourceList_FocusState(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)

	// Focused state
	focusedResult := testutil.RenderWithFocus(t, rl, 80, 20, true)
	focusedResult.
		AssertNotEmpty(t).
		AssertHasBorder(t)

	// Unfocused state
	unfocusedResult := testutil.RenderWithFocus(t, rl, 80, 20, false)
	unfocusedResult.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestResourceList_HeightRespected(t *testing.T) {
	for _, h := range []int{5, 10, 15, 20, 30} {
		t.Run(testutil.DimensionSet{Name: "height", Height: h}.String(), func(t *testing.T) {
			rl := NewResourceList(styles.DefaultStyles())
			rl.SetResources(testutil.ManyResources)

			result := testutil.RenderComponent(t, rl, 80, h)

			result.
				AssertHeight(t, h).
				AssertNoLineOverflow(t)
		})
	}
}

func TestResourceList_ScrollbarAppearsWhenNeeded(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())

	// Few resources - no scrollbar
	rl.SetResources(testutil.FewResources)
	result := testutil.RenderComponent(t, rl, 40, 15)
	result.AssertNoScrollbar(t)

	// Many resources - scrollbar should appear
	rl.SetResources(testutil.ManyResources)
	result = testutil.RenderComponent(t, rl, 40, 10)
	result.AssertHasScrollbar(t)
}

func TestResourceList_MinimalDimensions(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)

	// Very small dimensions should not panic
	result := testutil.RenderComponent(t, rl, 15, 5)
	result.AssertNotEmpty(t)
}

func TestResourceList_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		rl := NewResourceList(styles.DefaultStyles())
		rl.SetResources(testutil.SampleResources)

		result := testutil.RenderComponent(t, rl, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t)
	})
}

func TestResourceList_ActionTypes(t *testing.T) {
	actions := []terraform.ActionType{
		terraform.ActionCreate,
		terraform.ActionUpdate,
		terraform.ActionDelete,
		terraform.ActionReplace,
	}

	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			rl := NewResourceList(styles.DefaultStyles())
			rl.SetResources([]terraform.ResourceChange{
				testutil.ResourceWithAction(action),
			})

			result := testutil.RenderComponent(t, rl, 80, 20)

			result.
				AssertNotEmpty(t).
				AssertHasBorder(t)
		})
	}
}

func TestResourceList_ModuleResources(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.ModuleResources)

	result := testutil.RenderComponent(t, rl, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestResourceList_ItemCount(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.ManyResources)
	rl.SetFocused(true)

	result := testutil.RenderComponent(t, rl, 80, 20)

	// Should show item count in footer when focused
	result.AssertHasItemCount(t)
}

func TestResourceList_Selection(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)
	rl.SetFocused(true)

	// Select different items
	rl.SetSelectedIndex(0)
	result0 := testutil.RenderComponent(t, rl, 80, 20)
	result0.AssertNotEmpty(t)

	rl.SetSelectedIndex(2)
	result2 := testutil.RenderComponent(t, rl, 80, 20)
	result2.AssertNotEmpty(t)
}

func TestResourceList_LongAddresses(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	// Use module resources which have longer addresses
	rl.SetResources(testutil.ModuleResources)

	result := testutil.RenderComponent(t, rl, 60, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertNoLineOverflow(t)
}
