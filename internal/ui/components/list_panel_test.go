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
