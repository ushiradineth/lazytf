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

		// Only check for "Resources" title when width is sufficient
		if d.Width >= 40 {
			result.AssertContains(t, "Resources")
		}
	})
}

func TestResourceList_SelectionHighlight(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)
	rl.SetSelectedIndex(0)
	rl.SetFocused(true)

	result := testutil.RenderWithFocus(t, rl, 80, 20, true)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)

	// The first sample resource should be visible
	result.AssertContainsAny(t, "aws_instance", "new_server")
}

func TestResourceList_EmptyState(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(nil)

	result := testutil.RenderComponent(t, rl, 80, 20)

	result.
		AssertContains(t, "No resources").
		AssertHasBorder(t)
}

func TestResourceList_ScrollbarAppearsWhenNeeded(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())

	// Few resources - no scrollbar
	rl.SetResources(testutil.FewResources)
	result := testutil.RenderComponent(t, rl, 40, 10)
	result.AssertNoScrollbar(t)

	// Many resources - scrollbar should appear
	rl.SetResources(testutil.ManyResources)
	result = testutil.RenderComponent(t, rl, 40, 10)
	result.AssertHasScrollbar(t)
}

func TestResourceList_FocusState(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)

	// Test focused state
	focusedResult := testutil.RenderWithFocus(t, rl, 80, 20, true)
	focusedResult.
		AssertHasBorder(t).
		AssertNotEmpty(t)

	// Test unfocused state
	unfocusedResult := testutil.RenderWithFocus(t, rl, 80, 20, false)
	unfocusedResult.
		AssertHasBorder(t).
		AssertNotEmpty(t)
}

func TestResourceList_WithModules(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.ModuleResources)

	result := testutil.RenderComponent(t, rl, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestResourceList_HeightRespected(t *testing.T) {
	for _, h := range []int{5, 10, 20, 30} {
		t.Run(testutil.DimensionSet{Name: "height", Height: h}.String(), func(t *testing.T) {
			rl := NewResourceList(styles.DefaultStyles())
			rl.SetResources(testutil.ManyResources) // Many resources to overflow

			result := testutil.RenderComponent(t, rl, 80, h)

			result.
				AssertHeight(t, h).
				AssertNoLineOverflow(t)
		})
	}
}

func TestResourceList_Summary(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)
	rl.SetSummary(3, 2, 1, 1) // create, update, delete, replace

	result := testutil.RenderComponent(t, rl, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertContainsAny(t, "+3", "~2", "-1")
}

func TestResourceList_FilterIndicators(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)

	// With all filters enabled
	result := testutil.RenderComponent(t, rl, 80, 20)
	result.AssertContainsAll(t, "C", "D", "R", "U")

	// Disable a filter
	rl.SetFilter(terraform.ActionDelete, false)
	result = testutil.RenderComponent(t, rl, 80, 20)
	result.AssertContainsAll(t, "C", "R", "U") // D still shown but may be dimmed
}

func TestResourceList_MinimalDimensions(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	rl.SetResources(testutil.SampleResources)

	// Very small dimensions should not panic
	result := testutil.RenderComponent(t, rl, 10, 5)
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

func TestResourceList_StatusDisplay(t *testing.T) {
	rl := NewResourceList(styles.DefaultStyles())
	resources := []terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	}
	rl.SetResources(resources)

	// Without status
	rl.SetShowStatus(false)
	result := testutil.RenderComponent(t, rl, 80, 20)
	result.AssertNotEmpty(t)

	// With status
	state := terraform.NewOperationState()
	state.StartResource("aws_instance.web", terraform.ActionCreate)
	rl.SetOperationState(state)
	rl.SetShowStatus(true)

	result = testutil.RenderComponent(t, rl, 80, 20)
	result.AssertNotEmpty(t)
}
