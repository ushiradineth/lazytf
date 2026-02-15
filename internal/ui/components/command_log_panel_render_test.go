package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func TestCommandLogPanel_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		clp := NewCommandLogPanel(styles.DefaultStyles())
		clp.SetLogText(testutil.SampleLogText)

		result := testutil.RenderComponent(t, clp, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t).
			AssertHasBorder(t)
	})
}

func TestCommandLogPanel_EmptyState(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())
	clp.SetSize(80, 20)

	result := testutil.RenderComponent(t, clp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertContains(t, "Tip:")
}

func TestCommandLogPanel_WithLogText(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())
	clp.SetLogText("Terraform will perform the following actions:\n  + resource will be created\n  - resource will be destroyed")

	result := testutil.RenderComponent(t, clp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertHasPanelID(t, "[4]")
}

func TestCommandLogPanel_WithDiagnostics(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())
	clp.SetDiagnostics(testutil.SampleDiagnostics)

	result := testutil.RenderComponent(t, clp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestCommandLogPanel_FocusState(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())
	clp.SetLogText(testutil.SampleLogText)

	// Both focused and unfocused should render valid output
	focusedResult := testutil.RenderWithFocus(t, clp, 80, 20, true)
	focusedResult.
		AssertNotEmpty(t).
		AssertHasBorder(t)

	unfocusedResult := testutil.RenderWithFocus(t, clp, 80, 20, false)
	unfocusedResult.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestCommandLogPanel_HeightRespected(t *testing.T) {
	for _, h := range []int{5, 10, 15, 20, 30} {
		t.Run(testutil.DimensionSet{Name: "height", Height: h}.String(), func(t *testing.T) {
			clp := NewCommandLogPanel(styles.DefaultStyles())
			clp.SetLogText(testutil.SampleLogText)

			result := testutil.RenderComponent(t, clp, 80, h)

			result.
				AssertHeight(t, h).
				AssertNoLineOverflow(t)
		})
	}
}

func TestCommandLogPanel_LongLogs(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())

	// Create a very long log
	var sb strings.Builder
	for i := range 100 {
		sb.WriteString("Log line ")
		sb.WriteString(testutil.IntToString(i))
		sb.WriteString(": This is a sample log entry with some additional text.\n")
	}
	clp.SetLogText(sb.String())

	result := testutil.RenderComponent(t, clp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertHeight(t, 20).
		AssertNoLineOverflow(t)
}

func TestCommandLogPanel_ScrollbarRendering(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())

	// Short log renders correctly
	clp.SetLogText("Short log")
	result := testutil.RenderComponent(t, clp, 40, 10)
	result.AssertNotEmpty(t).AssertHasBorder(t)

	// Long log renders correctly
	var longLogBuilder strings.Builder
	for i := range 50 {
		longLogBuilder.WriteString("Line ")
		longLogBuilder.WriteString(testutil.IntToString(i))
		longLogBuilder.WriteString("\n")
	}
	clp.SetLogText(longLogBuilder.String())
	result = testutil.RenderComponent(t, clp, 40, 10)
	result.AssertNotEmpty(t).AssertHasBorder(t).AssertHeight(t, 10)
}

func TestCommandLogPanel_MinimalDimensions(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())
	clp.SetLogText("Test log")

	// Very small dimensions should not panic
	result := testutil.RenderComponent(t, clp, 15, 5)
	result.AssertNotEmpty(t)
}

func TestCommandLogPanel_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		clp := NewCommandLogPanel(styles.DefaultStyles())
		clp.SetLogText(testutil.SampleLogText)

		result := testutil.RenderComponent(t, clp, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t)
	})
}

func TestCommandLogPanel_Visibility(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())
	clp.SetLogText("Test log")
	clp.SetSize(80, 20)

	// Visible by default
	result := testutil.RenderComponent(t, clp, 80, 20)
	result.AssertNotEmpty(t)

	// Hidden
	clp.SetVisible(false)
	view := clp.View()
	if view != "" {
		t.Errorf("Expected empty view when hidden, got: %s", view)
	}
}

func TestCommandLogPanel_SessionLogs(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())
	clp.AppendSessionLog("plan", "terraform plan", "Plan: 2 to add, 0 to change, 1 to destroy.")
	clp.AppendSessionLog("apply", "terraform apply", "Apply complete!")

	result := testutil.RenderComponent(t, clp, 80, 20)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestCommandLogPanel_DiagnosticSeverities(t *testing.T) {
	severities := []struct {
		name        string
		diagnostics []terraform.Diagnostic
	}{
		{
			name: "error",
			diagnostics: []terraform.Diagnostic{
				{Severity: "error", Summary: "Error occurred", Detail: "Something went wrong"},
			},
		},
		{
			name: "warning",
			diagnostics: []terraform.Diagnostic{
				{Severity: "warning", Summary: "Warning", Detail: "This might be a problem"},
			},
		},
	}

	for _, tc := range severities {
		t.Run(tc.name, func(t *testing.T) {
			clp := NewCommandLogPanel(styles.DefaultStyles())
			clp.SetDiagnostics(tc.diagnostics)

			result := testutil.RenderComponent(t, clp, 80, 20)

			result.
				AssertNotEmpty(t).
				AssertHasBorder(t)
		})
	}
}

func TestCommandLogPanel_LogTextRendering(t *testing.T) {
	clp := NewCommandLogPanel(styles.DefaultStyles())
	clp.SetLogText("Raw log output")
	result := testutil.RenderComponent(t, clp, 80, 20)
	result.AssertNotEmpty(t).AssertHasBorder(t)
}
