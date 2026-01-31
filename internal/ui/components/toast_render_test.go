package components

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func TestToast_AllLevels(t *testing.T) {
	levels := []struct {
		name  string
		level ToastLevel
	}{
		{"Info", ToastInfo},
		{"Success", ToastSuccess},
		{"Warning", ToastWarning},
		{"Error", ToastError},
	}

	for _, tc := range levels {
		t.Run(tc.name, func(t *testing.T) {
			toast := NewToast(styles.DefaultStyles())
			toast.SetSize(80, 24)
			toast.Show("Test message for "+tc.name, tc.level)

			result := testutil.RenderOverlay(t, toast, 80, 24)

			result.
				AssertNotEmpty(t).
				AssertHasBorder(t).
				AssertContains(t, "Test message")
		})
	}
}

func TestToast_AllPositions(t *testing.T) {
	positions := []struct {
		name     string
		position ToastPosition
	}{
		{"TopLeft", ToastTopLeft},
		{"TopRight", ToastTopRight},
		{"BottomLeft", ToastBottomLeft},
		{"BottomRight", ToastBottomRight},
	}

	for _, tc := range positions {
		t.Run(tc.name, func(t *testing.T) {
			toast := NewToast(styles.DefaultStyles())
			toast.SetSize(80, 24)
			toast.SetPosition(tc.position)
			toast.Show("Test message", ToastInfo)

			result := testutil.RenderOverlay(t, toast, 80, 24)

			result.
				AssertNotEmpty(t).
				AssertHeight(t, 24).
				AssertNoLineOverflow(t)
		})
	}
}

func TestToast_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		toast := NewToast(styles.DefaultStyles())
		toast.Show("Test toast message", ToastInfo)

		result := testutil.RenderOverlay(t, toast, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t)
	})
}

func TestToast_Hidden(t *testing.T) {
	toast := NewToast(styles.DefaultStyles())
	toast.SetSize(80, 24)
	// Don't show - should render base unchanged

	result := testutil.RenderOverlay(t, toast, 80, 24)

	// Hidden toast should fill dimensions with base content
	result.AssertHeight(t, 24)
}

func TestToast_LongMessage(t *testing.T) {
	toast := NewToast(styles.DefaultStyles())
	toast.SetSize(80, 24)
	toast.Show("This is a very long toast message that should be properly wrapped or truncated to fit within the toast dimensions without overflowing", ToastInfo)

	result := testutil.RenderOverlay(t, toast, 80, 24)

	result.AssertNoLineOverflow(t)
}

func TestToast_LevelColors(t *testing.T) {
	s := styles.DefaultStyles()

	// Each level should have different styling
	infoToast := NewToast(s)
	infoToast.SetSize(80, 24)
	infoToast.Show("Info", ToastInfo)
	infoResult := testutil.RenderOverlay(t, infoToast, 80, 24)

	errorToast := NewToast(s)
	errorToast.SetSize(80, 24)
	errorToast.Show("Error", ToastError)
	errorResult := testutil.RenderOverlay(t, errorToast, 80, 24)

	// Info and Error should have different styling
	infoResult.AssertStyleDifferentFrom(t, errorResult, "info vs error toast")
}

func TestToast_SuccessLevel(t *testing.T) {
	toast := NewToast(styles.DefaultStyles())
	toast.SetSize(80, 24)
	toast.ShowSuccess("Operation completed successfully")

	result := testutil.RenderOverlay(t, toast, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "Operation completed")
}

func TestToast_WarningLevel(t *testing.T) {
	toast := NewToast(styles.DefaultStyles())
	toast.SetSize(80, 24)
	toast.ShowWarning("This is a warning")

	result := testutil.RenderOverlay(t, toast, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "warning")
}

func TestToast_ErrorLevel(t *testing.T) {
	toast := NewToast(styles.DefaultStyles())
	toast.SetSize(80, 24)
	toast.ShowError("An error occurred")

	result := testutil.RenderOverlay(t, toast, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "error")
}

func TestToast_SmallDimensions(t *testing.T) {
	toast := NewToast(styles.DefaultStyles())
	toast.Show("Test", ToastInfo)

	// Should not panic with small dimensions
	result := testutil.RenderOverlay(t, toast, 30, 10)

	result.
		AssertHeight(t, 10).
		AssertNoLineOverflow(t)
}

func TestToast_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		toast := NewToast(styles.DefaultStyles())
		toast.Show("Widescreen toast", ToastInfo)

		result := testutil.RenderOverlay(t, toast, d.Width, d.Height)

		result.
			AssertHeight(t, d.Height).
			AssertNoLineOverflow(t)
	})
}
