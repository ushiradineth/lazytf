package components

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestEnvironmentPanelRender(_ *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Set up test environments
	envs := []environment.Environment{
		{
			Name:      "dummy",
			Path:      "/Users/test/terraform/dummy",
			Strategy:  environment.StrategyFolder,
			IsCurrent: true,
		},
		{
			Name:      "basic",
			Path:      "/Users/test/fixtures/basic",
			Strategy:  environment.StrategyFolder,
			IsCurrent: false,
		},
	}

	panel.SetEnvironmentInfo("dummy", "/Users/test", environment.StrategyFolder, envs)
	panel.SetSize(50, 10) // Width 50, Height 10
	panel.SetFocused(true)

	// Select first item
	panel.selectedIndex = 0

	// Render and print
	output := panel.View()
	fmt.Println("=== Panel Output (width=50, height=10) ===")
	fmt.Println(output)
	fmt.Println("=== End Output ===")

	// Print line by line with markers
	lines := strings.Split(output, "\n")
	fmt.Printf("\n=== Line-by-line analysis (total %d lines) ===\n", len(lines))
	for i, line := range lines {
		// Show visible width
		visWidth := 0
		for _, r := range line {
			if r >= 32 { // printable
				visWidth++
			}
		}
		fmt.Printf("Line %d (len=%d): %q\n", i, len(line), line)
	}

	// Test individual item rendering
	fmt.Println("\n=== Testing renderItem directly ===")
	testWidth := 46 // content width (50 - 2 borders - 2 for scrollbar potential)

	item1 := envListItem{
		env:    envs[0],
		label:  "dummy",
		detail: "",
	}
	item1.env.IsCurrent = true

	item2 := envListItem{
		env:    envs[1],
		label:  "basic",
		detail: "",
	}

	rendered1 := panel.renderItem(item1, testWidth, true)  // selected
	rendered2 := panel.renderItem(item2, testWidth, false) // not selected

	fmt.Printf("Item 1 (selected, width=%d):\n", testWidth)
	fmt.Printf("  Raw: %q\n", rendered1)
	fmt.Printf("  Len: %d bytes\n", len(rendered1))

	fmt.Printf("Item 2 (not selected, width=%d):\n", testWidth)
	fmt.Printf("  Raw: %q\n", rendered2)
	fmt.Printf("  Len: %d bytes\n", len(rendered2))

	// Check if items exceed width
	fmt.Println("\n=== Width check ===")
	fmt.Printf("Expected content width: %d\n", testWidth)
	fmt.Printf("Item 1 visible width: %d (should be <= %d)\n", visibleWidth(rendered1), testWidth)
	fmt.Printf("Item 2 visible width: %d (should be <= %d)\n", visibleWidth(rendered2), testWidth)
}

func visibleWidth(s string) int {
	// Count visible characters (ignoring ANSI codes)
	inEscape := false
	width := 0
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		width++
	}
	return width
}

func TestEnvironmentPanelItemWidth(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	env := environment.Environment{
		Name:      "dummy",
		Path:      "/Users/test/terraform/dummy",
		Strategy:  environment.StrategyFolder,
		IsCurrent: true,
	}

	item := envListItem{
		env:   env,
		label: "dummy",
	}
	item.env.IsCurrent = true

	// Test various widths
	for _, width := range []int{20, 30, 40, 50} {
		selected := panel.renderItem(item, width, true)
		notSelected := panel.renderItem(item, width, false)

		selWidth := visibleWidth(selected)
		notSelWidth := visibleWidth(notSelected)

		fmt.Printf("Width %d: selected=%d, notSelected=%d\n", width, selWidth, notSelWidth)

		if selWidth > width {
			t.Errorf("Selected item exceeds width %d: got %d", width, selWidth)
		}
		if notSelWidth > width {
			t.Errorf("Non-selected item exceeds width %d: got %d", width, notSelWidth)
		}
	}
}

func TestEnvironmentPanelWithStyledOutput(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	env := environment.Environment{
		Name:      "dummy",
		Path:      "/Users/test/terraform/dummy",
		Strategy:  environment.StrategyFolder,
		IsCurrent: true,
	}

	item := envListItem{
		env:   env,
		label: "dummy",
	}
	item.env.IsCurrent = true

	width := 40

	// Render selected item
	selected := panel.renderItem(item, width, true)
	notSelected := panel.renderItem(item, width, false)

	fmt.Println("=== Styled output analysis ===")
	fmt.Printf("Width target: %d\n\n", width)

	fmt.Println("Selected item:")
	fmt.Printf("  Raw bytes: %d\n", len(selected))
	fmt.Printf("  Visible width: %d\n", visibleWidth(selected))
	fmt.Printf("  Contains ANSI: %v\n", strings.Contains(selected, "\x1b["))
	fmt.Printf("  Output: %s|\n", selected) // | shows where it ends

	fmt.Println("\nNot selected item:")
	fmt.Printf("  Raw bytes: %d\n", len(notSelected))
	fmt.Printf("  Visible width: %d\n", visibleWidth(notSelected))
	fmt.Printf("  Contains ANSI: %v\n", strings.Contains(notSelected, "\x1b["))
	fmt.Printf("  Output: %s|\n", notSelected) // | shows where it ends

	// Check the styled content
	if visibleWidth(selected) != width {
		t.Errorf("Selected visible width %d != target %d", visibleWidth(selected), width)
	}
	if visibleWidth(notSelected) != width {
		t.Errorf("Not selected visible width %d != target %d", visibleWidth(notSelected), width)
	}
}

func TestEnvironmentPanelFrameIntegration(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)
	frame := NewPanelFrame(s)

	// Simulate real conditions
	panelWidth := 50
	panelHeight := 6

	panel.SetSize(panelWidth, panelHeight)
	frame.SetSize(panelWidth, panelHeight)

	// Set up environments
	envs := []environment.Environment{
		{Name: "dummy", Path: "/test/terraform/dummy", Strategy: environment.StrategyFolder, IsCurrent: true},
		{Name: "basic", Path: "/test/fixtures/basic", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("dummy", "/test", environment.StrategyFolder, envs)
	panel.SetFocused(true)
	panel.selectedIndex = 0

	// Get the content width from frame
	frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		Tabs:          []string{"Folders"},
		Focused:       true,
		ShowScrollbar: false,
	})

	contentWidth := frame.ContentWidth()
	fmt.Printf("=== Frame integration test ===\n")
	fmt.Printf("Panel width: %d\n", panelWidth)
	fmt.Printf("Content width from frame: %d\n", contentWidth)

	// Manually render items at the content width
	item := envListItem{
		env:   envs[0],
		label: "dummy",
	}
	item.env.IsCurrent = true

	rendered := panel.renderItem(item, contentWidth, true)
	fmt.Printf("\nItem rendered at contentWidth=%d:\n", contentWidth)
	fmt.Printf("  visibleWidth: %d\n", visibleWidth(rendered))
	fmt.Printf("  lipgloss.Width: %d\n", lipgloss.Width(rendered))
	fmt.Printf("  runewidth: %d\n", runewidth.StringWidth(rendered))
	fmt.Printf("  Output: [%s]\n", rendered)

	// Check what padLine does to it
	padded := frame.padLine(rendered, contentWidth)
	fmt.Printf("\nAfter frame.padLine:\n")
	fmt.Printf("  visibleWidth: %d\n", visibleWidth(padded))
	fmt.Printf("  lipgloss.Width: %d\n", lipgloss.Width(padded))
	fmt.Printf("  Output: [%s]\n", padded)

	// Check if there's a mismatch
	if lipgloss.Width(rendered) != visibleWidth(rendered) {
		fmt.Printf("\n⚠️  WARNING: lipgloss.Width (%d) != visibleWidth (%d)\n",
			lipgloss.Width(rendered), visibleWidth(rendered))
	}

	if visibleWidth(rendered) > contentWidth {
		t.Errorf("Item width %d exceeds content width %d", visibleWidth(rendered), contentWidth)
	}
}

func TestPanelFramePadLine(_ *testing.T) {
	s := styles.DefaultStyles()
	frame := NewPanelFrame(s)

	// Test with plain text
	plain := "hello"
	padded := frame.padLine(plain, 10)
	fmt.Printf("Plain text padLine:\n")
	fmt.Printf("  Input: %q (len=%d)\n", plain, len(plain))
	fmt.Printf("  Output: %q (len=%d)\n", padded, len(padded))
	fmt.Printf("  Visible: %d\n", visibleWidth(padded))

	// Test with ANSI styled text
	styled := "\x1b[1m\x1b[48;5;240mhello\x1b[0m"
	paddedStyled := frame.padLine(styled, 10)
	fmt.Printf("\nStyled text padLine:\n")
	fmt.Printf("  Input visible width: %d\n", visibleWidth(styled))
	fmt.Printf("  lipgloss.Width: %d\n", lipgloss.Width(styled))
	fmt.Printf("  Output visible width: %d\n", visibleWidth(paddedStyled))
	fmt.Printf("  Output lipgloss.Width: %d\n", lipgloss.Width(paddedStyled))
}

func TestSelectedItemWithANSI(t *testing.T) {
	// Test the exact scenario that caused issues: styled selected line
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	env := environment.Environment{
		Name:      "dummy",
		Path:      "/Users/test/terraform/dummy",
		Strategy:  environment.StrategyFolder,
		IsCurrent: true,
	}

	item := envListItem{
		env:   env,
		label: "dummy",
	}
	item.env.IsCurrent = true

	width := 48 // typical content width

	// Render selected item
	selected := panel.renderItem(item, width, true)

	// Verify visible width matches target
	selectedVisible := visibleWidth(selected)
	if selectedVisible != width {
		t.Errorf("Selected item visible width = %d, want %d", selectedVisible, width)
	}

	// Verify lipgloss.Width also matches (important for frame padding)
	selectedLipgloss := lipgloss.Width(selected)
	if selectedLipgloss != width {
		t.Errorf("Selected item lipgloss.Width = %d, want %d", selectedLipgloss, width)
	}

	// Now simulate what PanelFrame does
	frame := NewPanelFrame(s)
	padded := frame.padLine(selected, width)

	// The padded line should still be exactly the right width
	paddedVisible := visibleWidth(padded)
	if paddedVisible != width {
		t.Errorf("After padLine, visible width = %d, want %d", paddedVisible, width)
	}

	fmt.Printf("\n=== Selected item with ANSI test ===\n")
	fmt.Printf("Width target: %d\n", width)
	fmt.Printf("Selected visible: %d\n", selectedVisible)
	fmt.Printf("Selected lipgloss: %d\n", selectedLipgloss)
	fmt.Printf("After padLine visible: %d\n", paddedVisible)
	fmt.Printf("Output: [%s]\n", padded)
}
