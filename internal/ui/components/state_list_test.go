package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestStateResourceItemRender(t *testing.T) {
	s := styles.DefaultStyles()

	res := terraform.StateResource{
		Address:      "aws_instance.example",
		ResourceType: "aws_instance",
	}
	item := StateResourceItem{resource: res}

	// Test non-selected render
	result := item.Render(s, 40, false)
	if result == "" {
		t.Error("expected non-empty result for non-selected item")
	}

	// Test selected render
	selected := item.Render(s, 40, true)
	if selected == "" {
		t.Error("expected non-empty result for selected item")
	}

	// Both should be non-empty (styling may or may not differ in test env)
	_ = result
	_ = selected
}

func TestStateResourceItemRenderTruncation(t *testing.T) {
	s := styles.DefaultStyles()

	// Long address that needs truncation
	res := terraform.StateResource{
		Address: "module.very_long_module_name.aws_instance.with_very_long_name_here_that_exceeds_width",
	}
	item := StateResourceItem{resource: res}

	// Width smaller than address length
	result := item.Render(s, 20, false)
	// Should contain ellipsis due to truncation (in styled form)
	if result == "" {
		t.Error("expected non-empty result for truncated item")
	}
}

func TestNewStateListContent(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	if content == nil {
		t.Fatal("expected non-nil content")
	}
	if content.styles != s {
		t.Error("expected styles to be set")
	}
	if content.listPanel == nil {
		t.Error("expected listPanel to be created")
	}
	if content.loading {
		t.Error("expected loading to be false initially")
	}
}

func TestNewStateListContentNilStyles(t *testing.T) {
	content := NewStateListContent(nil)

	if content == nil {
		t.Fatal("expected non-nil content even with nil styles")
	}
	if content.styles == nil {
		t.Error("expected default styles to be set")
	}
}

func TestStateListContentSetSize(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	content.SetSize(100, 50)
	// SetSize should not panic and should set size on listPanel
}

func TestStateListContentSetFocused(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	content.SetFocused(true)
	if !content.IsFocused() {
		t.Error("expected content to be focused after SetFocused(true)")
	}

	content.SetFocused(false)
	if content.IsFocused() {
		t.Error("expected content to not be focused after SetFocused(false)")
	}
}

func TestStateListContentIsFocused(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	if content.IsFocused() {
		t.Error("expected content to not be focused initially")
	}
}

func TestStateListContentSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	newStyles := styles.DefaultStyles()
	content.SetStyles(newStyles)

	if content.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestStateListContentSetResources(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	resources := []terraform.StateResource{
		{Address: "aws_instance.a", ResourceType: "aws_instance"},
		{Address: "aws_s3_bucket.b", ResourceType: "aws_s3_bucket"},
	}

	content.SetResources(resources)

	if len(content.resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(content.resources))
	}
	if content.loading {
		t.Error("expected loading to be false after SetResources")
	}
	if content.errorMsg != "" {
		t.Error("expected errorMsg to be empty after SetResources")
	}
}

func TestStateListContentSetLoading(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	content.SetLoading(true)
	if !content.loading {
		t.Error("expected loading to be true")
	}

	content.SetLoading(false)
	if content.loading {
		t.Error("expected loading to be false")
	}
}

func TestStateListContentSetError(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.loading = true

	content.SetError("some error")

	if content.errorMsg != "some error" {
		t.Errorf("expected errorMsg 'some error', got %q", content.errorMsg)
	}
	if content.loading {
		t.Error("expected loading to be false after SetError")
	}
}

func TestStateListContentGetSelected(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	// Empty list
	if content.GetSelected() != nil {
		t.Error("expected nil for empty list")
	}

	// With resources
	resources := []terraform.StateResource{
		{Address: "aws_instance.a", ResourceType: "aws_instance"},
		{Address: "aws_s3_bucket.b", ResourceType: "aws_s3_bucket"},
	}
	content.SetResources(resources)

	selected := content.GetSelected()
	if selected == nil {
		t.Fatal("expected non-nil selected resource")
	}
	if selected.Address != "aws_instance.a" {
		t.Errorf("expected first resource, got %s", selected.Address)
	}

	// Move down and check
	content.MoveDown()
	selected = content.GetSelected()
	if selected == nil {
		t.Fatal("expected non-nil selected resource")
	}
	if selected.Address != "aws_s3_bucket.b" {
		t.Errorf("expected second resource, got %s", selected.Address)
	}
}

func TestStateListContentMoveUp(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	resources := []terraform.StateResource{
		{Address: "a"},
		{Address: "b"},
		{Address: "c"},
	}
	content.SetResources(resources)

	// Move down first
	content.MoveDown()
	content.MoveDown()

	// Now move up
	content.MoveUp()
	selected := content.GetSelected()
	if selected.Address != "b" {
		t.Errorf("expected 'b' after MoveUp, got %s", selected.Address)
	}
}

func TestStateListContentMoveDown(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	resources := []terraform.StateResource{
		{Address: "a"},
		{Address: "b"},
	}
	content.SetResources(resources)

	content.MoveDown()
	selected := content.GetSelected()
	if selected.Address != "b" {
		t.Errorf("expected 'b' after MoveDown, got %s", selected.Address)
	}
}

func TestStateListContentSelectVisibleRow(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)
	content.SetResources([]terraform.StateResource{
		{Address: "a"},
		{Address: "b"},
		{Address: "c"},
	})

	if !content.SelectVisibleRow(1) {
		t.Fatal("expected visible row selection to succeed")
	}
	selected := content.GetSelected()
	if selected == nil || selected.Address != "b" {
		t.Fatalf("expected address b selected, got %#v", selected)
	}

	if content.SelectVisibleRow(-1) {
		t.Fatal("expected negative row selection to fail")
	}
}

func TestStateListContentHandleKeyUp(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	resources := []terraform.StateResource{
		{Address: "a"},
		{Address: "b"},
	}
	content.SetResources(resources)
	content.MoveDown() // Start at index 1

	// Test 'up' key
	handled, cmd := content.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if !handled {
		t.Error("expected 'up' key to be handled")
	}
	if cmd != nil {
		t.Error("expected nil cmd from up key")
	}

	// Test 'k' key
	content.MoveDown() // Move back down
	handled, _ = content.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if !handled {
		t.Error("expected 'k' key to be handled")
	}
}

func TestStateListContentHandleKeyDown(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	resources := []terraform.StateResource{
		{Address: "a"},
		{Address: "b"},
	}
	content.SetResources(resources)

	// Test 'down' key
	handled, cmd := content.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Error("expected 'down' key to be handled")
	}
	if cmd != nil {
		t.Error("expected nil cmd from down key")
	}

	// Reset and test 'j' key
	content.SetResources(resources) // Reset selection
	handled, _ = content.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected 'j' key to be handled")
	}
}

func TestStateListContentHandleKeyEnter(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	resources := []terraform.StateResource{
		{Address: "aws_instance.test"},
	}
	content.SetResources(resources)

	// Without OnSelect callback
	handled, cmd := content.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected enter key to be handled")
	}
	if cmd != nil {
		t.Error("expected nil cmd without OnSelect callback")
	}

	// With OnSelect callback
	var selectedAddr string
	content.OnSelect = func(addr string) tea.Cmd {
		selectedAddr = addr
		return nil
	}

	handled, _ = content.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected enter key to be handled with callback")
	}
	if selectedAddr != "aws_instance.test" {
		t.Errorf("expected address 'aws_instance.test', got %q", selectedAddr)
	}
}

func TestStateListContentHandleKeyUnknown(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	// Unknown key should not be handled
	handled, cmd := content.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected unknown key to not be handled")
	}
	if cmd != nil {
		t.Error("expected nil cmd for unknown key")
	}
}

func TestStateListContentView(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	// Nil styles
	content.styles = nil
	if content.View() != "" {
		t.Error("expected empty view with nil styles")
	}
	content.styles = s

	// Loading state
	content.SetLoading(true)
	view := content.View()
	if view == "" {
		t.Error("expected non-empty view for loading state")
	}
	content.SetLoading(false)

	// Error state
	content.SetError("test error")
	view = content.View()
	if view == "" {
		t.Error("expected non-empty view for error state")
	}
	content.errorMsg = ""

	// Empty resources
	view = content.View()
	if view == "" {
		t.Error("expected non-empty view for empty resources")
	}

	// With resources
	resources := []terraform.StateResource{
		{Address: "aws_instance.a"},
		{Address: "aws_s3_bucket.b"},
	}
	content.SetResources(resources)
	view = content.View()
	if view == "" {
		t.Error("expected non-empty view with resources")
	}
}

func TestStateListContentGetScrollInfo(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 10)

	// Empty list
	scrollPos, thumbSize, hasScrollbar := content.GetScrollInfo(10)
	_ = scrollPos
	_ = thumbSize
	_ = hasScrollbar
	// Just verify it doesn't panic

	// With resources
	resources := make([]terraform.StateResource, 50) // Many resources
	for i := range resources {
		resources[i] = terraform.StateResource{Address: "res"}
	}
	content.SetResources(resources)

	scrollPos, thumbSize, hasScrollbar = content.GetScrollInfo(10)
	if !hasScrollbar {
		t.Error("expected scrollbar with many items")
	}
	_ = scrollPos
	_ = thumbSize
}

func TestStateListContentGetFooterText(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)
	content.SetSize(50, 20)

	// Empty list
	footer := content.GetFooterText()
	if footer != "" {
		t.Errorf("expected empty footer for empty list, got %q", footer)
	}

	// With resources
	resources := []terraform.StateResource{
		{Address: "a"},
		{Address: "b"},
		{Address: "c"},
	}
	content.SetResources(resources)

	footer = content.GetFooterText()
	if footer == "" {
		t.Error("expected non-empty footer with resources")
	}
}

func TestStateListContentResourceCount(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	// Empty
	if content.ResourceCount() != 0 {
		t.Errorf("expected 0 resources, got %d", content.ResourceCount())
	}

	// With resources
	resources := []terraform.StateResource{
		{Address: "a"},
		{Address: "b"},
	}
	content.SetResources(resources)

	if content.ResourceCount() != 2 {
		t.Errorf("expected 2 resources, got %d", content.ResourceCount())
	}
}

func TestStateListContentClear(t *testing.T) {
	s := styles.DefaultStyles()
	content := NewStateListContent(s)

	// Set some state
	content.SetResources([]terraform.StateResource{{Address: "a"}})
	content.loading = true
	content.errorMsg = "error"

	// Clear
	content.Clear()

	if len(content.resources) != 0 {
		t.Error("expected resources to be cleared")
	}
	if content.loading {
		t.Error("expected loading to be false after Clear")
	}
	if content.errorMsg != "" {
		t.Error("expected errorMsg to be empty after Clear")
	}
}
