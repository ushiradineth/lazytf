package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestMainAreaViewWithNilStylesReturnsEmpty(t *testing.T) {
	m := &MainArea{}
	if got := m.View(); got != "" {
		t.Fatalf("expected empty view for nil styles, got %q", got)
	}
}

func TestMainAreaViewWithZeroHeightReturnsEmpty(t *testing.T) {
	m := &MainArea{styles: styles.DefaultStyles()}
	if got := m.View(); got != "" {
		t.Fatalf("expected empty view for zero height, got %q", got)
	}
}

func TestSetSelectedResourceDoesNotResetScrollForSameResource(t *testing.T) {
	m := NewMainArea(styles.DefaultStyles(), diff.NewEngine(), nil, nil)
	m.SetSize(80, 12)
	m.SetFocused(true)

	before := make(map[string]any)
	after := make(map[string]any)
	for i := range 40 {
		key := "field_" + string(rune('a'+(i%26))) + string(rune('A'+(i/26)))
		before[key] = "old"
		after[key] = "new"
	}

	resource := &terraform.ResourceChange{
		Address: "test_resource.example",
		Action:  terraform.ActionUpdate,
		Change: &terraform.Change{
			Before: before,
			After:  after,
		},
	}

	m.SetSelectedResource(resource)
	_ = m.View()

	handled, _ := m.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Fatal("expected down key to be handled in diff mode")
	}

	initialOffset, _, _ := m.diffViewer.GetScrollInfo()
	if initialOffset == 0 {
		t.Fatal("expected scroll offset to move after scrolling down")
	}

	m.SetSelectedResource(resource)
	afterOffset, _, _ := m.diffViewer.GetScrollInfo()
	if afterOffset != initialOffset {
		t.Fatalf("expected same-resource selection to preserve scroll offset, got %d want %d", afterOffset, initialOffset)
	}
}
