package components

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func createTestEnvironments() []environment.Environment {
	return []environment.Environment{
		{Name: "dev", Path: "/path/to/dev", IsCurrent: true, Strategy: environment.StrategyWorkspace},
		{Name: "staging", Path: "/path/to/staging", IsCurrent: false, Strategy: environment.StrategyWorkspace},
		{Name: "prod", Path: "/path/to/prod", IsCurrent: false, Strategy: environment.StrategyWorkspace},
	}
}

func createManyEnvironments(n int) []environment.Environment {
	envs := make([]environment.Environment, n)
	for i := range n {
		envs[i] = environment.Environment{
			Name:      "env-" + testutil.IntToString(i),
			Path:      "/path/to/env-" + testutil.IntToString(i),
			IsCurrent: i == 0,
			Strategy:  environment.StrategyWorkspace,
		}
	}
	return envs
}

func TestEnvironmentPanel_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		ep := NewEnvironmentPanel(styles.DefaultStyles())
		ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())

		result := testutil.RenderComponent(t, ep, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t).
			AssertHasBorder(t)
	})
}

func TestEnvironmentPanel_EmptyState(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())
	ep.SetEnvironmentInfo("", "", environment.StrategyWorkspace, nil)

	result := testutil.RenderComponent(t, ep, 80, 10)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertContains(t, "No workspaces")
}

func TestEnvironmentPanel_WithEnvironments(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())
	ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())

	result := testutil.RenderComponent(t, ep, 80, 10)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertHasPanelID(t, "[1]")
}

func TestEnvironmentPanel_CurrentMarker(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())
	ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())
	ep.SetFocused(true)

	result := testutil.RenderComponent(t, ep, 80, 10)

	// Should show current environment marker (*)
	result.
		AssertNotEmpty(t).
		AssertContains(t, "*")
}

func TestEnvironmentPanel_FocusState(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())
	ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())

	// Focused state
	focusedResult := testutil.RenderWithFocus(t, ep, 80, 10, true)

	// Unfocused state
	unfocusedResult := testutil.RenderWithFocus(t, ep, 80, 10, false)

	// Should have different styling
	focusedResult.AssertStyleDifferentFrom(t, unfocusedResult, "focused vs unfocused environment panel")
}

func TestEnvironmentPanel_SelectionHighlight(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())
	ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())
	ep.SetFocused(true)
	ep.SetSelectedIndex(1)

	result := testutil.RenderWithFocus(t, ep, 80, 10, true)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestEnvironmentPanel_ScrollbarAppearsWhenNeeded(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())

	// Few environments - no scrollbar
	ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())
	ep.SetFocused(true)
	result := testutil.RenderComponent(t, ep, 40, 10)
	result.AssertNoScrollbar(t)

	// Many environments - scrollbar should appear
	ep.SetEnvironmentInfo("env-0", "", environment.StrategyWorkspace, createManyEnvironments(50))
	ep.SetFocused(true)
	result = testutil.RenderComponent(t, ep, 40, 10)
	result.AssertHasScrollbar(t)
}

func TestEnvironmentPanel_HeightRespected(t *testing.T) {
	for _, h := range []int{5, 10, 15, 20} {
		t.Run(testutil.DimensionSet{Name: "height", Height: h}.String(), func(t *testing.T) {
			ep := NewEnvironmentPanel(styles.DefaultStyles())
			ep.SetEnvironmentInfo("env-0", "", environment.StrategyWorkspace, createManyEnvironments(50))

			result := testutil.RenderComponent(t, ep, 80, h)

			result.
				AssertHeight(t, h).
				AssertNoLineOverflow(t)
		})
	}
}

func TestEnvironmentPanel_MinimalDimensions(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())
	ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())

	// Very small dimensions should not panic
	result := testutil.RenderComponent(t, ep, 15, 5)
	result.AssertNotEmpty(t)
}

func TestEnvironmentPanel_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		ep := NewEnvironmentPanel(styles.DefaultStyles())
		ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())

		result := testutil.RenderComponent(t, ep, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t)
	})
}

func TestEnvironmentPanel_ItemCount(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())
	ep.SetEnvironmentInfo("env-5", "", environment.StrategyWorkspace, createManyEnvironments(29))
	ep.SetFocused(true)
	ep.SetSelectedIndex(5)

	result := testutil.RenderComponent(t, ep, 80, 15)

	// Should show item count like "6 of 29" when focused
	result.AssertHasItemCount(t)
}

func TestEnvironmentPanel_UnfocusedShowsCurrent(t *testing.T) {
	ep := NewEnvironmentPanel(styles.DefaultStyles())
	ep.SetEnvironmentInfo("dev", "", environment.StrategyWorkspace, createTestEnvironments())
	ep.SetFocused(false)

	result := testutil.RenderComponent(t, ep, 80, 10)

	// Should show the current environment even when unfocused
	result.
		AssertNotEmpty(t).
		AssertContains(t, "dev")
}
