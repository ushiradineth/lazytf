package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

// ResourceList displays a list of resources.
type ResourceList struct {
	viewport       viewport.Model
	frame          *PanelFrame
	resources      []terraform.ResourceChange
	selectedIndex  int
	diffEngine     *diff.Engine
	styles         *styles.Styles
	width          int
	height         int
	filterActions  map[terraform.ActionType]bool
	searchQuery    string
	groupExpanded  map[string]bool
	visibleItems   []listItem
	allExpanded    bool
	searchActive   bool
	matchScores    map[string]int
	lastMove       int
	operationState *terraform.OperationState
	showStatus     bool
	focused        bool

	// Summary counts for plan
	summaryCreate  int
	summaryUpdate  int
	summaryDelete  int
	summaryReplace int
}

// NewResourceList creates a new resource list component.
func NewResourceList(s *styles.Styles) *ResourceList {
	vp := viewport.New(0, 0)

	return &ResourceList{
		viewport:      vp,
		frame:         NewPanelFrame(s),
		resources:     []terraform.ResourceChange{},
		selectedIndex: 0,
		diffEngine:    diff.NewEngine(),
		styles:        s,
		filterActions: map[terraform.ActionType]bool{
			terraform.ActionCreate:  true,
			terraform.ActionUpdate:  true,
			terraform.ActionDelete:  true,
			terraform.ActionReplace: true,
		},
		groupExpanded: make(map[string]bool),
		allExpanded:   true,
	}
}

// SetSize sets the dimensions of the resource list.
func (r *ResourceList) SetSize(width, height int) {
	r.width = width
	r.height = height
	r.frame.SetSize(width, height)
	// Viewport dimensions are content area (minus borders)
	r.viewport.Width = width - 2
	r.viewport.Height = height - 2
	r.updateViewport()
}

// contentWidth returns the width available for content (excluding borders and scrollbar).
func (r *ResourceList) contentWidth() int {
	w := r.width - 2 // left and right border
	if len(r.visibleItems) > r.viewport.Height {
		w-- // scrollbar
	}
	if w < 1 {
		return 1
	}
	return w
}

// SetResources updates the list of resources to display.
func (r *ResourceList) SetResources(resources []terraform.ResourceChange) {
	r.resources = resources
	r.selectedIndex = 0
	r.diffEngine.ResetCache()
	r.updateViewport()
}

// Refresh recalculates the viewport content.
func (r *ResourceList) Refresh() {
	r.updateViewport()
}

// SetOperationState attaches the operation state for status display.
func (r *ResourceList) SetOperationState(state *terraform.OperationState) {
	r.operationState = state
	r.updateViewport()
}

// SetShowStatus toggles the status column.
func (r *ResourceList) SetShowStatus(show bool) {
	r.showStatus = show
	r.updateViewport()
}

// ShowStatus reports whether the status column is enabled.
func (r *ResourceList) ShowStatus() bool {
	return r.showStatus
}

// SetFilter sets which action types to display.
func (r *ResourceList) SetFilter(actionType terraform.ActionType, enabled bool) {
	r.filterActions[actionType] = enabled
	r.updateViewport()
}

// SetSearchQuery sets the current search query for filtering resources.
func (r *ResourceList) SetSearchQuery(query string) {
	r.searchQuery = strings.ToLower(strings.TrimSpace(query))
	r.updateViewport()
}

// SetFocused sets the focus state (implements Panel interface).
func (r *ResourceList) SetFocused(focused bool) {
	r.focused = focused
}

// IsFocused returns whether the panel is focused (implements Panel interface).
func (r *ResourceList) IsFocused() bool {
	return r.focused
}

// SetStyles updates the component styles.
func (r *ResourceList) SetStyles(s *styles.Styles) {
	r.styles = s
	if r.frame != nil {
		r.frame.SetStyles(s)
	}
}

// HandleKey handles key events (implements Panel interface).
func (r *ResourceList) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		r.MoveUp()
		return true, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys(keybinds.KeyDown, "j"))):
		r.MoveDown()
		return true, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		r.ToggleGroup()
		return true, nil
	}
	return false, nil
}

// SetSummary sets the plan summary counts.
func (r *ResourceList) SetSummary(create, update, deleteCount, replace int) {
	r.summaryCreate = create
	r.summaryUpdate = update
	r.summaryDelete = deleteCount
	r.summaryReplace = replace
	r.updateViewport()
}

func (r *ResourceList) summaryHeaderLines() int {
	if r.summaryCreate > 0 || r.summaryUpdate > 0 || r.summaryDelete > 0 || r.summaryReplace > 0 {
		return 1
	}
	return 0
}

// SelectVisibleRow sets selection by visible row index within the panel content area.
func (r *ResourceList) SelectVisibleRow(row int) bool {
	if row < 0 || r.viewport.Height <= 0 {
		return false
	}
	listRow := row - r.summaryHeaderLines()
	if listRow < 0 {
		return false
	}
	idx := r.viewport.YOffset + listRow
	if idx < 0 || idx >= len(r.visibleItems) {
		return false
	}
	r.selectedIndex = idx
	r.lastMove = 0
	r.adjustViewportOffset()
	return true
}

// MoveUp moves the selection up.
func (r *ResourceList) MoveUp() {
	r.moveSelection(-1)
}

// MoveDown moves the selection down.
func (r *ResourceList) MoveDown() {
	r.moveSelection(1)
}

func (r *ResourceList) moveSelection(delta int) {
	if len(r.visibleItems) == 0 || delta == 0 {
		return
	}

	previous := r.selectedIndex
	next := previous + delta
	if next < 0 {
		next = 0
	}
	if next >= len(r.visibleItems) {
		next = len(r.visibleItems) - 1
	}
	if next == previous {
		return
	}

	r.selectedIndex = next
	if delta > 0 {
		r.lastMove = 1
	} else {
		r.lastMove = -1
	}
	r.adjustViewportOffset()
}

// Init initializes the component.
func (r *ResourceList) Init() tea.Cmd {
	return nil
}

// Update handles messages (implements Panel interface).
func (r *ResourceList) Update(msg tea.Msg) (any, tea.Cmd) {
	var cmd tea.Cmd
	handled := false

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, key.NewBinding(key.WithKeys("up", "k"))):
			r.MoveUp()
			handled = true
		case key.Matches(keyMsg, key.NewBinding(key.WithKeys(keybinds.KeyDown, "j"))):
			r.MoveDown()
			handled = true
		case key.Matches(keyMsg, key.NewBinding(key.WithKeys("enter", " "))):
			r.ToggleGroup()
		}
	}

	if handled {
		return r, nil
	}

	r.viewport, cmd = r.viewport.Update(msg)
	return r, cmd
}

// View renders the resource list.
func (r *ResourceList) View() string {
	// Build content with summary header
	var contentParts []string

	// Add summary header if we have counts
	if r.summaryCreate > 0 || r.summaryUpdate > 0 || r.summaryDelete > 0 || r.summaryReplace > 0 {
		summary := r.renderSummaryHeader()
		contentParts = append(contentParts, summary)
	}

	// Add resource list
	var listView string
	if len(r.visibleItems) == 0 {
		listView = r.viewport.View()
	} else {
		listView = r.renderVisibleItems()
	}
	contentParts = append(contentParts, listView)

	content := strings.Join(contentParts, "\n")

	// Calculate content dimensions
	contentHeight := max(1, r.height-2)

	// Build title with filter indicators
	titleText := "Resources"
	indicatorText := r.renderFilterIndicators()
	if indicatorText != "" {
		titleText = titleText + " " + indicatorText
	}

	// Configure frame with scrollbar info
	scrollPos, thumbSize := CalculateScrollParams(r.viewport.YOffset, r.viewport.Height, len(r.visibleItems))
	r.frame.SetConfig(PanelFrameConfig{
		PanelID:       "[2]",
		Tabs:          []string{titleText},
		ActiveTab:     0,
		Focused:       r.focused,
		FooterText:    r.buildFooterText(),
		ShowScrollbar: len(r.visibleItems) > contentHeight,
		ScrollPos:     scrollPos,
		ThumbSize:     thumbSize,
	})

	// Split content into lines
	contentLines := strings.Split(content, "\n")

	// Pad content lines to fill panel
	result := make([]string, contentHeight)
	contentW := r.frame.ContentWidth()
	emptyLine := GetPadding(contentW)
	for i := range contentHeight {
		if i < len(contentLines) {
			result[i] = r.padLineToWidth(contentLines[i], contentW)
		} else {
			result[i] = emptyLine
		}
	}

	return r.frame.RenderWithContent(result)
}

// buildFooterText builds the footer text with item count.
func (r *ResourceList) buildFooterText() string {
	if len(r.visibleItems) == 0 {
		return ""
	}
	return FormatItemCount(r.selectedIndex+1, len(r.visibleItems))
}

// padLineToWidth pads a line to the given width.
func (r *ResourceList) padLineToWidth(line string, width int) string {
	return PadLine(line, width)
}

// renderSummaryHeader renders the plan summary counts: +5 ~3 -2 ±1.
func (r *ResourceList) renderSummaryHeader() string {
	parts := []string{}
	if r.summaryCreate > 0 {
		parts = append(parts, r.styles.DiffAdd.Render(fmt.Sprintf("+%d", r.summaryCreate)))
	}
	if r.summaryUpdate > 0 {
		parts = append(parts, r.styles.DiffChange.Render(fmt.Sprintf("~%d", r.summaryUpdate)))
	}
	if r.summaryDelete > 0 {
		parts = append(parts, r.styles.DiffRemove.Render(fmt.Sprintf("-%d", r.summaryDelete)))
	}
	if r.summaryReplace > 0 {
		parts = append(parts, r.styles.DiffChange.Render(fmt.Sprintf("±%d", r.summaryReplace)))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func (r *ResourceList) renderFilterIndicators() string {
	parts := []string{
		r.filterIndicator(terraform.ActionCreate, "C"),
		r.filterIndicator(terraform.ActionDelete, "D"),
		r.filterIndicator(terraform.ActionReplace, "R"),
		r.filterIndicator(terraform.ActionUpdate, "U"),
	}
	return strings.Join(parts, "/")
}

func (r *ResourceList) filterIndicator(action terraform.ActionType, label string) string {
	enabled := r.filterActions[action]
	text := label
	if enabled {
		return text
	}
	return r.styles.Dimmed.Render(text)
}

// getFilteredResources returns resources that pass the current filter.
func (r *ResourceList) getFilteredResources() []terraform.ResourceChange {
	filtered := []terraform.ResourceChange{}
	query := r.searchQuery
	r.searchActive = query != ""
	r.matchScores = make(map[string]int)
	for _, resource := range r.resources {
		if !r.filterActions[resource.Action] {
			continue
		}
		if query == "" || resourceMatchesQuery(resource, query) {
			if query != "" {
				r.matchScores[resource.Address] = bestQueryScore(query, resource)
			}
			filtered = append(filtered, resource)
		}
	}
	if query != "" {
		sort.Slice(filtered, func(i, j int) bool {
			left := r.matchScores[filtered[i].Address]
			right := r.matchScores[filtered[j].Address]
			if left == right {
				return filtered[i].Address < filtered[j].Address
			}
			return left < right
		})
	}
	return filtered
}

// updateViewport regenerates the viewport content based on current state.
func (r *ResourceList) updateViewport() {
	filtered := r.getFilteredResources()
	if len(filtered) == 0 {
		r.selectedIndex = 0
		r.visibleItems = nil
		r.viewport.YOffset = 0
		r.viewport.SetContent(r.styles.Dimmed.Render("No resources to display"))
		return
	}

	r.visibleItems = r.buildVisibleItems(filtered)
	if len(r.visibleItems) == 0 {
		r.selectedIndex = 0
		r.viewport.YOffset = 0
		r.viewport.SetContent(r.styles.Dimmed.Render("No resources to display"))
		return
	}

	if r.selectedIndex >= len(r.visibleItems) {
		r.selectedIndex = len(r.visibleItems) - 1
	}
	r.viewport.SetContent(strings.Repeat("\n", len(r.visibleItems)-1))
	r.adjustViewportOffset()
}

func (r *ResourceList) adjustViewportOffset() {
	if r.viewport.Height <= 0 || len(r.visibleItems) == 0 {
		return
	}

	maxOffset := max(0, len(r.visibleItems)-r.viewport.Height)

	anchorTop := min(2, r.viewport.Height-1)
	anchorBottom := max(r.viewport.Height-3, anchorTop)

	switch {
	case r.lastMove > 0:
		threshold := r.viewport.YOffset + anchorBottom
		if r.selectedIndex > threshold {
			r.viewport.YOffset = r.selectedIndex - anchorBottom
		}
	case r.lastMove < 0:
		threshold := r.viewport.YOffset + anchorTop
		if r.selectedIndex < threshold {
			r.viewport.YOffset = r.selectedIndex - anchorTop
		}
	default:
		if r.selectedIndex < r.viewport.YOffset {
			r.viewport.YOffset = r.selectedIndex
		} else if r.selectedIndex >= r.viewport.YOffset+r.viewport.Height {
			r.viewport.YOffset = r.selectedIndex - r.viewport.Height + 1
		}
	}

	if r.viewport.YOffset < 0 {
		r.viewport.YOffset = 0
	} else if r.viewport.YOffset > maxOffset {
		r.viewport.YOffset = maxOffset
	}
}

func (r *ResourceList) renderVisibleItems() string {
	if len(r.visibleItems) == 0 {
		return ""
	}

	start := max(0, r.viewport.YOffset)
	if start >= len(r.visibleItems) {
		start = len(r.visibleItems) - 1
	}
	end := len(r.visibleItems)
	if r.viewport.Height > 0 {
		end = min(start+r.viewport.Height, len(r.visibleItems))
	}

	var content strings.Builder
	contentWidth := r.contentWidth()
	for i := start; i < end; i++ {
		item := r.visibleItems[i]
		// Only show selection highlight when panel is focused
		isSelected := r.focused && i == r.selectedIndex
		switch item.kind {
		case itemGroup:
			content.WriteString(r.renderGroup(item.label, item.count, isSelected, item.expanded, item.indent, contentWidth))
		case itemResource:
			content.WriteString(r.renderResource(item.resource, isSelected, item.indent, contentWidth))
		}
		content.WriteString("\n")
	}

	if r.viewport.Height > 0 {
		for i := end - start; i < r.viewport.Height; i++ {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderResource renders a single resource line.
func (r *ResourceList) renderResource(resource *terraform.ResourceChange, isSelected bool, indent int, contentWidth int) string {
	var output strings.Builder

	// Get action style and icon
	actionIcon := resource.Action.GetActionIcon()
	actionStyle := r.getActionStyle(resource.Action)
	statusBadge, opStatus, elapsed := r.getStatusDisplay(*resource)

	// Calculate change count
	changeCount := r.diffEngine.CountChanges(resource)

	// Render the header line
	prefix := ""
	if indent > 0 {
		prefix = GetPadding(indent)
	}
	address := resource.Address
	if indent > 0 {
		address = trimModulePrefix(resource.Address, indent)
	}
	headerSuffix := ""
	if changeCount > 0 {
		headerSuffix = fmt.Sprintf("  (%d changes)", changeCount)
	}
	if elapsed != "" {
		headerSuffix += "  " + elapsed
	}

	selectedBg := r.styles.SelectedLineBackground
	iconStyle := actionStyle
	statusStyle := r.getStatusStyle(opStatus)
	addressStyle := r.styles.LineItemText
	suffixStyle := r.styles.LineItemText
	spaceStyle := lipgloss.NewStyle()
	prefixText := prefix
	if isSelected {
		iconStyle = iconStyle.Background(selectedBg).Bold(true)
		statusStyle = statusStyle.Background(selectedBg).Bold(true)
		addressStyle = addressStyle.Background(selectedBg).Bold(true)
		suffixStyle = suffixStyle.Background(selectedBg).Bold(true)
		spaceStyle = lipgloss.NewStyle().Background(selectedBg)
		prefixText = lipgloss.NewStyle().Background(selectedBg).Render(prefix)
	}
	icon := iconStyle.Render(actionIcon)
	statusText := ""
	if statusBadge != "" {
		statusText = statusStyle.Render(statusBadge)
	}
	addressText := addressStyle.Render(address)
	suffixText := ""
	if headerSuffix != "" {
		suffixText = suffixStyle.Render(headerSuffix)
	}
	headerLine := fmt.Sprintf("%s%s%s", prefixText, icon, spaceStyle.Render(" "))
	if statusText != "" {
		headerLine += statusText + spaceStyle.Render(" ")
	}
	headerLine += addressText + suffixText
	if contentWidth > 0 {
		headerLine = lipgloss.NewStyle().MaxWidth(contentWidth).Render(headerLine)
	}
	if isSelected {
		if contentWidth > 0 {
			headerLine = PadLineWithBg(headerLine, contentWidth, selectedBg)
		}
	} else if contentWidth > 0 {
		headerLine = PadLine(headerLine, contentWidth)
	}
	output.WriteString(headerLine)

	return output.String()
}

func trimModulePrefix(address string, indent int) string {
	if indent <= 0 {
		return address
	}
	for i := 0; i < indent/2; i++ {
		if !strings.HasPrefix(address, "module.") {
			break
		}
		parts := strings.SplitN(address, ".", 3)
		if len(parts) != 3 {
			break
		}
		address = parts[2]
	}
	return address
}

type moduleNode struct {
	name      string
	path      string
	children  map[string]*moduleNode
	resources []terraform.ResourceChange
}

func newModuleNode(name string) *moduleNode {
	return &moduleNode{
		name:     name,
		children: make(map[string]*moduleNode),
	}
}

func (n *moduleNode) insert(path []string, resource terraform.ResourceChange) {
	current := n
	for _, segment := range path {
		child, ok := current.children[segment]
		if !ok {
			child = newModuleNode(segment)
			if current.path == "" {
				child.path = "module." + segment
			} else {
				child.path = current.path + ".module." + segment
			}
			current.children[segment] = child
		}
		current = child
	}
	current.resources = append(current.resources, resource)
}

func (n *moduleNode) sortedChildNames() []string {
	names := make([]string, 0, len(n.children))
	for name := range n.children {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (n *moduleNode) countTotal() int {
	total := len(n.resources)
	for _, child := range n.children {
		total += child.countTotal()
	}
	return total
}

func (n *moduleNode) minScore(scores map[string]int) int {
	if len(scores) == 0 {
		return 0
	}
	minScore := -1
	for i := range n.resources {
		score := scores[n.resources[i].Address]
		if minScore == -1 || score < minScore {
			minScore = score
		}
	}
	for _, child := range n.children {
		score := child.minScore(scores)
		if minScore == -1 || score < minScore {
			minScore = score
		}
	}
	if minScore == -1 {
		return 0
	}
	return minScore
}

func modulePath(address string) []string {
	if !strings.HasPrefix(address, "module.") {
		return nil
	}
	parts := strings.Split(address, ".")
	names := []string{}
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "module" && i+1 < len(parts) {
			names = append(names, parts[i+1])
		}
	}
	return names
}

func fuzzyScore(query, candidate string) int {
	if query == "" {
		return 0
	}
	q := []rune(query)
	c := []rune(candidate)
	if len(q) > len(c) {
		return -1
	}
	pos := make([]int, 0, len(q))
	qi := 0
	for i, r := range c {
		if r == q[qi] {
			pos = append(pos, i)
			qi++
			if qi == len(q) {
				break
			}
		}
	}
	if qi != len(q) {
		return -1
	}
	span := pos[len(pos)-1] - pos[0]
	start := pos[0]
	gaps := 0
	dots := 0
	for i := 1; i < len(pos); i++ {
		gaps += pos[i] - pos[i-1] - 1
		for j := pos[i-1] + 1; j < pos[i]; j++ {
			if c[j] == '.' {
				dots++
			}
		}
	}
	return span + start + gaps + dots*4
}

func normalizeAddressForScore(address string) string {
	return strings.ReplaceAll(address, "module.", "")
}

func bestQueryScore(query string, resource terraform.ResourceChange) int {
	if query == "" {
		return 0
	}
	candidate := strings.ToLower(normalizeAddressForScore(resource.Address))
	best := fuzzyScore(query, candidate)
	if resource.ResourceType != "" {
		if score := fuzzyScore(query, strings.ToLower(resource.ResourceType)); score >= 0 && (best == -1 || score < best) {
			best = score
		}
	}
	if resource.ResourceName != "" {
		if score := fuzzyScore(query, strings.ToLower(resource.ResourceName)); score >= 0 && (best == -1 || score < best) {
			best = score
		}
	}
	return best
}

// renderDiff renders a single diff line.
func (r *ResourceList) renderDiff(d diff.MinimalDiff) string {
	symbol := d.Action.GetActionSymbol()
	path := formatPathForDisplay(d.Path)

	var style lipgloss.Style
	var line string

	switch d.Action {
	case diff.DiffAdd:
		style = r.styles.DiffAdd
		line = fmt.Sprintf("  %s %s: %v", symbol, path, formatValue(d.NewValue))
	case diff.DiffRemove:
		style = r.styles.DiffRemove
		line = fmt.Sprintf("  %s %s: %v", symbol, path, formatValue(d.OldValue))
	case diff.DiffChange:
		style = r.styles.DiffChange
		if oldStr, okOld := d.OldValue.(string); okOld {
			if newStr, okNew := d.NewValue.(string); okNew && strings.Contains(oldStr, "\n") && strings.Contains(newStr, "\n") {
				if multi := formatMultilineStringDiff(path, oldStr, newStr); multi != "" {
					return r.renderMultilineDiff(multi)
				}
			}
		}
		line = fmt.Sprintf("  %s %s: %v → %v", symbol, path, formatValue(d.OldValue), formatValue(d.NewValue))
	default:
		style = r.styles.Dimmed
		line = "  ? " + path
	}

	if r.width > 0 {
		line = PadLine(line, r.width)
	}
	return style.Render(line)
}

// getActionStyle returns the appropriate style for an action type.
// Uses terraform's fixed CLI colors for consistency.
func (r *ResourceList) getActionStyle(action terraform.ActionType) lipgloss.Style {
	switch action {
	case terraform.ActionCreate:
		return r.styles.DiffAdd
	case terraform.ActionUpdate:
		return r.styles.DiffChange
	case terraform.ActionDelete:
		return r.styles.DiffRemove
	case terraform.ActionReplace:
		return r.styles.DiffChange
	default:
		return r.styles.Dimmed
	}
}

func (r *ResourceList) getStatusDisplay(resource terraform.ResourceChange) (string, terraform.OperationStatus, string) {
	if !r.showStatus || r.operationState == nil {
		return "", "", ""
	}
	op := r.operationState.GetResourceStatus(resource.Address)
	if op == nil {
		return "[ ]", "", ""
	}

	var badge string
	switch op.Status {
	case terraform.StatusPending:
		badge = "[.]"
	case terraform.StatusInProgress:
		badge = "[>]"
	case terraform.StatusComplete:
		badge = "[*]"
	case terraform.StatusErrored:
		badge = "[x]"
	default:
		badge = "[ ]"
	}

	elapsed := op.ElapsedTime
	if op.Status == terraform.StatusInProgress && !op.StartTime.IsZero() {
		elapsed = time.Since(op.StartTime)
	}
	elapsedText := ""
	if elapsed > 0 {
		elapsedText = formatShortDuration(elapsed)
	}
	return badge, op.Status, elapsedText
}

// getStatusStyle returns the appropriate style for a status.
func (r *ResourceList) getStatusStyle(status terraform.OperationStatus) lipgloss.Style {
	switch status {
	case terraform.StatusPending:
		return r.styles.Dimmed
	case terraform.StatusInProgress:
		return r.styles.DiffChange // yellow
	case terraform.StatusComplete:
		return r.styles.DiffAdd // green
	case terraform.StatusErrored:
		return r.styles.DiffRemove // red
	default:
		return r.styles.Dimmed
	}
}

func formatShortDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// GetSelectedResource returns the currently selected resource.
func (r *ResourceList) GetSelectedResource() *terraform.ResourceChange {
	if r.selectedIndex >= 0 && r.selectedIndex < len(r.visibleItems) {
		item := r.visibleItems[r.selectedIndex]
		if item.kind == itemResource {
			return item.resource
		}
	}
	return nil
}

// GetSelectedIndex returns the currently selected item index (implements SelectablePanel).
func (r *ResourceList) GetSelectedIndex() int {
	return r.selectedIndex
}

// SetSelectedIndex sets the selected item index (implements SelectablePanel).
func (r *ResourceList) SetSelectedIndex(index int) {
	if index >= 0 && index < len(r.visibleItems) {
		r.selectedIndex = index
		r.updateViewport()
	}
}

// ItemCount returns the total number of visible items (implements SelectablePanel).
func (r *ResourceList) ItemCount() int {
	return len(r.visibleItems)
}

type itemKind int

const (
	itemGroup itemKind = iota
	itemResource
)

type listItem struct {
	kind     itemKind
	label    string
	path     string
	count    int
	expanded bool
	resource *terraform.ResourceChange
	indent   int
}

func (r *ResourceList) buildVisibleItems(resources []terraform.ResourceChange) []listItem {
	root := newModuleNode("")
	ungrouped := make([]terraform.ResourceChange, 0)

	for i := range resources {
		resource := resources[i]
		path := modulePath(resource.Address)
		if len(path) == 0 {
			ungrouped = append(ungrouped, resource)
			continue
		}
		root.insert(path, resource)
	}

	items := make([]listItem, 0, len(resources))
	childNames := root.sortedChildNames()
	if r.searchActive {
		sort.Slice(childNames, func(i, j int) bool {
			left := root.children[childNames[i]].minScore(r.matchScores)
			right := root.children[childNames[j]].minScore(r.matchScores)
			if left == right {
				return childNames[i] < childNames[j]
			}
			return left < right
		})
	}
	for _, name := range childNames {
		child := root.children[name]
		items = append(items, r.appendNodeItems(child, 0)...)
	}

	if r.searchActive {
		sort.Slice(ungrouped, func(i, j int) bool {
			left := r.matchScores[ungrouped[i].Address]
			right := r.matchScores[ungrouped[j].Address]
			if left == right {
				return ungrouped[i].Address < ungrouped[j].Address
			}
			return left < right
		})
	} else {
		sort.Slice(ungrouped, func(i, j int) bool {
			return ungrouped[i].Address < ungrouped[j].Address
		})
	}
	for i := range ungrouped {
		items = append(items, listItem{
			kind:     itemResource,
			resource: &ungrouped[i],
			indent:   0,
		})
	}

	return items
}

func (r *ResourceList) appendNodeItems(node *moduleNode, depth int) []listItem {
	items := []listItem{}
	total := node.countTotal()
	// Skip group header for single resources, but not during search to preserve context
	if !r.searchActive && total == 1 && len(node.children) == 0 {
		res := node.resources[0]
		items = append(items, listItem{
			kind:     itemResource,
			resource: &res,
			indent:   (depth + 1) * 2,
		})
		return items
	}

	if _, ok := r.groupExpanded[node.path]; !ok {
		r.groupExpanded[node.path] = r.allExpanded
	}

	items = append(items, listItem{
		kind:     itemGroup,
		label:    "module." + node.name,
		path:     node.path,
		count:    total,
		expanded: r.groupExpanded[node.path],
		indent:   depth * 2,
	})
	if !r.groupExpanded[node.path] {
		return items
	}

	if r.searchActive {
		sort.Slice(node.resources, func(i, j int) bool {
			left := r.matchScores[node.resources[i].Address]
			right := r.matchScores[node.resources[j].Address]
			if left == right {
				return node.resources[i].Address < node.resources[j].Address
			}
			return left < right
		})
	} else {
		sort.Slice(node.resources, func(i, j int) bool {
			return node.resources[i].Address < node.resources[j].Address
		})
	}
	for i := range node.resources {
		items = append(items, listItem{
			kind:     itemResource,
			resource: &node.resources[i],
			indent:   (depth + 1) * 2,
		})
	}

	childNames := node.sortedChildNames()
	if r.searchActive {
		sort.Slice(childNames, func(i, j int) bool {
			left := node.children[childNames[i]].minScore(r.matchScores)
			right := node.children[childNames[j]].minScore(r.matchScores)
			if left == right {
				return childNames[i] < childNames[j]
			}
			return left < right
		})
	}
	for _, name := range childNames {
		child := node.children[name]
		items = append(items, r.appendNodeItems(child, depth+1)...)
	}

	return items
}

func (r *ResourceList) renderGroup(group string, count int, isSelected, expanded bool, indent int, contentWidth int) string {
	indicator := "▶"
	if expanded {
		indicator = "▼"
	}
	prefix := ""
	if indent > 0 {
		prefix = GetPadding(indent)
	}
	line := fmt.Sprintf("%s%s %s (%d)", prefix, indicator, group, count)
	if contentWidth > 0 {
		line = runewidth.Truncate(line, contentWidth, "...")
	}

	if isSelected {
		selectedBg := r.styles.SelectedLineBackground
		line = r.styles.LineItemText.Background(selectedBg).Bold(true).Render(line)
		if contentWidth > 0 {
			line = PadLineWithBg(line, contentWidth, selectedBg)
		}
		return line
	}

	line = r.styles.Dimmed.Bold(true).Render(line)
	if contentWidth > 0 {
		line = PadLine(line, contentWidth)
	}
	return line
}

func (r *ResourceList) ToggleGroup() {
	if r.selectedIndex < 0 || r.selectedIndex >= len(r.visibleItems) {
		return
	}
	item := r.visibleItems[r.selectedIndex]
	if item.kind != itemGroup {
		return
	}
	r.groupExpanded[item.path] = !r.groupExpanded[item.path]
	r.allExpanded = r.computeAllExpanded()
	r.updateViewport()
}

func (r *ResourceList) ToggleAllGroups() {
	if len(r.groupExpanded) == 0 {
		return
	}
	target := !r.allExpanded
	for group := range r.groupExpanded {
		r.groupExpanded[group] = target
	}
	r.allExpanded = target
	r.updateViewport()
}

func (r *ResourceList) computeAllExpanded() bool {
	for _, expanded := range r.groupExpanded {
		if !expanded {
			return false
		}
	}
	return len(r.groupExpanded) > 0
}

func (r *ResourceList) firstResourceIndex() int {
	for i, item := range r.visibleItems {
		if item.kind == itemResource {
			return i
		}
	}
	return -1
}

func resourceMatchesQuery(resource terraform.ResourceChange, query string) bool {
	if query == "" {
		return true
	}
	if fuzzyMatch(query, strings.ToLower(resource.Address)) {
		return true
	}
	if resource.ResourceType != "" && fuzzyMatch(query, strings.ToLower(resource.ResourceType)) {
		return true
	}
	if resource.ResourceName != "" && fuzzyMatch(query, strings.ToLower(resource.ResourceName)) {
		return true
	}
	return false
}

func fuzzyMatch(query, candidate string) bool {
	if query == "" {
		return true
	}
	q := []rune(query)
	c := []rune(candidate)
	if len(q) > len(c) {
		return false
	}
	i := 0
	for _, r := range c {
		if r == q[i] {
			i++
			if i == len(q) {
				return true
			}
		}
	}
	return false
}

func (r *ResourceList) renderMultilineDiff(block string) string {
	lines := strings.Split(block, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		style := r.styles.Dimmed
		switch {
		case strings.HasPrefix(trimmed, "~ "):
			style = r.styles.DiffChange
		case strings.HasPrefix(trimmed, "- "):
			style = r.styles.DiffRemove
		case strings.HasPrefix(trimmed, "+ "):
			style = r.styles.DiffAdd
		}

		if r.width > 0 {
			line = PadLine(line, r.width)
		}
		out = append(out, style.Render(line))
	}
	return strings.Join(out, "\n")
}
