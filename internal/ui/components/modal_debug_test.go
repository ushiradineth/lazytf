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
