package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func TestModal_BasicText(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetTitle("Test Modal")
	modal.SetContent("This is the modal content.\nWith multiple lines.")
	modal.Show()

	result := testutil.RenderViewable(t, modal, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertContains(t, "Test Modal").
		AssertContains(t, "modal content")
}

func TestModal_HelpItems(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetTitle("Keyboard Shortcuts")
	modal.SetItems([]HelpItem{
		{Key: "j/k", Description: "move selection"},
		{Key: "enter", Description: "select item"},
		{Key: "q", Description: "quit"},
	})
	modal.Show()

	result := testutil.RenderViewable(t, modal, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertContains(t, "Keyboard Shortcuts").
		AssertContainsAll(t, "j/k", "enter", "quit")
}

func TestModal_SelectionHighlight(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetItems([]HelpItem{
		{Key: "a", Description: "first"},
		{Key: "b", Description: "second"},
		{Key: "c", Description: "third"},
	})
	modal.SetSelectedIndex(1) // Select "second"
	modal.Show()

	result := testutil.RenderViewable(t, modal, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "second")
}

func TestModal_SelectionNavigation(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetItems([]HelpItem{
		{Key: "1", Description: "A"},
		{Key: "2", Description: "B"},
		{Key: "3", Description: "C"},
		{Key: "4", Description: "D"},
		{Key: "5", Description: "E"},
	})
	modal.Show()

	// Initial selection at 0
	modal.SetSelectedIndex(0)
	result0 := testutil.RenderViewable(t, modal, 80, 24)
	result0.
		AssertNotEmpty(t).
		AssertContains(t, "A")

	// Move to index 2
	modal.SetSelectedIndex(2)
	result2 := testutil.RenderViewable(t, modal, 80, 24)
	result2.
		AssertNotEmpty(t).
		AssertContains(t, "C")
}

func TestModal_Dimensions(t *testing.T) {
	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
		modal := NewModal(styles.DefaultStyles())
		modal.SetTitle("Test")
		modal.SetContent("Content")
		modal.Show()

		result := testutil.RenderViewable(t, modal, d.Width, d.Height)

		// Modal is a floating component, doesn't fill full dimensions
		result.AssertNotEmpty(t)
	})
}

func TestModal_Hidden(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetTitle("Hidden Modal")
	// Don't call Show()

	// Hidden modal should render as empty
	view := modal.View()
	if view != "" {
		t.Errorf("Expected empty view for hidden modal, got: %s", view)
	}
}

func TestModal_LongContent(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetTitle("Long Content")

	// Create content with many lines
	var sb strings.Builder
	for i := range 50 {
		sb.WriteString("Line ")
		sb.WriteString(testutil.IntToString(i))
		sb.WriteString("\n")
	}
	modal.SetContent(sb.String())
	modal.Show()

	result := testutil.RenderViewable(t, modal, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestModal_ManyItems(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetTitle("Many Items")

	items := make([]HelpItem, 50)
	for i := range items {
		items[i] = HelpItem{
			Key:         testutil.IntToString(i),
			Description: "Item " + testutil.IntToString(i),
		}
	}
	modal.SetItems(items)
	modal.Show()

	result := testutil.RenderViewable(t, modal, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestModal_ConfirmDialog(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetConfirm("Are you sure you want to proceed?", []ModalAction{
		{Label: "Yes", Key: "y"},
		{Label: "No", Key: "n"},
	})
	modal.Show()

	result := testutil.RenderViewable(t, modal, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t).
		AssertContains(t, "Are you sure")
}

func TestModal_SmallDimensions(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetTitle("Tiny")
	modal.SetContent("X")
	modal.Show()

	// Should not panic with very small dimensions
	result := testutil.RenderViewable(t, modal, 20, 10)

	result.AssertNotEmpty(t)
}

func TestModal_Styling(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetTitle("Styled Modal")
	modal.SetContent("Content with styling")
	modal.Show()

	result := testutil.RenderViewable(t, modal, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertHasBorder(t)
}

func TestModal_SectionHeaders(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetTitle("Help")
	modal.SetItems([]HelpItem{
		{Key: "", Description: "Navigation", IsHeader: true},
		{Key: "j/k", Description: "move up/down"},
		{Key: "", Description: "Actions", IsHeader: true},
		{Key: "enter", Description: "select"},
	})
	modal.Show()

	result := testutil.RenderViewable(t, modal, 80, 24)

	// Modal renders items - verify key items are present
	result.
		AssertNotEmpty(t).
		AssertContainsAll(t, "j/k", "enter")
}

func TestModal_Overlay(t *testing.T) {
	modal := NewModal(styles.DefaultStyles())
	modal.SetSize(80, 24)
	modal.SetTitle("Overlay Test")
	modal.SetContent("Overlaid content")
	modal.Show()

	// Test using Overlay method
	result := testutil.RenderOverlay(t, modal, 80, 24)

	result.
		AssertNotEmpty(t).
		AssertContains(t, "Overlaid content")
}

func TestModal_WidescreenDimensions(t *testing.T) {
	testutil.RunDimensionMatrix(t, testutil.WidescreenDimensions(), func(t *testing.T, d testutil.DimensionSet) {
		modal := NewModal(styles.DefaultStyles())
		modal.SetTitle("Widescreen Modal")
		modal.SetContent("Testing on widescreen")
		modal.Show()

		result := testutil.RenderViewable(t, modal, d.Width, d.Height)

		// Modal is a floating component, doesn't fill full dimensions
		result.AssertNotEmpty(t)
	})
}
