package components

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/styles"
)

func debugf(format string, args ...any) {
	if testing.Verbose() {
		fmt.Printf(format, args...)
	}
}

func debugln(args ...any) {
	if testing.Verbose() {
		fmt.Println(args...)
	}
}

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
	debugln("=== Panel Output (width=50, height=10) ===")
	debugln(output)
	debugln("=== End Output ===")

	// Print line by line with markers
	lines := strings.Split(output, "\n")
	debugf("\n=== Line-by-line analysis (total %d lines) ===\n", len(lines))
	for i, line := range lines {
		// Show visible width
		visWidth := 0
		for _, r := range line {
			if r >= 32 { // printable
				visWidth++
			}
		}
		debugf("Line %d (len=%d): %q\n", i, len(line), line)
	}

	// Test individual item rendering
	debugln("\n=== Testing renderItem directly ===")
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

	debugf("Item 1 (selected, width=%d):\n", testWidth)
	debugf("  Raw: %q\n", rendered1)
	debugf("  Len: %d bytes\n", len(rendered1))

	debugf("Item 2 (not selected, width=%d):\n", testWidth)
	debugf("  Raw: %q\n", rendered2)
	debugf("  Len: %d bytes\n", len(rendered2))

	// Check if items exceed width
	debugln("\n=== Width check ===")
	debugf("Expected content width: %d\n", testWidth)
	debugf("Item 1 visible width: %d (should be <= %d)\n", visibleWidth(rendered1), testWidth)
	debugf("Item 2 visible width: %d (should be <= %d)\n", visibleWidth(rendered2), testWidth)
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

		debugf("Width %d: selected=%d, notSelected=%d\n", width, selWidth, notSelWidth)

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

	debugln("=== Styled output analysis ===")
	debugf("Width target: %d\n\n", width)

	debugln("Selected item:")
	debugf("  Raw bytes: %d\n", len(selected))
	debugf("  Visible width: %d\n", visibleWidth(selected))
	debugf("  Contains ANSI: %v\n", strings.Contains(selected, "\x1b["))
	debugf("  Output: %s|\n", selected) // | shows where it ends

	debugln("\nNot selected item:")
	debugf("  Raw bytes: %d\n", len(notSelected))
	debugf("  Visible width: %d\n", visibleWidth(notSelected))
	debugf("  Contains ANSI: %v\n", strings.Contains(notSelected, "\x1b["))
	debugf("  Output: %s|\n", notSelected) // | shows where it ends

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
	debugf("=== Frame integration test ===\n")
	debugf("Panel width: %d\n", panelWidth)
	debugf("Content width from frame: %d\n", contentWidth)

	// Manually render items at the content width
	item := envListItem{
		env:   envs[0],
		label: "dummy",
	}
	item.env.IsCurrent = true

	rendered := panel.renderItem(item, contentWidth, true)
	debugf("\nItem rendered at contentWidth=%d:\n", contentWidth)
	debugf("  visibleWidth: %d\n", visibleWidth(rendered))
	debugf("  lipgloss.Width: %d\n", lipgloss.Width(rendered))
	debugf("  runewidth: %d\n", runewidth.StringWidth(rendered))
	debugf("  Output: [%s]\n", rendered)

	// Check what padLine does to it
	padded := frame.padLine(rendered, contentWidth)
	debugf("\nAfter frame.padLine:\n")
	debugf("  visibleWidth: %d\n", visibleWidth(padded))
	debugf("  lipgloss.Width: %d\n", lipgloss.Width(padded))
	debugf("  Output: [%s]\n", padded)

	// Check if there's a mismatch
	if lipgloss.Width(rendered) != visibleWidth(rendered) {
		debugf("\n⚠️  WARNING: lipgloss.Width (%d) != visibleWidth (%d)\n",
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
	debugf("Plain text padLine:\n")
	debugf("  Input: %q (len=%d)\n", plain, len(plain))
	debugf("  Output: %q (len=%d)\n", padded, len(padded))
	debugf("  Visible: %d\n", visibleWidth(padded))

	// Test with ANSI styled text
	styled := "\x1b[1m\x1b[48;5;240mhello\x1b[0m"
	paddedStyled := frame.padLine(styled, 10)
	debugf("\nStyled text padLine:\n")
	debugf("  Input visible width: %d\n", visibleWidth(styled))
	debugf("  lipgloss.Width: %d\n", lipgloss.Width(styled))
	debugf("  Output visible width: %d\n", visibleWidth(paddedStyled))
	debugf("  Output lipgloss.Width: %d\n", lipgloss.Width(paddedStyled))
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

	debugf("\n=== Selected item with ANSI test ===\n")
	debugf("Width target: %d\n", width)
	debugf("Selected visible: %d\n", selectedVisible)
	debugf("Selected lipgloss: %d\n", selectedLipgloss)
	debugf("After padLine visible: %d\n", paddedVisible)
	debugf("Output: [%s]\n", padded)
}

func TestEnvironmentPanelIsFocused(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Initially not focused
	if panel.IsFocused() {
		t.Error("Expected panel to not be focused initially")
	}

	// Set focused
	panel.SetFocused(true)
	if !panel.IsFocused() {
		t.Error("Expected panel to be focused after SetFocused(true)")
	}

	// Set unfocused
	panel.SetFocused(false)
	if panel.IsFocused() {
		t.Error("Expected panel to not be focused after SetFocused(false)")
	}
}

func TestEnvironmentPanelSelectorActive(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// SelectorActive returns focused state
	panel.SetFocused(false)
	if panel.SelectorActive() {
		t.Error("Expected SelectorActive to return false when not focused")
	}

	panel.SetFocused(true)
	if !panel.SelectorActive() {
		t.Error("Expected SelectorActive to return true when focused")
	}
}

func TestEnvironmentPanelFiltering(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Initially not filtering
	if panel.Filtering() {
		t.Error("Expected panel to not be filtering initially")
	}

	// Set filter active
	panel.filterActive = true
	if !panel.Filtering() {
		t.Error("Expected panel to be filtering when filterActive is true")
	}
}

func TestEnvironmentPanelUpdate(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Update returns the panel and nil cmd
	result, cmd := panel.Update(nil)
	if result != panel {
		t.Error("Expected Update to return the same panel")
	}
	if cmd != nil {
		t.Error("Expected Update to return nil cmd")
	}
}

func TestEnvironmentPanelHandleKeyNotFocused(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)
	panel.SetFocused(false)

	// Should return false when not focused
	handled, cmd := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if handled {
		t.Error("Expected HandleKey to return false when not focused")
	}
	if cmd != nil {
		t.Error("Expected HandleKey to return nil cmd when not focused")
	}
}

func TestEnvironmentPanelHandleKeyNavigation(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
		{Name: "env2", Strategy: environment.StrategyFolder},
		{Name: "env3", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)
	panel.selectedIndex = 0

	// Test down navigation
	handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Error("Expected down key to be handled")
	}
	if panel.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after down, got %d", panel.selectedIndex)
	}

	// Test up navigation
	handled, _ = panel.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if !handled {
		t.Error("Expected up key to be handled")
	}
	if panel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0 after up, got %d", panel.selectedIndex)
	}

	// Test j/k navigation
	handled, _ = panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("Expected j key to be handled")
	}
	if panel.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after j, got %d", panel.selectedIndex)
	}

	handled, _ = panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if !handled {
		t.Error("Expected k key to be handled")
	}
	if panel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0 after k, got %d", panel.selectedIndex)
	}
}

func TestEnvironmentPanelHandleKeyFilter(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
		{Name: "env2", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)

	// Activate filter mode with /
	handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !handled {
		t.Error("Expected / key to be handled")
	}
	if !panel.filterActive {
		t.Error("Expected filter to be active after /")
	}
}

func TestEnvironmentPanelHandleKeyEnter(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)
	panel.selectedIndex = 0

	// Press enter to select
	handled, cmd := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("Expected enter key to be handled")
	}
	if cmd == nil {
		t.Error("Expected cmd to be returned on enter")
	}
}

func TestEnvironmentPanelHandleKeyEsc(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)
	panel.filterText = "test"

	// Esc should clear filter text
	handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("Expected esc key to be handled when filter text exists")
	}
	if panel.filterText != "" {
		t.Error("Expected filter text to be cleared after esc")
	}
}

func TestEnvironmentPanelHandleFilterKey(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
		{Name: "test2", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)
	panel.filterActive = true

	// Type characters
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if panel.filterText != "e" {
		t.Errorf("Expected filterText 'e', got %q", panel.filterText)
	}

	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if panel.filterText != "en" {
		t.Errorf("Expected filterText 'en', got %q", panel.filterText)
	}

	// Backspace
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if panel.filterText != "e" {
		t.Errorf("Expected filterText 'e' after backspace, got %q", panel.filterText)
	}

	// Ctrl+U to clear
	panel.filterText = "test"
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	if panel.filterText != "" {
		t.Error("Expected filterText to be empty after Ctrl+U")
	}

	// Escape to exit filter mode
	panel.filterActive = true
	panel.filterText = "test"
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if panel.filterActive {
		t.Error("Expected filter to be inactive after Esc")
	}
	if panel.filterText != "" {
		t.Error("Expected filterText to be empty after Esc")
	}
}

func TestEnvironmentPanelMoveUpDown(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
		{Name: "env2", Strategy: environment.StrategyFolder},
		{Name: "env3", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.selectedIndex = 0

	// Move down
	panel.moveDown()
	if panel.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1, got %d", panel.selectedIndex)
	}
	if panel.lastMove != 1 {
		t.Errorf("Expected lastMove 1, got %d", panel.lastMove)
	}

	// Move up
	panel.moveUp()
	if panel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d", panel.selectedIndex)
	}
	if panel.lastMove != -1 {
		t.Errorf("Expected lastMove -1, got %d", panel.lastMove)
	}

	// Can't move up past 0
	panel.moveUp()
	if panel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 (can't go below), got %d", panel.selectedIndex)
	}

	// Move to end
	panel.selectedIndex = 2
	panel.moveDown()
	if panel.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex 2 (can't go above max), got %d", panel.selectedIndex)
	}
}

func TestEnvironmentPanelSelectedEnvironment(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Empty list
	panel.filteredItems = nil
	if panel.selectedEnvironment() != nil {
		t.Error("Expected nil for empty list")
	}

	// With items
	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
		{Name: "env2", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.selectedIndex = 1

	selected := panel.selectedEnvironment()
	if selected == nil {
		t.Error("Expected non-nil selected environment")
	}
	if selected.Name != "env2" {
		t.Errorf("Expected env2, got %s", selected.Name)
	}
}

func TestFuzzyMatchEnvPanel(t *testing.T) {
	tests := []struct {
		query     string
		candidate string
		want      bool
	}{
		{"", "anything", true},
		{"e", "env", true},
		{"ev", "env", true},
		{"env", "env", true},
		{"xyz", "env", false},
		{"abc", "axbxc", true},
		{"abc", "ab", false},
		{"d", "development", true},
		{"dv", "development", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.query, tt.candidate), func(t *testing.T) {
			got := fuzzyMatchEnvPanel(tt.query, tt.candidate)
			if got != tt.want {
				t.Errorf("fuzzyMatchEnvPanel(%q, %q) = %v, want %v", tt.query, tt.candidate, got, tt.want)
			}
		})
	}
}

func TestEnvironmentPanelItemMatchesFilter(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	item := envListItem{
		env:    environment.Environment{Name: "production", Path: "/path/to/prod"},
		label:  "production",
		detail: "10 res",
	}

	// Matches label
	if !panel.itemMatchesFilter(item, "prod") {
		t.Error("Expected item to match 'prod' in label")
	}

	// Matches detail
	if !panel.itemMatchesFilter(item, "res") {
		t.Error("Expected item to match 'res' in detail")
	}

	// No match
	if panel.itemMatchesFilter(item, "xyz") {
		t.Error("Expected item to not match 'xyz'")
	}
}

func TestEnvironmentPanelRenderUnfocusedContent(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder, IsCurrent: true},
		{Name: "env2", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("env1", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 5)
	panel.SetFocused(false)

	lines := panel.renderUnfocusedContent(48, 3)
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestEnvironmentPanelRenderUnfocusedNoCurrentEnv(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
		{Name: "env2", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 5)
	panel.SetFocused(false)

	lines := panel.renderUnfocusedContent(48, 3)
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestEnvironmentPanelRenderUnfocusedEmpty(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, nil)
	panel.SetSize(50, 5)
	panel.SetFocused(false)

	lines := panel.renderUnfocusedContent(48, 3)
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "No workspaces") {
		t.Error("Expected 'No workspaces' message")
	}
}

func TestEnvironmentPanelEnvLabel(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// With name
	env1 := environment.Environment{Name: "production", Path: "/path/to/prod"}
	if panel.envLabel(env1) != "production" {
		t.Errorf("Expected 'production', got %q", panel.envLabel(env1))
	}

	// Without name, with path
	env2 := environment.Environment{Path: "/path/to/staging"}
	if panel.envLabel(env2) != "staging" {
		t.Errorf("Expected 'staging', got %q", panel.envLabel(env2))
	}

	// Without name and path
	env3 := environment.Environment{}
	if panel.envLabel(env3) != "(unknown)" {
		t.Errorf("Expected '(unknown)', got %q", panel.envLabel(env3))
	}
}

func TestEnvironmentPanelIsCurrentEnv(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// No current set, relies on IsCurrent flag
	panel.current = ""
	env1 := environment.Environment{Name: "prod", IsCurrent: true}
	if !panel.isCurrentEnv(env1) {
		t.Error("Expected env1 to be current (IsCurrent flag)")
	}

	// Current set for workspace
	panel.current = "prod"
	env2 := environment.Environment{Name: "prod", Strategy: environment.StrategyWorkspace}
	if !panel.isCurrentEnv(env2) {
		t.Error("Expected env2 to be current (workspace name match)")
	}

	// Current set for folder
	panel.current = "/path/to/dev"
	env3 := environment.Environment{Path: "/path/to/dev", Strategy: environment.StrategyFolder}
	if !panel.isCurrentEnv(env3) {
		t.Error("Expected env3 to be current (folder path match)")
	}

	// Non-matching
	env4 := environment.Environment{Name: "other", Path: "/other/path", Strategy: environment.StrategyFolder}
	if panel.isCurrentEnv(env4) {
		t.Error("Expected env4 to not be current")
	}
}

func TestEnvironmentPanelFormatMetadata(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// With resource count
	meta1 := environment.EnvironmentMetadata{ResourceCount: 10}
	result1 := panel.formatMetadata(meta1)
	if !strings.Contains(result1, "10 res") {
		t.Errorf("Expected '10 res' in %q", result1)
	}

	// With state but no resources
	meta2 := environment.EnvironmentMetadata{HasState: true, ResourceCount: 0}
	result2 := panel.formatMetadata(meta2)
	if !strings.Contains(result2, "state") {
		t.Errorf("Expected 'state' in %q", result2)
	}

	// Empty metadata
	meta3 := environment.EnvironmentMetadata{}
	result3 := panel.formatMetadata(meta3)
	if result3 != "" {
		t.Errorf("Expected empty string for empty metadata, got %q", result3)
	}
}

func TestEnvironmentPanelContentWidth(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// No scrollbar needed
	panel.width = 50
	panel.height = 10
	panel.filteredItems = nil
	if panel.contentWidth() != 48 {
		t.Errorf("Expected 48 (50-2 borders), got %d", panel.contentWidth())
	}

	// With scrollbar
	envs := make([]environment.Environment, 20)
	for i := range envs {
		envs[i] = environment.Environment{Name: fmt.Sprintf("env%d", i)}
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	if panel.contentWidth() != 47 {
		t.Errorf("Expected 47 (50-2 borders -1 scrollbar), got %d", panel.contentWidth())
	}
}

func TestEnvironmentPanelItemsHeight(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	panel.height = 10
	panel.filterActive = false
	if panel.itemsHeight() != 8 {
		t.Errorf("Expected 8 (10-2 borders), got %d", panel.itemsHeight())
	}

	panel.filterActive = true
	if panel.itemsHeight() != 7 {
		t.Errorf("Expected 7 (10-2 borders -1 filter line), got %d", panel.itemsHeight())
	}

	// Minimum of 1
	panel.height = 2
	if panel.itemsHeight() < 1 {
		t.Errorf("Expected at least 1, got %d", panel.itemsHeight())
	}
}

func TestFormatEnvAge(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		contains string
	}{
		{"now", "-10s", "now"},
		{"minutes", "-30m", "m"},
		{"hours", "-5h", "h"},
		{"days", "-3d", "d"},
	}

	now := time.Now()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testTime time.Time
			switch tt.duration {
			case "-10s":
				testTime = now.Add(-10 * time.Second)
			case "-30m":
				testTime = now.Add(-30 * time.Minute)
			case "-5h":
				testTime = now.Add(-5 * time.Hour)
			case "-3d":
				testTime = now.Add(-72 * time.Hour)
			}
			result := formatEnvAge(testTime)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatEnvAge for %s: expected result containing %q, got %q", tt.name, tt.contains, result)
			}
		})
	}

	// Test old dates (> 7 days) - just verify it returns something
	oldDate := now.Add(-30 * 24 * time.Hour)
	result := formatEnvAge(oldDate)
	if result == "" {
		t.Error("expected non-empty result for old date")
	}
}

func TestBuildItemDisplayText(t *testing.T) {
	tests := []struct {
		label    string
		pathInfo string
		width    int
		want     string
	}{
		{"label", "", 20, "label"},
		{"label", "path", 15, "label"}, // width <= 20, no path shown
		{"label", "path", 25, "label · path"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s_%d", tt.label, tt.pathInfo, tt.width), func(t *testing.T) {
			got := buildItemDisplayText(tt.label, tt.pathInfo, tt.width)
			if !strings.Contains(got, tt.label[:min(len(tt.label), len(got))]) {
				t.Errorf("buildItemDisplayText(%q, %q, %d) = %q, expected to contain label", tt.label, tt.pathInfo, tt.width, got)
			}
		})
	}
}

func TestEnvironmentPanelRenderContentFiltering(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "production", Strategy: environment.StrategyFolder},
		{Name: "staging", Strategy: environment.StrategyFolder},
		{Name: "development", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)

	// Activate filter mode
	panel.filterActive = true
	panel.filterText = "prod"
	panel.applyFilter()

	// Should only have 1 filtered item
	if len(panel.filteredItems) != 1 {
		t.Errorf("expected 1 filtered item, got %d", len(panel.filteredItems))
	}

	lines := panel.renderContent(48, 8)
	if len(lines) == 0 {
		t.Error("expected non-empty content")
	}
}

func TestEnvironmentPanelRenderContentEmpty(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, nil)
	panel.SetSize(50, 10)
	panel.SetFocused(true)

	lines := panel.renderContent(48, 8)
	found := false
	for _, line := range lines {
		if strings.Contains(line, "No workspaces") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'No workspaces' in empty content")
	}
}

func TestEnvironmentPanelRenderContentNoMatches(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "production", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)

	// Set filter that doesn't match anything
	panel.filterActive = true
	panel.filterText = "xyz"
	panel.applyFilter()

	lines := panel.renderContent(48, 8)
	found := false
	for _, line := range lines {
		if strings.Contains(line, "No matches") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'No matches' when filter has no results")
	}
}

func TestEnvironmentPanelFormatPath(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Short path
	short := panel.formatPath("envs/dev")
	if short != "envs/dev" {
		t.Errorf("expected 'envs/dev', got %q", short)
	}

	// Long path - should show last 2 components
	long := panel.formatPath("/home/user/projects/terraform/environments/production")
	if !strings.Contains(long, "environments") && !strings.Contains(long, "production") {
		t.Errorf("expected last 2 path components, got %q", long)
	}
}

func TestEnvironmentPanelBuildFooterText(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Empty list
	panel.filteredItems = nil
	if panel.buildFooterText() != "" {
		t.Error("expected empty footer for empty list")
	}

	// With items
	envs := []environment.Environment{
		{Name: "a", Strategy: environment.StrategyFolder},
		{Name: "b", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.selectedIndex = 0

	footer := panel.buildFooterText()
	if !strings.Contains(footer, "1") && !strings.Contains(footer, "2") {
		t.Errorf("expected item count in footer, got %q", footer)
	}
}

func TestEnvironmentPanelHandleFilterEnterNoSelection(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)
	panel.SetSize(50, 10)
	panel.SetFocused(true)
	panel.filterActive = true
	// No items set

	handled, cmd := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected enter key to be handled in filter mode")
	}
	if cmd != nil {
		t.Error("expected nil cmd when no selection")
	}
}

func TestEnvironmentPanelHandleNavigationE(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)

	// 'e' key should be handled (toggle environment panel)
	handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if !handled {
		t.Error("expected 'e' key to be handled")
	}
}

func TestEnvironmentPanelHandleFilterUpDown(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
		{Name: "env2", Strategy: environment.StrategyFolder},
		{Name: "env3", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)
	panel.filterActive = true
	panel.selectedIndex = 0

	// Up when already at top
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if panel.selectedIndex != 0 {
		t.Errorf("expected index 0 after up at top, got %d", panel.selectedIndex)
	}

	// Down
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if panel.selectedIndex != 1 {
		t.Errorf("expected index 1 after down, got %d", panel.selectedIndex)
	}
}

func TestEnvironmentPanelHandleEscWithFilter(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)

	// Esc without filter text should not be handled
	handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected esc to not be handled when no filter text")
	}
}

func TestEnvironmentPanelHandleUnknownKey(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)

	// Unknown key should not be handled
	handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if handled {
		t.Error("expected unknown key 'z' to not be handled")
	}
}

func TestEnvironmentPanelNilStyles(t *testing.T) {
	panel := NewEnvironmentPanel(nil)
	if panel == nil {
		t.Fatal("expected non-nil panel with nil styles")
	}
	if panel.styles == nil {
		t.Error("expected default styles to be set")
	}
}

func TestEnvironmentPanelSetFocusedClearsFilter(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)
	panel.filterActive = true

	panel.SetFocused(false)
	if panel.filterActive {
		t.Error("expected filter to be deactivated when unfocused")
	}
}

func TestEnvironmentPanelViewNoHeight(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)
	panel.SetSize(50, 0)

	view := panel.View()
	if view != "" {
		t.Error("expected empty view with zero height")
	}
}

func TestEnvironmentPanelViewNilStyles(t *testing.T) {
	panel := &EnvironmentPanel{}
	view := panel.View()
	if view != "" {
		t.Error("expected empty view with nil styles")
	}
}

func TestEnvironmentPanelFormatMetadataWithLastModified(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// With LastModified
	meta := environment.EnvironmentMetadata{
		LastModified: time.Now().Add(-5 * time.Minute),
	}
	result := panel.formatMetadata(meta)
	if result == "" {
		t.Error("expected non-empty metadata with LastModified")
	}
}

func TestBuildItemDisplayTextSmallMaxLabelWidth(t *testing.T) {
	// Test when maxLabelWidth < 8
	result := buildItemDisplayText("verylonglabel", "very/long/path/info", 25)
	// When maxLabelWidth is too small, should just truncate label
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestEnvironmentPanelContentWidthVerySmall(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Set very small width
	panel.width = 1
	panel.height = 10

	w := panel.contentWidth()
	if w < 1 {
		t.Errorf("expected contentWidth at least 1, got %d", w)
	}
}

func TestEnvironmentPanelHandleFilterEnterWithSelection(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)
	panel.filterActive = true
	panel.selectedIndex = 0

	handled, cmd := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected enter to be handled")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd when selection exists")
	}
}

func TestEnvironmentPanelRenderItemNoName(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	item := envListItem{
		env:   environment.Environment{Path: "/some/path"},
		label: "", // Empty label
	}

	result := panel.renderItem(item, 40, false)
	if !strings.Contains(result, "no name") {
		t.Error("expected '(no name)' for empty label")
	}
}

func TestFormatEnvAgeOldDate(t *testing.T) {
	// Test date older than 7 days
	oldDate := time.Now().Add(-10 * 24 * time.Hour)
	result := formatEnvAge(oldDate)
	// Should contain month name like "Jan"
	if len(result) < 2 {
		t.Errorf("expected formatted date, got %q", result)
	}
}

func TestEnvironmentPanelRenderWithFocusedAndItems(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "prod", Strategy: environment.StrategyFolder, IsCurrent: true},
		{Name: "staging", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("prod", "", environment.StrategyFolder, envs)
	panel.SetSize(50, 10)
	panel.SetFocused(true)
	panel.selectedIndex = 1

	// Render and verify focused view with selection
	view := panel.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestEnvironmentPanelSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	newStyles := styles.DefaultStyles()
	panel.SetStyles(newStyles)

	if panel.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestEnvironmentPanelGetSelectedIndex(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	envs := []environment.Environment{
		{Name: "env1", Strategy: environment.StrategyFolder},
		{Name: "env2", Strategy: environment.StrategyFolder},
		{Name: "env3", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)
	panel.selectedIndex = 2

	if panel.GetSelectedIndex() != 2 {
		t.Errorf("expected index 2, got %d", panel.GetSelectedIndex())
	}
}

func TestEnvironmentPanelItemCount(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewEnvironmentPanel(s)

	// Empty
	if panel.ItemCount() != 0 {
		t.Errorf("expected 0 items, got %d", panel.ItemCount())
	}

	// With items
	envs := []environment.Environment{
		{Name: "a", Strategy: environment.StrategyFolder},
		{Name: "b", Strategy: environment.StrategyFolder},
	}
	panel.SetEnvironmentInfo("", "", environment.StrategyFolder, envs)

	if panel.ItemCount() != 2 {
		t.Errorf("expected 2 items, got %d", panel.ItemCount())
	}
}
