package components

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

// EnvironmentChangedMsg is sent when the user selects a new environment.
type EnvironmentChangedMsg struct {
	Environment environment.Environment
}

// envListItem represents an environment in the list.
type envListItem struct {
	env    environment.Environment
	label  string
	detail string
}

// EnvironmentPanel displays a list of workspaces/environments with filtering.
type EnvironmentPanel struct {
	styles *styles.Styles
	frame  *PanelFrame
	width  int
	height int

	// Focus and filtering state
	focused      bool
	filterActive bool
	filterText   string

	// Environment state
	current      string
	environments []environment.Environment

	// List state
	items         []envListItem
	filteredItems []envListItem
	selectedIndex int
	scrollOffset  int
	lastMove      int
}

// NewEnvironmentPanel creates a new environment panel.
func NewEnvironmentPanel(s *styles.Styles) *EnvironmentPanel {
	if s == nil {
		s = styles.DefaultStyles()
	}
	return &EnvironmentPanel{
		styles: s,
		frame:  NewPanelFrame(s),
	}
}

// SetSize updates the panel dimensions.
func (e *EnvironmentPanel) SetSize(width, height int) {
	e.width = width
	e.height = height
	e.frame.SetSize(width, height)
	e.adjustScrollOffset()
}

// SetFocused sets the focus state.
func (e *EnvironmentPanel) SetFocused(focused bool) {
	e.focused = focused
	if !focused {
		e.filterActive = false
	}
}

// IsFocused returns whether the panel is focused.
func (e *EnvironmentPanel) IsFocused() bool {
	return e.focused
}

// SetStyles updates the component styles.
func (e *EnvironmentPanel) SetStyles(s *styles.Styles) {
	e.styles = s
	if e.frame != nil {
		e.frame.SetStyles(s)
	}
}

// SelectorActive reports whether filtering is active.
// For compatibility - the panel is always in "selector" mode now.
func (e *EnvironmentPanel) SelectorActive() bool {
	return e.focused
}

// SetEnvironmentInfo updates the environment information.
func (e *EnvironmentPanel) SetEnvironmentInfo(current, _ string, _ environment.StrategyType, environments []environment.Environment) {
	e.current = current
	e.environments = environments
	e.rebuildItems()
}

// Filtering reports whether filter input is active.
func (e *EnvironmentPanel) Filtering() bool {
	return e.filterActive
}

// Update handles Bubble Tea messages.
func (e *EnvironmentPanel) Update(_ tea.Msg) (any, tea.Cmd) {
	return e, nil
}

// HandleKey handles key events.
func (e *EnvironmentPanel) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	if !e.focused {
		return false, nil
	}

	// Handle filter input when active
	if e.filterActive {
		if handled, cmd := e.handleFilterKey(msg); handled {
			return true, cmd
		}
	}

	// Navigation and actions
	return e.handleNavigationKey(msg.String())
}

func (e *EnvironmentPanel) handleFilterKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		e.filterActive = false
		e.filterText = ""
		e.applyFilter()
		return true, nil
	case tea.KeyEnter:
		if selected := e.selectedEnvironment(); selected != nil {
			e.filterActive = false
			e.filterText = ""
			e.applyFilter()
			return true, func() tea.Msg {
				return EnvironmentChangedMsg{Environment: *selected}
			}
		}
		return true, nil
	case tea.KeyBackspace:
		if e.filterText != "" {
			e.filterText = e.filterText[:len(e.filterText)-1]
			e.applyFilter()
		}
		return true, nil
	case tea.KeyCtrlU:
		e.filterText = ""
		e.applyFilter()
		return true, nil
	case tea.KeyRunes:
		e.filterText += string(msg.Runes)
		e.applyFilter()
		return true, nil
	case tea.KeyUp:
		e.moveUp()
		return true, nil
	case tea.KeyDown:
		e.moveDown()
		return true, nil
	default:
		return false, nil
	}
}

func (e *EnvironmentPanel) handleNavigationKey(key string) (bool, tea.Cmd) {
	switch key {
	case "/":
		e.filterActive = true
		return true, nil
	case "up", "k":
		e.moveUp()
		return true, nil
	case keybinds.KeyDown, "j":
		e.moveDown()
		return true, nil
	case "enter":
		if selected := e.selectedEnvironment(); selected != nil {
			return true, func() tea.Msg {
				return EnvironmentChangedMsg{Environment: *selected}
			}
		}
		return true, nil
	case "esc":
		if e.filterText != "" {
			e.filterText = ""
			e.applyFilter()
			return true, nil
		}
		return false, nil
	case "e":
		return true, nil
	default:
		return false, nil
	}
}

// View renders the panel.
func (e *EnvironmentPanel) View() string {
	if e.styles == nil || e.height <= 0 {
		return ""
	}

	// Total content area (excluding borders)
	totalContentHeight := max(1, e.height-2)

	contentWidth := e.contentWidth()

	// Only show scrollbar and footer when focused
	hasScrollbar := e.focused && len(e.filteredItems) > e.itemsHeight()
	footerText := ""
	if e.focused {
		footerText = e.buildFooterText()
	}

	// Build title - always "Environment"
	titleText := "Environment"

	// Configure frame
	scrollPos, thumbSize := CalculateScrollParams(e.scrollOffset, e.itemsHeight(), len(e.filteredItems))
	e.frame.SetConfig(PanelFrameConfig{
		PanelID:       "[1]",
		Tabs:          []string{titleText},
		ActiveTab:     0,
		Focused:       e.focused,
		FooterText:    footerText,
		ShowScrollbar: hasScrollbar,
		ScrollPos:     scrollPos,
		ThumbSize:     thumbSize,
	})

	// Render content
	lines := e.renderContent(contentWidth, totalContentHeight)

	return e.frame.RenderWithContent(lines)
}

// rebuildItems creates the item list from environments.
func (e *EnvironmentPanel) rebuildItems() {
	e.items = make([]envListItem, 0, len(e.environments))

	for _, env := range e.environments {
		item := envListItem{
			env:    env,
			label:  e.envLabel(env),
			detail: e.formatMetadata(env.Metadata),
		}
		item.env.IsCurrent = e.isCurrentEnv(env)
		e.items = append(e.items, item)
	}

	e.applyFilter()
}

// applyFilter filters items based on filter text.
func (e *EnvironmentPanel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(e.filterText))

	if query == "" {
		e.filteredItems = e.items
	} else {
		e.filteredItems = make([]envListItem, 0)
		for _, item := range e.items {
			if e.itemMatchesFilter(item, query) {
				e.filteredItems = append(e.filteredItems, item)
			}
		}
	}

	// Adjust selection
	if e.selectedIndex >= len(e.filteredItems) {
		e.selectedIndex = max(0, len(e.filteredItems)-1)
	}

	// Try to keep current environment selected if visible
	if query == "" {
		for i, item := range e.filteredItems {
			if item.env.IsCurrent {
				e.selectedIndex = i
				break
			}
		}
	}

	e.adjustScrollOffset()
}

// itemMatchesFilter checks if an item matches the filter query.
func (e *EnvironmentPanel) itemMatchesFilter(item envListItem, query string) bool {
	if fuzzyMatchEnvPanel(query, strings.ToLower(item.label)) {
		return true
	}
	if item.detail != "" && fuzzyMatchEnvPanel(query, strings.ToLower(item.detail)) {
		return true
	}
	return false
}

// fuzzyMatchEnvPanel performs fuzzy matching.
func fuzzyMatchEnvPanel(query, candidate string) bool {
	if query == "" {
		return true
	}
	q := []rune(query)
	c := []rune(candidate)
	if len(q) > len(c) {
		return false
	}
	qi := 0
	for _, r := range c {
		if r == q[qi] {
			qi++
			if qi == len(q) {
				return true
			}
		}
	}
	return false
}

// envLabel returns the display label for an environment.
func (e *EnvironmentPanel) envLabel(env environment.Environment) string {
	// Try Name first
	if env.Name != "" {
		return env.Name
	}
	// Fallback to path basename
	if env.Path != "" {
		return filepath.Base(env.Path)
	}
	return "(unknown)"
}

// isCurrentEnv checks if an environment is the current one.
func (e *EnvironmentPanel) isCurrentEnv(env environment.Environment) bool {
	if e.current == "" {
		return env.IsCurrent
	}
	if env.Strategy == environment.StrategyWorkspace {
		return env.Name == e.current
	}
	return env.Path == e.current || env.Name == e.current
}

// formatMetadata creates a compact metadata string.
func (e *EnvironmentPanel) formatMetadata(meta environment.EnvironmentMetadata) string {
	parts := []string{}
	if meta.ResourceCount > 0 {
		parts = append(parts, fmt.Sprintf("%d res", meta.ResourceCount))
	}
	if meta.HasState && meta.ResourceCount == 0 {
		parts = append(parts, "state")
	}
	if !meta.LastModified.IsZero() {
		parts = append(parts, formatEnvAge(meta.LastModified))
	}
	return strings.Join(parts, " · ")
}

// selectedEnvironment returns the currently selected environment.
func (e *EnvironmentPanel) selectedEnvironment() *environment.Environment {
	if e.selectedIndex >= 0 && e.selectedIndex < len(e.filteredItems) {
		env := e.filteredItems[e.selectedIndex].env
		return &env
	}
	return nil
}

// GetSelectedIndex returns the currently selected item index (implements SelectablePanel).
func (e *EnvironmentPanel) GetSelectedIndex() int {
	return e.selectedIndex
}

// SetSelectedIndex sets the selected item index (implements SelectablePanel).
func (e *EnvironmentPanel) SetSelectedIndex(index int) {
	if index >= 0 && index < len(e.filteredItems) {
		e.selectedIndex = index
		e.adjustScrollOffset()
	}
}

// SelectVisibleRow updates selection from a visible content row and returns the selected environment.
func (e *EnvironmentPanel) SelectVisibleRow(row int) *environment.Environment {
	if row < 0 {
		return nil
	}
	if e.filterActive {
		row--
	}
	if row < 0 {
		return nil
	}
	idx := e.scrollOffset + row
	if idx < 0 || idx >= len(e.filteredItems) {
		return nil
	}
	e.selectedIndex = idx
	e.lastMove = 0
	e.adjustScrollOffset()
	env := e.filteredItems[idx].env
	return &env
}

// ItemCount returns the total number of filtered items (implements SelectablePanel).
func (e *EnvironmentPanel) ItemCount() int {
	return len(e.filteredItems)
}

// moveUp moves selection up.
func (e *EnvironmentPanel) moveUp() {
	if e.selectedIndex > 0 {
		e.selectedIndex--
		e.lastMove = -1
		e.adjustScrollOffset()
	}
}

// moveDown moves selection down.
func (e *EnvironmentPanel) moveDown() {
	if e.selectedIndex < len(e.filteredItems)-1 {
		e.selectedIndex++
		e.lastMove = 1
		e.adjustScrollOffset()
	}
}

// itemsHeight returns height available for item list (excluding filter line if active).
func (e *EnvironmentPanel) itemsHeight() int {
	h := e.height - 2 // borders
	if e.filterActive {
		h-- // filter input line
	}
	if h < 1 {
		return 1
	}
	return h
}

// contentWidth returns width available for content.
func (e *EnvironmentPanel) contentWidth() int {
	w := e.width - 2 // borders
	if len(e.filteredItems) > e.itemsHeight() {
		w-- // scrollbar
	}
	if w < 1 {
		return 1
	}
	return w
}

// adjustScrollOffset ensures selected item is visible.
func (e *EnvironmentPanel) adjustScrollOffset() {
	itemsHeight := e.itemsHeight()
	if itemsHeight <= 0 || len(e.filteredItems) == 0 {
		e.scrollOffset = 0
		return
	}

	// Anchor positions for smooth scrolling (consistent with ResourceList)
	anchorTop := min(2, itemsHeight-1)
	anchorBottom := max(itemsHeight-3, anchorTop)
	e.scrollOffset = adjustScrollOffset(
		e.scrollOffset,
		e.selectedIndex,
		len(e.filteredItems),
		itemsHeight,
		e.lastMove,
		anchorTop,
		anchorBottom,
	)
}

// renderContent renders the panel content.
func (e *EnvironmentPanel) renderContent(width, totalHeight int) []string {
	lines := make([]string, 0, totalHeight)

	// When not focused, show only the current workspace (no selection highlight)
	if !e.focused {
		return e.renderUnfocusedContent(width, totalHeight)
	}

	// Filter input line (if active)
	if e.filterActive {
		filterLine := e.styles.Dimmed.Render("/") + e.filterText
		if e.filterText == "" {
			filterLine = e.styles.Dimmed.Render("/ type to filter...")
		}
		lines = append(lines, e.padLine(filterLine, width))
	}

	// Calculate remaining height for items
	itemsHeight := totalHeight
	if e.filterActive {
		itemsHeight--
	}

	// Empty state
	if len(e.filteredItems) == 0 {
		msg := "No workspaces"
		if e.filterText != "" {
			msg = "No matches"
		}
		lines = append(lines, e.padLine(e.styles.Dimmed.Render(msg), width))
		emptyLine := GetPadding(width)
		for len(lines) < totalHeight {
			lines = append(lines, emptyLine)
		}
		return lines
	}

	// Render visible items
	start := e.scrollOffset
	end := min(start+itemsHeight, len(e.filteredItems))

	for i := start; i < end; i++ {
		item := e.filteredItems[i]
		isSelected := e.focused && i == e.selectedIndex
		lines = append(lines, e.renderItem(item, width, isSelected))
	}

	// Pad remaining lines
	emptyLine := GetPadding(width)
	for len(lines) < totalHeight {
		lines = append(lines, emptyLine)
	}

	return lines
}

// renderUnfocusedContent renders a simplified view when panel is not focused.
func (e *EnvironmentPanel) renderUnfocusedContent(width, totalHeight int) []string {
	lines := make([]string, 0, totalHeight)

	// Find current workspace
	var currentItem *envListItem
	for i := range e.filteredItems {
		if e.filteredItems[i].env.IsCurrent {
			currentItem = &e.filteredItems[i]
			break
		}
	}

	switch {
	case currentItem != nil:
		// Show current workspace without selection highlighting
		lines = append(lines, e.renderItem(*currentItem, width, false))
	case len(e.filteredItems) > 0:
		// No current marked, show first item
		lines = append(lines, e.renderItem(e.filteredItems[0], width, false))
	default:
		// Empty state
		lines = append(lines, e.padLine(e.styles.Dimmed.Render("No workspaces"), width))
	}

	// Pad remaining lines
	emptyLine := GetPadding(width)
	for len(lines) < totalHeight {
		lines = append(lines, emptyLine)
	}

	return lines
}

// renderItem renders a single environment item.
func (e *EnvironmentPanel) renderItem(item envListItem, width int, isSelected bool) string {
	// Current marker
	marker := " "
	if item.env.IsCurrent {
		marker = "*"
	}

	// Build the base text
	label := item.label
	if label == "" {
		label = "(no name)"
	}

	// Build path info (show relative or short path)
	pathInfo := ""
	if item.env.Path != "" {
		pathInfo = e.formatPath(item.env.Path)
	}

	// Calculate available width for content (after marker)
	markerWidth := 2 // marker + space
	availableWidth := max(1, width-markerWidth)

	// Build the display: "name · path" or just "name"
	displayText := buildItemDisplayText(label, pathInfo, availableWidth)

	// Truncate the entire display if it still exceeds available width
	if runewidth.StringWidth(displayText) > availableWidth {
		displayText = runewidth.Truncate(displayText, availableWidth, "…")
	}

	// Build full line: marker + display (no padding yet)
	line := marker + " " + displayText

	// Apply styling
	if isSelected {
		bg := e.styles.SelectedLineBackground
		// Style the content first
		styled := e.styles.LineItemText.Background(bg).Bold(true).Render(line)
		// Then pad with background color to fill width
		return padStyledWithBg(styled, width, bg)
	}

	// Non-selected: just pad with spaces
	return padToWidth(line, width)
}

func buildItemDisplayText(label, pathInfo string, availableWidth int) string {
	if pathInfo == "" || availableWidth <= 20 {
		return runewidth.Truncate(label, availableWidth, "…")
	}

	separator := " · "
	sepWidth := runewidth.StringWidth(separator)
	pathWidth := runewidth.StringWidth(pathInfo)

	// Calculate max label width.
	maxLabelWidth := availableWidth - sepWidth - pathWidth
	if maxLabelWidth < 8 {
		return runewidth.Truncate(label, availableWidth, "…")
	}

	truncLabel := label
	if runewidth.StringWidth(label) > maxLabelWidth {
		truncLabel = runewidth.Truncate(label, maxLabelWidth, "…")
	}
	return truncLabel + separator + pathInfo
}

// padStyledWithBg pads a styled string to width with background color.
func padStyledWithBg(styled string, width int, bg lipgloss.AdaptiveColor) string {
	return PadLineWithBg(styled, width, bg)
}

// padToWidth pads a plain string to width with spaces.
func padToWidth(line string, width int) string {
	return PadLine(line, width)
}

// formatPath returns a short display version of the path.
func (e *EnvironmentPanel) formatPath(path string) string {
	// Show just the last 2 path components for brevity
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	if len(parts) > 2 {
		return filepath.Join(parts[len(parts)-2:]...)
	}
	return path
}

// padLine pads a line to the given width.
func (e *EnvironmentPanel) padLine(line string, width int) string {
	return padToWidth(line, width)
}

// buildFooterText builds the footer text.
func (e *EnvironmentPanel) buildFooterText() string {
	if len(e.filteredItems) == 0 {
		return ""
	}
	return FormatItemCount(e.selectedIndex+1, len(e.filteredItems))
}

// formatEnvAge formats a time as a relative age string.
func formatEnvAge(t time.Time) string {
	delta := time.Since(t)
	if delta < time.Minute {
		return "now"
	}
	if delta < time.Hour {
		return fmt.Sprintf("%dm", int(delta.Minutes()))
	}
	if delta < 24*time.Hour {
		return fmt.Sprintf("%dh", int(delta.Hours()))
	}
	if delta < 7*24*time.Hour {
		return fmt.Sprintf("%dd", int(delta.Hours()/24))
	}
	return t.Format("Jan 2")
}
