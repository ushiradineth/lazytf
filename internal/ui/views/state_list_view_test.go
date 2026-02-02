package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestNewStateListView(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewStateListView(s)
	if view == nil {
		t.Fatal("expected non-nil view")
	}
	if view.styles != s {
		t.Error("styles not set correctly")
	}
}

func TestStateListViewSetSize(t *testing.T) {
	view := NewStateListView(styles.DefaultStyles())
	view.SetSize(80, 24)
	if view.width != 80 {
		t.Errorf("expected width 80, got %d", view.width)
	}
	if view.height != 24 {
		t.Errorf("expected height 24, got %d", view.height)
	}
}

func TestStateListViewSetResources(t *testing.T) {
	view := NewStateListView(styles.DefaultStyles())
	resources := []terraform.StateResource{
		{Address: "aws_instance.web"},
		{Address: "aws_s3_bucket.data"},
	}
	view.SetResources(resources)

	if len(view.resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(view.resources))
	}
	if view.selected != 0 {
		t.Errorf("expected selected to be 0, got %d", view.selected)
	}
	if view.offset != 0 {
		t.Errorf("expected offset to be 0, got %d", view.offset)
	}
}

func TestStateListViewGetSelected(t *testing.T) {
	view := NewStateListView(styles.DefaultStyles())

	// Empty resources
	if view.GetSelected() != nil {
		t.Error("expected nil for empty resources")
	}

	// With resources
	resources := []terraform.StateResource{
		{Address: "aws_instance.web"},
		{Address: "aws_s3_bucket.data"},
	}
	view.SetResources(resources)

	selected := view.GetSelected()
	if selected == nil {
		t.Fatal("expected non-nil selected")
	}
	if selected.Address != "aws_instance.web" {
		t.Errorf("expected first resource, got %s", selected.Address)
	}

	// Invalid selection
	view.selected = -1
	if view.GetSelected() != nil {
		t.Error("expected nil for negative selection")
	}

	view.selected = 100
	if view.GetSelected() != nil {
		t.Error("expected nil for out of bounds selection")
	}
}

func TestStateListViewMoveUp(t *testing.T) {
	view := NewStateListView(styles.DefaultStyles())
	view.SetSize(80, 10)
	resources := []terraform.StateResource{
		{Address: "resource1"},
		{Address: "resource2"},
		{Address: "resource3"},
	}
	view.SetResources(resources)

	// Move down first
	view.selected = 2
	view.offset = 1

	// Move up
	view.MoveUp()
	if view.selected != 1 {
		t.Errorf("expected selected 1, got %d", view.selected)
	}

	// Move up again (offset should adjust)
	view.MoveUp()
	if view.selected != 0 {
		t.Errorf("expected selected 0, got %d", view.selected)
	}
	if view.offset != 0 {
		t.Errorf("expected offset 0, got %d", view.offset)
	}

	// Move up at top (should stay)
	view.MoveUp()
	if view.selected != 0 {
		t.Errorf("expected selected to stay at 0, got %d", view.selected)
	}
}

func TestStateListViewMoveDown(t *testing.T) {
	view := NewStateListView(styles.DefaultStyles())
	view.SetSize(80, 5) // Small height to test scrolling
	resources := []terraform.StateResource{
		{Address: "resource1"},
		{Address: "resource2"},
		{Address: "resource3"},
		{Address: "resource4"},
		{Address: "resource5"},
	}
	view.SetResources(resources)

	// Move down
	view.MoveDown()
	if view.selected != 1 {
		t.Errorf("expected selected 1, got %d", view.selected)
	}

	// Move down multiple times
	view.MoveDown()
	view.MoveDown()
	view.MoveDown()
	if view.selected != 4 {
		t.Errorf("expected selected 4, got %d", view.selected)
	}

	// Move down at bottom (should stay)
	view.MoveDown()
	if view.selected != 4 {
		t.Errorf("expected selected to stay at 4, got %d", view.selected)
	}
}

func TestStateListViewVisibleRows(t *testing.T) {
	view := NewStateListView(styles.DefaultStyles())

	// Normal height
	view.SetSize(80, 10)
	if rows := view.visibleRows(); rows != 8 {
		t.Errorf("expected 8 visible rows (10-2), got %d", rows)
	}

	// Minimum height
	view.SetSize(80, 2)
	if rows := view.visibleRows(); rows != 1 {
		t.Errorf("expected 1 visible row minimum, got %d", rows)
	}

	// Very small height
	view.SetSize(80, 1)
	if rows := view.visibleRows(); rows != 1 {
		t.Errorf("expected 1 visible row, got %d", rows)
	}
}

func TestStateListViewUpdate(t *testing.T) {
	view := NewStateListView(styles.DefaultStyles())
	view.SetSize(80, 10)
	resources := []terraform.StateResource{
		{Address: "resource1"},
		{Address: "resource2"},
	}
	view.SetResources(resources)

	// Test 'j' key
	view.selected = 0
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if view.selected != 1 {
		t.Errorf("expected selected 1 after 'j', got %d", view.selected)
	}

	// Test 'k' key
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if view.selected != 0 {
		t.Errorf("expected selected 0 after 'k', got %d", view.selected)
	}

	// Test 'down' key
	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	if view.selected != 1 {
		t.Errorf("expected selected 1 after down, got %d", view.selected)
	}

	// Test 'up' key
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	if view.selected != 0 {
		t.Errorf("expected selected 0 after up, got %d", view.selected)
	}
}

func TestStateListViewView(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewStateListView(s)
	view.SetSize(60, 10)
	resources := []terraform.StateResource{
		{Address: "aws_instance.web"},
		{Address: "aws_s3_bucket.data"},
	}
	view.SetResources(resources)

	out := view.View()

	if !strings.Contains(out, "Terraform State") {
		t.Error("expected title in output")
	}
	if !strings.Contains(out, "aws_instance.web") {
		t.Error("expected first resource in output")
	}
	if !strings.Contains(out, "aws_s3_bucket.data") {
		t.Error("expected second resource in output")
	}
	if !strings.Contains(out, "navigate") {
		t.Error("expected footer with navigation help")
	}
}

func TestStateListViewViewNilStyles(t *testing.T) {
	view := &StateListView{}
	out := view.View()
	if out != "" {
		t.Error("expected empty output for nil styles")
	}
}

func TestStateListViewScrolling(t *testing.T) {
	view := NewStateListView(styles.DefaultStyles())
	view.SetSize(80, 5) // height 5 = 3 visible rows (5 - header - footer)

	resources := make([]terraform.StateResource, 10)
	for i := range 10 {
		resources[i] = terraform.StateResource{Address: "resource" + string(rune('0'+i))}
	}
	view.SetResources(resources)

	// Move down beyond visible area
	for range 5 {
		view.MoveDown()
	}

	if view.selected != 5 {
		t.Errorf("expected selected 5, got %d", view.selected)
	}

	// Offset should have scrolled
	if view.offset <= 0 {
		t.Errorf("expected offset > 0 for scrolled view, got %d", view.offset)
	}
}

func TestStateListViewSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewStateListView(s)

	newStyles := styles.DefaultStyles()
	view.SetStyles(newStyles)

	if view.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}
