package components

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestModalOverlayDebug(t *testing.T) {
	s := styles.DefaultStyles()

	// Create a base view with realistic dimensions (80x20)
	width := 80
	height := 20
	baseLines := make([]string, height)
	for i := range height {
		line := fmt.Sprintf("Line %2d: ", i+1)
		padding := strings.Repeat("=", width-len(line))
		baseLines[i] = line + padding
	}
	baseView := strings.Join(baseLines, "\n")

	fmt.Println("=== BASE VIEW ===")
	fmt.Println(baseView)
	fmt.Println()

	// Create modal
	modal := NewModal(s)
	modal.SetSize(width, height)
	modal.SetTitle("Keybinds")
	modal.SetContent("j/k: navigate\nenter: select\nq: quit")
	modal.Show()

	fmt.Println("=== MODAL VIEW (standalone) ===")
	modalView := modal.View()
	fmt.Println(modalView)
	fmt.Println()

	fmt.Println("=== MODAL OVERLAY ===")
	result := modal.Overlay(baseView)
	fmt.Println(result)
	fmt.Println()

	// Check dimensions
	fmt.Printf("Base view lines: %d\n", len(strings.Split(baseView, "\n")))
	fmt.Printf("Base view width: %d\n", lipgloss.Width(baseLines[0]))
	fmt.Printf("Modal width setting: %d\n", width)
	fmt.Printf("Modal height setting: %d\n", height)
	fmt.Printf("Modal box width: %d\n", lipgloss.Width(modalView))
	fmt.Printf("Modal box height: %d\n", lipgloss.Height(modalView))

	// Check if base content is visible around modal
	if !strings.Contains(result, "Line  1") {
		t.Error("Line 1 not visible in overlay - top should be visible")
	}
	if !strings.Contains(result, "Line 20") {
		t.Error("Line 20 not visible in overlay - bottom should be visible")
	}
	if !strings.Contains(result, "Keybinds") {
		t.Error("Modal title not visible in overlay")
	}

	// Verify modal is centered (check that content appears on both sides)
	resultLines := strings.Split(result, "\n")
	middleLine := resultLines[height/2]
	fmt.Printf("\nMiddle line analysis:\n")
	fmt.Printf("Middle line: %q\n", middleLine)
	fmt.Printf("Contains 'Line': %v\n", strings.Contains(middleLine, "Line"))
}

func TestModalScrolling(t *testing.T) {
	s := styles.DefaultStyles()

	// Create modal with lots of content
	modal := NewModal(s)
	modal.SetSize(80, 30) // 30 lines tall screen

	// Create 50 lines of content
	var contentLines []string
	for i := 1; i <= 50; i++ {
		contentLines = append(contentLines, fmt.Sprintf("Line %d: some content here", i))
	}
	modal.SetTitle("Test Modal")
	modal.SetContent(strings.Join(contentLines, "\n"))
	modal.Show()

	offset, maxOffset, viewport, total := modal.GetScrollInfo()
	fmt.Printf("Initial state:\n")
	fmt.Printf("  scrollOffset: %d\n", offset)
	fmt.Printf("  maxScrollOffset: %d\n", maxOffset)
	fmt.Printf("  viewportHeight: %d\n", viewport)
	fmt.Printf("  totalContentLines: %d\n", total)

	if maxOffset == 0 {
		t.Errorf("maxScrollOffset should be > 0 for 50 lines of content, got %d", maxOffset)
	}

	// Test scrolling down
	modal.ScrollDown()
	modal.ScrollDown()
	modal.ScrollDown()

	offset, _, _, _ = modal.GetScrollInfo()
	fmt.Printf("\nAfter 3x ScrollDown:\n")
	fmt.Printf("  scrollOffset: %d\n", offset)

	if offset != 3 {
		t.Errorf("After 3 ScrollDown calls, offset should be 3, got %d", offset)
	}

	// Test scrolling up
	modal.ScrollUp()
	offset, _, _, _ = modal.GetScrollInfo()
	fmt.Printf("\nAfter 1x ScrollUp:\n")
	fmt.Printf("  scrollOffset: %d\n", offset)

	if offset != 2 {
		t.Errorf("After ScrollUp, offset should be 2, got %d", offset)
	}
}

func TestModalConfirmMode(t *testing.T) {
	s := styles.DefaultStyles()

	// Create modal in confirm mode
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Confirm Apply")

	actions := []ModalAction{
		{Key: "y", Label: "Yes, apply"},
		{Key: "n", Label: "No, cancel"},
	}
	modal.SetConfirm("Plan summary:\n  + 3 to add\n  ~ 1 to change\n\nDo you want to apply these changes?", actions)
	modal.Show()

	// Verify confirm mode is active
	if !modal.IsConfirmMode() {
		t.Error("Expected modal to be in confirm mode")
	}

	// Render the modal
	view := modal.View()
	fmt.Println("=== CONFIRM MODAL ===")
	fmt.Println(view)

	// Check content is visible
	if !strings.Contains(view, "Confirm Apply") {
		t.Error("Title not visible in confirm modal")
	}
	if !strings.Contains(view, "Plan summary") {
		t.Error("Message not visible in confirm modal")
	}
	if !strings.Contains(view, "Yes, apply") {
		t.Error("Yes action not visible in confirm modal")
	}
	if !strings.Contains(view, "No, cancel") {
		t.Error("No action not visible in confirm modal")
	}

	// Verify actions are retrievable
	retrievedActions := modal.GetConfirmActions()
	if len(retrievedActions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(retrievedActions))
	}
	if retrievedActions[0].Key != "y" {
		t.Errorf("Expected first action key to be 'y', got '%s'", retrievedActions[0].Key)
	}
}

func TestModalOverlayWithANSI(t *testing.T) {
	s := styles.DefaultStyles()

	// Create a styled base view (simulating real TUI content with ANSI codes)
	width := 80
	height := 20

	// Create styled content using lipgloss (which produces ANSI codes)
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	baseLines := make([]string, height)
	for i := range height {
		// Mix styled and unstyled content
		lineNum := highlightStyle.Render(fmt.Sprintf("[%2d]", i+1))
		content := normalStyle.Render(fmt.Sprintf(" Resource %d ", i+1))
		padding := strings.Repeat("─", width-lipgloss.Width(lineNum)-lipgloss.Width(content))
		baseLines[i] = lineNum + content + padding
	}
	baseView := strings.Join(baseLines, "\n")

	fmt.Println("=== STYLED BASE VIEW ===")
	fmt.Println(baseView)
	fmt.Println()

	// Create modal
	modal := NewModal(s)
	modal.SetSize(width, height)
	modal.SetTitle("Help")
	modal.SetContent("Press ? to close")
	modal.Show()

	fmt.Println("=== MODAL OVERLAY ON STYLED VIEW ===")
	result := modal.Overlay(baseView)
	fmt.Println(result)
	fmt.Println()

	// Check that ANSI codes are NOT corrupted (no raw escape sequences visible)
	// If ANSI handling is broken, we'd see things like "2;78;78;78m" in the output
	if strings.Contains(result, ";78m") || strings.Contains(result, "78;78") {
		t.Error("ANSI escape codes are corrupted - raw sequences visible in output")
	}

	// Check modal content is visible
	if !strings.Contains(result, "Help") {
		t.Error("Modal title not visible in overlay")
	}

	// Check base content is preserved outside modal
	if !strings.Contains(result, "Resource 1") {
		t.Error("Base content should be visible outside modal area")
	}
	if !strings.Contains(result, "Resource 20") {
		t.Error("Last line should be visible below modal")
	}
}

func TestModalHideAndVisibility(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")
	modal.SetContent("Content")

	// Initially not visible
	if modal.IsVisible() {
		t.Error("Expected modal to be hidden initially")
	}

	// Show it
	modal.Show()
	if !modal.IsVisible() {
		t.Error("Expected modal to be visible after Show()")
	}

	// Hide it
	modal.Hide()
	if modal.IsVisible() {
		t.Error("Expected modal to be hidden after Hide()")
	}
}

func TestModalSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)

	newStyles := styles.DefaultStyles()
	modal.SetStyles(newStyles)

	if modal.styles != newStyles {
		t.Error("Expected styles to be updated")
	}
}

func TestModalItemModeSelection(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Select")

	items := []HelpItem{
		{Key: "1", Description: "Item 1"},
		{Key: "2", Description: "Item 2"},
		{Key: "3", Description: "Item 3"},
	}
	modal.SetItems(items)
	modal.Show()

	// Check initial selection
	if modal.GetSelectedIndex() != 0 {
		t.Errorf("Expected initial selection 0, got %d", modal.GetSelectedIndex())
	}

	// Set valid selection
	modal.SetSelectedIndex(1)
	if modal.GetSelectedIndex() != 1 {
		t.Errorf("Expected selection 1, got %d", modal.GetSelectedIndex())
	}

	// Invalid selection (out of bounds) should be ignored
	modal.SetSelectedIndex(100)
	if modal.GetSelectedIndex() != 1 {
		t.Errorf("Expected selection to remain 1 after invalid set, got %d", modal.GetSelectedIndex())
	}

	// Negative selection should be ignored
	modal.SetSelectedIndex(-1)
	if modal.GetSelectedIndex() != 1 {
		t.Errorf("Expected selection to remain 1 after negative set, got %d", modal.GetSelectedIndex())
	}
}

func TestModalMoveSelectionUp(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")

	items := []HelpItem{
		{Key: "1", Description: "Item 1"},
		{Key: "2", Description: "Item 2"},
		{Key: "3", Description: "Item 3"},
	}
	modal.SetItems(items)
	modal.Show()

	// Start at index 2
	modal.SetSelectedIndex(2)
	if modal.GetSelectedIndex() != 2 {
		t.Fatalf("Expected initial selection 2, got %d", modal.GetSelectedIndex())
	}

	// Move up
	modal.moveSelectionUp()
	if modal.GetSelectedIndex() != 1 {
		t.Errorf("Expected selection 1 after moveUp, got %d", modal.GetSelectedIndex())
	}

	// Move up again
	modal.moveSelectionUp()
	if modal.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection 0 after second moveUp, got %d", modal.GetSelectedIndex())
	}

	// Move up at top should stay at 0
	modal.moveSelectionUp()
	if modal.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection to remain 0, got %d", modal.GetSelectedIndex())
	}
}

func TestModalMoveSelectionDown(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")

	items := []HelpItem{
		{Key: "1", Description: "Item 1"},
		{Key: "2", Description: "Item 2"},
		{Key: "3", Description: "Item 3"},
	}
	modal.SetItems(items)
	modal.Show()

	// Start at index 0
	if modal.GetSelectedIndex() != 0 {
		t.Fatalf("Expected initial selection 0, got %d", modal.GetSelectedIndex())
	}

	// Move down
	modal.moveSelectionDown()
	if modal.GetSelectedIndex() != 1 {
		t.Errorf("Expected selection 1 after moveDown, got %d", modal.GetSelectedIndex())
	}

	// Move down again
	modal.moveSelectionDown()
	if modal.GetSelectedIndex() != 2 {
		t.Errorf("Expected selection 2 after second moveDown, got %d", modal.GetSelectedIndex())
	}

	// Move down at bottom should stay at 2
	modal.moveSelectionDown()
	if modal.GetSelectedIndex() != 2 {
		t.Errorf("Expected selection to remain 2, got %d", modal.GetSelectedIndex())
	}
}

func TestModalMoveSelectionSkipsHeaders(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")

	items := []HelpItem{
		{Key: "1", Description: "Item 1"},
		{Key: "", Description: "Header", IsHeader: true},
		{Key: "2", Description: "Item 2"},
	}
	modal.SetItems(items)
	modal.Show()

	// Start at index 0
	modal.SetSelectedIndex(0)

	// Move down should skip header and go to index 2
	modal.moveSelectionDown()
	if modal.GetSelectedIndex() != 2 {
		t.Errorf("Expected selection 2 (skipping header), got %d", modal.GetSelectedIndex())
	}

	// Move up should skip header and go to index 0
	modal.moveSelectionUp()
	if modal.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection 0 (skipping header), got %d", modal.GetSelectedIndex())
	}
}

func TestModalScrollUpContentMode(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")

	// Create many lines of content
	var lines []string
	for i := 1; i <= 50; i++ {
		lines = append(lines, fmt.Sprintf("Line %d", i))
	}
	modal.SetContent(strings.Join(lines, "\n"))
	modal.Show()

	// Scroll down first
	modal.ScrollDown()
	modal.ScrollDown()
	modal.ScrollDown()

	offset, _, _, _ := modal.GetScrollInfo()
	if offset != 3 {
		t.Errorf("Expected offset 3 after scroll down, got %d", offset)
	}

	// Now scroll up
	modal.ScrollUp()
	offset, _, _, _ = modal.GetScrollInfo()
	if offset != 2 {
		t.Errorf("Expected offset 2 after scroll up, got %d", offset)
	}

	// Scroll up to zero
	modal.ScrollUp()
	modal.ScrollUp()
	offset, _, _, _ = modal.GetScrollInfo()
	if offset != 0 {
		t.Errorf("Expected offset 0, got %d", offset)
	}

	// Scroll up at zero should stay at zero
	modal.ScrollUp()
	offset, _, _, _ = modal.GetScrollInfo()
	if offset != 0 {
		t.Errorf("Expected offset to remain 0, got %d", offset)
	}
}

func TestModalScrollDownItemMode(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")

	items := []HelpItem{
		{Key: "1", Description: "Item 1"},
		{Key: "2", Description: "Item 2"},
		{Key: "3", Description: "Item 3"},
	}
	modal.SetItems(items)
	modal.Show()

	// In item mode, ScrollDown should move selection
	modal.ScrollDown()
	if modal.GetSelectedIndex() != 1 {
		t.Errorf("Expected selection 1, got %d", modal.GetSelectedIndex())
	}

	// ScrollUp should also work in item mode
	modal.ScrollUp()
	if modal.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection 0, got %d", modal.GetSelectedIndex())
	}
}

func TestModalEnsureSelectionVisibleScrollsDown(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 30)
	modal.SetTitle("Test")

	// Create many items
	items := make([]HelpItem, 50)
	for i := range items {
		items[i] = HelpItem{Key: fmt.Sprintf("%d", i), Description: fmt.Sprintf("Item %d", i)}
	}
	modal.SetItems(items)
	modal.Show()

	// Jump to a low selection (should scroll down)
	modal.SetSelectedIndex(30)
	if modal.GetSelectedIndex() != 30 {
		t.Errorf("Expected selection 30, got %d", modal.GetSelectedIndex())
	}

	// Scroll offset should have been adjusted
	offset, _, _, _ := modal.GetScrollInfo()
	if offset < 10 {
		t.Errorf("Expected scroll offset to increase for far selection, got %d", offset)
	}
}

func TestModalMaxScrollOffsetNoScroll(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 50) // Large height

	// Only 3 lines of content (fits without scrolling)
	modal.SetContent("Line 1\nLine 2\nLine 3")
	modal.Show()

	_, maxOffset, _, _ := modal.GetScrollInfo()
	if maxOffset != 0 {
		t.Errorf("Expected max scroll offset 0 for small content, got %d", maxOffset)
	}
}

func TestModalOverlayNotVisible(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")
	modal.SetContent("Content")
	// Don't call Show()

	baseView := "This is the base view"
	result := modal.Overlay(baseView)

	if result != baseView {
		t.Error("Expected unchanged base view when modal is not visible")
	}
}

func TestModalOverlayNilStyles(t *testing.T) {
	modal := &Modal{}
	modal.visible = true

	baseView := "This is the base view"
	result := modal.Overlay(baseView)

	if result != baseView {
		t.Error("Expected unchanged base view when styles are nil")
	}
}

func TestModalOverlayZeroDimensions(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(0, 0)
	modal.SetTitle("Test")
	modal.SetContent("Content")
	modal.Show()

	baseView := "This is the base view"
	result := modal.Overlay(baseView)

	if result != baseView {
		t.Error("Expected unchanged base view when dimensions are zero")
	}
}

func TestModalOverlayShortBaseView(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")
	modal.SetContent("Content")
	modal.Show()

	// Base view with fewer lines than modal height
	baseView := "Line 1\nLine 2\nLine 3"
	result := modal.Overlay(baseView)

	// Should still work without crashing
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestModalSetSelectedIndexSkipsHeaders(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")

	items := []HelpItem{
		{Key: "1", Description: "Item 1"},
		{Key: "", Description: "Header", IsHeader: true},
		{Key: "2", Description: "Item 2"},
	}
	modal.SetItems(items)
	modal.Show()

	// Set index to 0 (valid)
	modal.SetSelectedIndex(0)
	if modal.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection 0, got %d", modal.GetSelectedIndex())
	}

	// Trying to set index to a header should be ignored
	modal.SetSelectedIndex(1)
	if modal.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection to remain 0 when trying to select header, got %d", modal.GetSelectedIndex())
	}
}

func TestModalViewNotVisible(t *testing.T) {
	s := styles.DefaultStyles()
	modal := NewModal(s)
	modal.SetSize(80, 20)
	modal.SetTitle("Test")
	modal.SetContent("Content")
	// Don't call Show()

	view := modal.View()
	if view != "" {
		t.Error("Expected empty view when modal is not visible")
	}
}

func TestModalViewNilStyles(t *testing.T) {
	modal := &Modal{}
	modal.visible = true

	view := modal.View()
	if view != "" {
		t.Error("Expected empty view when styles are nil")
	}
}
