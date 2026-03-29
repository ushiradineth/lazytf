package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// testItem implements ListPanelItem for testing.
type testItem struct {
	text string
}

func (t testItem) Render(s *styles.Styles, _ int, selected bool) string {
	line := t.text
	if selected {
		line = s.SelectedLine.Render(line)
	}
	return line
}

func TestListPanel_Basic(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	// Test empty panel
	view := panel.View()
	if !strings.Contains(view, "[2]") {
		t.Error("Panel should show panel ID")
	}
	if !strings.Contains(view, "No items") {
		t.Error("Empty panel should show 'No items'")
	}
}

func TestListPanel_WithItems(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetTabs([]string{"Resources"})
	panel.SetSize(40, 10)

	items := []ListPanelItem{
		testItem{"item 1"},
		testItem{"item 2"},
		testItem{"item 3"},
	}
	panel.SetItems(items)

	view := panel.View()
	if !strings.Contains(view, "item 1") {
		t.Error("Panel should show first item")
	}
	if !strings.Contains(view, "1 of 3") {
		t.Error("Panel should show item count")
	}
}

func TestListPanel_Navigation(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	items := []ListPanelItem{
		testItem{"item 1"},
		testItem{"item 2"},
		testItem{"item 3"},
	}
	panel.SetItems(items)

	if panel.GetSelectedIndex() != 0 {
		t.Error("Initial selection should be 0")
	}

	panel.MoveDown()
	if panel.GetSelectedIndex() != 1 {
		t.Error("Selection should be 1 after MoveDown")
	}

	panel.MoveUp()
	if panel.GetSelectedIndex() != 0 {
		t.Error("Selection should be 0 after MoveUp")
	}

	panel.End()
	if panel.GetSelectedIndex() != 2 {
		t.Error("Selection should be 2 after End")
	}

	panel.Home()
	if panel.GetSelectedIndex() != 0 {
		t.Error("Selection should be 0 after Home")
	}
}

func TestListPanel_Tabs(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetTabs([]string{"Tab1", "Tab2"})
	panel.SetSize(40, 10)

	view := panel.View()
	if !strings.Contains(view, "Tab1") {
		t.Error("Panel should show first tab")
	}
	if !strings.Contains(view, "Tab2") {
		t.Error("Panel should show second tab")
	}

	if panel.GetActiveTab() != 0 {
		t.Error("Initial active tab should be 0")
	}

	panel.NextTab()
	if panel.GetActiveTab() != 1 {
		t.Error("Active tab should be 1 after NextTab")
	}

	panel.PrevTab()
	if panel.GetActiveTab() != 0 {
		t.Error("Active tab should be 0 after PrevTab")
	}
}

func TestListPanel_Scrollbar(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 5) // Only 3 content lines (5 - 2 for borders)

	// Create more items than can fit
	items := make([]ListPanelItem, 10)
	for i := range items {
		items[i] = testItem{text: "item"}
	}
	panel.SetItems(items)

	view := panel.View()
	// Scrollbar should appear (▐ character)
	if !strings.Contains(view, "▐") {
		t.Error("Panel should show scrollbar when items exceed height")
	}
}

func TestListPanel_FocusState(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	if panel.IsFocused() {
		t.Error("Panel should not be focused initially")
	}

	panel.SetFocused(true)
	if !panel.IsFocused() {
		t.Error("Panel should be focused after SetFocused(true)")
	}
}

func TestListPanel_SetActiveTab(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetTabs([]string{"Tab1", "Tab2", "Tab3"})
	panel.SetSize(40, 10)

	panel.SetActiveTab(1)
	if panel.GetActiveTab() != 1 {
		t.Errorf("Expected active tab 1, got %d", panel.GetActiveTab())
	}

	// Out of bounds should be ignored (keeps current value)
	panel.SetActiveTab(10)
	if panel.GetActiveTab() != 1 {
		t.Errorf("Expected active tab 1 (unchanged), got %d", panel.GetActiveTab())
	}

	panel.SetActiveTab(-1)
	if panel.GetActiveTab() != 1 {
		t.Errorf("Expected active tab 1 (unchanged), got %d", panel.GetActiveTab())
	}

	// Valid set to 2
	panel.SetActiveTab(2)
	if panel.GetActiveTab() != 2 {
		t.Errorf("Expected active tab 2, got %d", panel.GetActiveTab())
	}
}

func TestListPanel_PageUpDown(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 6) // Small height for testing

	// Create many items
	items := make([]ListPanelItem, 20)
	for i := range items {
		items[i] = testItem{text: "item"}
	}
	panel.SetItems(items)

	// Start at beginning
	panel.SetSelectedIndex(0)

	// Page down
	panel.PageDown()
	if panel.GetSelectedIndex() == 0 {
		t.Error("Selection should move after PageDown")
	}

	// Move to middle
	panel.SetSelectedIndex(10)

	// Page up
	oldIndex := panel.GetSelectedIndex()
	panel.PageUp()
	if panel.GetSelectedIndex() >= oldIndex {
		t.Error("Selection should decrease after PageUp")
	}
}

func TestListPanel_ItemCount(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	if panel.ItemCount() != 0 {
		t.Errorf("Expected 0 items initially, got %d", panel.ItemCount())
	}

	items := []ListPanelItem{
		testItem{"item 1"},
		testItem{"item 2"},
	}
	panel.SetItems(items)

	if panel.ItemCount() != 2 {
		t.Errorf("Expected 2 items, got %d", panel.ItemCount())
	}
}

func TestListPanel_RenderContentLines(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	items := []ListPanelItem{
		testItem{"item 1"},
		testItem{"item 2"},
		testItem{"item 3"},
	}
	panel.SetItems(items)
	panel.SetFocused(true)

	lines := panel.RenderContentLines(38, 8) // width and height
	if len(lines) == 0 {
		t.Error("Expected non-empty content lines")
	}
}

func TestListPanel_GetScrollInfo(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 5) // Small height

	// Create many items
	items := make([]ListPanelItem, 20)
	for i := range items {
		items[i] = testItem{text: "item"}
	}
	panel.SetItems(items)

	scrollPos, thumbSize, hasScrollbar := panel.GetScrollInfo(3) // height parameter
	if !hasScrollbar {
		t.Error("Expected scrollbar when many items")
	}
	_ = scrollPos // Just check it doesn't panic
	_ = thumbSize
}

func TestListPanel_SelectVisibleRow(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 8)
	panel.SetItems([]ListPanelItem{
		testItem{"item 1"},
		testItem{"item 2"},
		testItem{"item 3"},
	})

	if !panel.SelectVisibleRow(1) {
		t.Fatal("expected row selection to succeed")
	}
	if panel.GetSelectedIndex() != 1 {
		t.Fatalf("expected selected index 1, got %d", panel.GetSelectedIndex())
	}

	if panel.SelectVisibleRow(-1) {
		t.Fatal("expected negative row selection to fail")
	}
	if panel.SelectVisibleRow(100) {
		t.Fatal("expected out-of-range row selection to fail")
	}
}

func TestListPanel_NilStyles(t *testing.T) {
	panel := NewListPanel("[2]", nil)
	if panel == nil {
		t.Fatal("Expected non-nil panel with nil styles")
	}
}

func TestListPanel_SetTabsWithTabs(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetTabs([]string{"Tab1"})
	panel.SetSize(40, 10)

	// SetTabs with multiple tabs
	panel.SetTabs([]string{"A", "B", "C"})
	view := panel.View()
	if !strings.Contains(view, "A") || !strings.Contains(view, "B") || !strings.Contains(view, "C") {
		t.Error("Panel should show all tab names")
	}
}

func TestListPanel_MoveUpAtTop(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	items := []ListPanelItem{
		testItem{"item 1"},
	}
	panel.SetItems(items)
	panel.SetSelectedIndex(0)

	// MoveUp at top should return false
	if panel.MoveUp() {
		t.Error("MoveUp at top should return false")
	}
}

func TestListPanel_MoveDownAtBottom(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	items := []ListPanelItem{
		testItem{"item 1"},
	}
	panel.SetItems(items)
	panel.SetSelectedIndex(0)

	// MoveDown at bottom should return false
	if panel.MoveDown() {
		t.Error("MoveDown at bottom should return false")
	}
}

func TestListPanel_HomeEnd(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	items := []ListPanelItem{
		testItem{"item 1"},
		testItem{"item 2"},
		testItem{"item 3"},
	}
	panel.SetItems(items)
	panel.SetSelectedIndex(1)

	// Home should go to 0
	if !panel.Home() {
		t.Error("Home should return true when not at 0")
	}
	if panel.GetSelectedIndex() != 0 {
		t.Errorf("Expected index 0, got %d", panel.GetSelectedIndex())
	}

	// Home again should return false
	if panel.Home() {
		t.Error("Home should return false when already at 0")
	}

	// End should go to last
	if !panel.End() {
		t.Error("End should return true when not at last")
	}
	if panel.GetSelectedIndex() != 2 {
		t.Errorf("Expected index 2, got %d", panel.GetSelectedIndex())
	}

	// End again should return false
	if panel.End() {
		t.Error("End should return false when already at last")
	}
}

func TestListPanel_RenderContentLinesZeroSize(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	items := []ListPanelItem{
		testItem{"item 1"},
	}
	panel.SetItems(items)

	// Zero width
	lines := panel.RenderContentLines(0, 8)
	if lines != nil {
		t.Error("expected nil for zero width")
	}

	// Zero height
	lines = panel.RenderContentLines(38, 0)
	if lines != nil {
		t.Error("expected nil for zero height")
	}
}

func TestListPanel_RenderContentLinesNilStyles(t *testing.T) {
	panel := &ListPanel{}
	lines := panel.RenderContentLines(38, 8)
	if lines != nil {
		t.Error("expected nil for nil styles")
	}
}

func TestListPanel_PageUpDownEdges(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 10)

	items := []ListPanelItem{
		testItem{"item 1"},
	}
	panel.SetItems(items)
	panel.SetSelectedIndex(0)

	// PageUp when only 1 item
	panel.PageUp()
	if panel.GetSelectedIndex() != 0 {
		t.Errorf("expected index 0 after PageUp with single item, got %d", panel.GetSelectedIndex())
	}

	// PageDown when only 1 item
	panel.PageDown()
	if panel.GetSelectedIndex() != 0 {
		t.Errorf("expected index 0 after PageDown with single item, got %d", panel.GetSelectedIndex())
	}
}

func TestListPanel_ViewEmptyItems(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetTabs([]string{"Test"})
	panel.SetSize(40, 10)
	panel.SetItems(nil)

	view := panel.View()
	if !strings.Contains(view, "No items") {
		t.Error("expected 'No items' in view for empty panel")
	}
}

func TestListPanel_SetTabsResetsActiveTab(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetTabs([]string{"Tab1", "Tab2", "Tab3"})
	panel.SetActiveTab(2)

	// Now set fewer tabs - activeTab should reset to 0
	panel.SetTabs([]string{"Tab1"})
	if panel.GetActiveTab() != 0 {
		t.Errorf("Expected activeTab to reset to 0 when tabs reduced, got %d", panel.GetActiveTab())
	}
}

func TestListPanel_ViewZeroWidth(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(0, 10)

	view := panel.View()
	if view != "" {
		t.Error("expected empty view with zero width")
	}
}

func TestListPanel_ContentWidthMinimum(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(2, 10)

	// With width=2, content width should be at least 1
	w := panel.contentWidth()
	if w < 1 {
		t.Errorf("Expected contentWidth at least 1, got %d", w)
	}
}

func TestListPanel_ContentHeightMinimum(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetSize(40, 2)

	// With height=2, content height should be at least 1
	h := panel.contentHeight()
	if h < 1 {
		t.Errorf("Expected contentHeight at least 1, got %d", h)
	}
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{12, "12"},
		{123, "123"},
		{-1, "-1"},
		{-42, "-42"},
	}

	for _, tt := range tests {
		got := intToString(tt.input)
		if got != tt.want {
			t.Errorf("intToString(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListPanel_SetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)

	newStyles := styles.DefaultStyles()
	panel.SetStyles(newStyles)

	// Panel should not panic and styles should be updated
	if panel.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestListPanel_ViewWithLongFooter(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewListPanel("[2]", s)
	panel.SetTabs([]string{"Test"})
	panel.SetSize(40, 10)
	panel.SetFocused(true)

	// Many items to get a longer footer
	items := make([]ListPanelItem, 100)
	for i := range items {
		items[i] = testItem{"item"}
	}
	panel.SetItems(items)
	panel.SetSelectedIndex(50)

	view := panel.View()
	if !strings.Contains(view, "51 of 100") {
		t.Error("expected '51 of 100' in footer")
	}
}
