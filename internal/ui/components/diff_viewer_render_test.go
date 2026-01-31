package components

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func newTestDiffViewer() *DiffViewer {
	return NewDiffViewer(styles.DefaultStyles(), testutil.NewTestDiffEngine())
}

func TestDiffViewer_CreateAction(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	resource := testutil.ResourceWithAction(terraform.ActionCreate)
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "aws_instance")
}

func TestDiffViewer_UpdateAction(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	resource := testutil.ResourceWithAction(terraform.ActionUpdate)
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "aws_instance")
}

func TestDiffViewer_DeleteAction(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	resource := testutil.ResourceWithAction(terraform.ActionDelete)
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "aws_instance")
}

func TestDiffViewer_ReplaceAction(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	resource := testutil.ResourceWithAction(terraform.ActionReplace)
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "aws_instance")
}

func TestDiffViewer_ComplexChange(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	resource := testutil.ResourceWithComplexChange()
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 80, 24)

	result.AssertNotEmpty(t)
}

func TestDiffViewer_MultilineChange(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	resource := testutil.ResourceWithMultilineChange()
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertNoLineOverflow(t)
}

func TestDiffViewer_LongAddress(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	resource := testutil.LongResourceAddress()
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertNoLineOverflow(t)
}

func TestDiffViewer_NilResource(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	result := testutil.RenderCapture(t, func() string {
		return dv.View(nil)
	}, 80, 24)

	// Should handle nil gracefully
	result.AssertNoLineOverflow(t)
}

func TestDiffViewer_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		dv := newTestDiffViewer()

		resource := testutil.ResourceWithAction(terraform.ActionUpdate)
		dv.SetSize(d.Width, d.Height)
		result := testutil.RenderCapture(t, func() string {
			return dv.View(&resource)
		}, d.Width, d.Height)

		result.AssertNoLineOverflow(t)
	})
}

func TestDiffViewer_SmallDimensions(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(30, 10)

	resource := testutil.ResourceWithAction(terraform.ActionUpdate)
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 30, 10)

	// Should not panic
	result.AssertNoLineOverflow(t)
}

func TestDiffViewer_NoChange(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	// Resource with no changes (no-op)
	resource := terraform.ResourceChange{
		Address:      "aws_instance.noop",
		ResourceType: "aws_instance",
		ResourceName: "noop",
		Action:       terraform.ActionNoOp,
	}
	result := testutil.RenderCapture(t, func() string {
		return dv.View(&resource)
	}, 80, 24)

	result.AssertNoLineOverflow(t)
}

func TestDiffViewer_AllActionTypes(t *testing.T) {
	dv := newTestDiffViewer()
	dv.SetSize(80, 24)

	for _, action := range []terraform.ActionType{
		terraform.ActionCreate,
		terraform.ActionUpdate,
		terraform.ActionDelete,
		terraform.ActionReplace,
	} {
		t.Run(string(action), func(t *testing.T) {
			resource := testutil.ResourceWithAction(action)
			result := testutil.RenderCapture(t, func() string {
				return dv.View(&resource)
			}, 80, 24)

			// Each action should render without error
			result.AssertNotEmpty(t)
		})
	}
}

func TestDiffViewer_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		dv := newTestDiffViewer()
		dv.SetSize(d.Width, d.Height)

		resource := testutil.ResourceWithComplexChange()
		result := testutil.RenderCapture(t, func() string {
			return dv.View(&resource)
		}, d.Width, d.Height)

		result.AssertNoLineOverflow(t)
	})
}
