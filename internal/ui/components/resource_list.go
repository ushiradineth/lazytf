package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/ushiradineth/tftui/internal/diff"
	"github.com/ushiradineth/tftui/internal/styles"
	"github.com/ushiradineth/tftui/internal/terraform"
)

// ResourceList displays a list of resources.
type ResourceList struct {
	viewport      viewport.Model
	resources     []terraform.ResourceChange
	selectedIndex int
	diffEngine    *diff.Engine
	styles        *styles.Styles
	width         int
	height        int
	filterActions map[terraform.ActionType]bool
	searchQuery   string
	groupExpanded map[string]bool
	visibleItems  []listItem
	allExpanded   bool
}

// NewResourceList creates a new resource list component
func NewResourceList(s *styles.Styles) *ResourceList {
	vp := viewport.New(0, 0)

	return &ResourceList{
		viewport:      vp,
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

// SetSize sets the dimensions of the resource list
func (r *ResourceList) SetSize(width, height int) {
	r.width = width
	r.height = height
	r.viewport.Width = width
	r.viewport.Height = height
	r.updateViewport()
}

// SetResources updates the list of resources to display
func (r *ResourceList) SetResources(resources []terraform.ResourceChange) {
	r.resources = resources
	r.selectedIndex = 0
	r.updateViewport()
}

// SetFilter sets which action types to display
func (r *ResourceList) SetFilter(actionType terraform.ActionType, enabled bool) {
	r.filterActions[actionType] = enabled
	r.updateViewport()
}

// SetSearchQuery sets the current search query for filtering resources.
func (r *ResourceList) SetSearchQuery(query string) {
	r.searchQuery = strings.TrimSpace(query)
	r.updateViewport()
}

// MoveUp moves the selection up
func (r *ResourceList) MoveUp() {
	if r.selectedIndex > 0 {
		r.selectedIndex--
		r.updateViewport()
	}
}

// MoveDown moves the selection down
func (r *ResourceList) MoveDown() {
	if r.selectedIndex < len(r.visibleItems)-1 {
		r.selectedIndex++
		r.updateViewport()
	}
}

// Init initializes the component
func (r *ResourceList) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (r *ResourceList) Update(msg tea.Msg) (*ResourceList, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			r.MoveUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			r.MoveDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			r.ToggleGroup()
		}
	}

	r.viewport, cmd = r.viewport.Update(msg)
	return r, cmd
}

// View renders the resource list
func (r *ResourceList) View() string {
	view := r.viewport.View()
	if r.width > 0 && r.height > 0 {
		return lipgloss.NewStyle().Width(r.width).Height(r.height).Render(view)
	}
	return view
}

// getFilteredResources returns resources that pass the current filter
func (r *ResourceList) getFilteredResources() []terraform.ResourceChange {
	filtered := []terraform.ResourceChange{}
	query := strings.ToLower(r.searchQuery)
	for _, resource := range r.resources {
		if !r.filterActions[resource.Action] {
			continue
		}
		if query == "" || resourceMatchesQuery(resource, query) {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// updateViewport regenerates the viewport content based on current state
func (r *ResourceList) updateViewport() {
	filtered := r.getFilteredResources()
	if len(filtered) == 0 {
		r.selectedIndex = 0
		r.visibleItems = nil
		r.viewport.SetContent(r.styles.Dimmed.Render("No resources to display"))
		return
	}

	r.visibleItems = r.buildVisibleItems(filtered)
	if len(r.visibleItems) == 0 {
		r.selectedIndex = 0
		r.viewport.SetContent(r.styles.Dimmed.Render("No resources to display"))
		return
	}

	if r.selectedIndex >= len(r.visibleItems) {
		r.selectedIndex = len(r.visibleItems) - 1
	}

	var content strings.Builder
	for i, item := range r.visibleItems {
		isSelected := i == r.selectedIndex
		switch item.kind {
		case itemGroup:
			content.WriteString(r.renderGroup(item.label, item.count, isSelected, item.expanded, item.indent))
		case itemResource:
			content.WriteString(r.renderResource(*item.resource, isSelected, item.indent))
		}
		content.WriteString("\n")
	}

	r.viewport.SetContent(content.String())
}

// renderResource renders a single resource line.
func (r *ResourceList) renderResource(resource terraform.ResourceChange, isSelected bool, indent int) string {
	var output strings.Builder

	// Get action style and icon
	actionIcon := resource.Action.GetActionIcon()
	actionStyle := r.getActionStyle(resource.Action)

	// Calculate change count
	changeCount := r.diffEngine.CountChanges(&resource)

	// Render the header line
	prefix := ""
	if indent > 0 {
		prefix = strings.Repeat(" ", indent)
	}
	address := resource.Address
	if indent > 0 {
		address = trimModulePrefix(resource.Address, indent)
	}
	headerBase := fmt.Sprintf("%s%s %s", prefix, actionIcon, address)
	headerSuffix := ""
	if changeCount > 0 {
		headerSuffix = fmt.Sprintf("  (%d changes)", changeCount)
	}

	if isSelected {
		headerLine := headerBase + headerSuffix
		if r.width > 0 {
			headerLine = runewidth.Truncate(headerLine, r.width, "...")
		}
		headerLine = r.styles.Selected.Render(headerLine)
		if r.width > 0 {
			headerLine = padAfterStyled(headerLine, r.width)
		}
		output.WriteString(headerLine)
	} else {
		headerLine := headerBase
		if headerSuffix != "" {
			headerLine += r.styles.Dimmed.Render(headerSuffix)
		}
		if r.width > 0 {
			headerLine = padLine(headerLine, r.width)
		}
		headerLine = actionStyle.Render(headerLine)
		output.WriteString(headerLine)
	}

	return output.String()
}

func trimModulePrefix(address string, indent int) string {
	if indent <= 0 {
		return address
	}
	for i := 0; i < indent/2; i++ {
		if strings.HasPrefix(address, "module.") {
			parts := strings.SplitN(address, ".", 3)
			if len(parts) == 3 {
				address = parts[2]
			} else {
				break
			}
		}
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

// renderDiff renders a single diff line
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
		line = fmt.Sprintf("  ? %s", path)
	}

	if r.width > 0 {
		line = padLine(line, r.width)
	}
	return style.Render(line)
}

// getActionStyle returns the appropriate style for an action type
func (r *ResourceList) getActionStyle(action terraform.ActionType) lipgloss.Style {
	switch action {
	case terraform.ActionCreate:
		return r.styles.Create
	case terraform.ActionUpdate:
		return r.styles.Update
	case terraform.ActionDelete:
		return r.styles.Delete
	case terraform.ActionReplace:
		return r.styles.Replace
	default:
		return r.styles.NoChange
	}
}

// GetSelectedResource returns the currently selected resource
func (r *ResourceList) GetSelectedResource() *terraform.ResourceChange {
	if r.selectedIndex >= 0 && r.selectedIndex < len(r.visibleItems) {
		item := r.visibleItems[r.selectedIndex]
		if item.kind == itemResource {
			return item.resource
		}
	}
	return nil
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
	for _, name := range childNames {
		child := root.children[name]
		items = append(items, r.appendNodeItems(child, 0)...)
	}

	sort.Slice(ungrouped, func(i, j int) bool {
		return ungrouped[i].Address < ungrouped[j].Address
	})
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
	if total == 1 && len(node.children) == 0 {
		res := node.resources[0]
		items = append(items, listItem{
			kind:     itemResource,
			resource: &res,
			indent:   depth * 2,
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

	sort.Slice(node.resources, func(i, j int) bool {
		return node.resources[i].Address < node.resources[j].Address
	})
	for i := range node.resources {
		items = append(items, listItem{
			kind:     itemResource,
			resource: &node.resources[i],
			indent:   (depth + 1) * 2,
		})
	}

	childNames := node.sortedChildNames()
	for _, name := range childNames {
		child := node.children[name]
		items = append(items, r.appendNodeItems(child, depth+1)...)
	}

	return items
}

func (r *ResourceList) renderGroup(group string, count int, isSelected, expanded bool, indent int) string {
	indicator := "▶"
	if expanded {
		indicator = "▼"
	}
	prefix := ""
	if indent > 0 {
		prefix = strings.Repeat(" ", indent)
	}
	line := fmt.Sprintf("%s%s %s (%d)", prefix, indicator, group, count)
	if r.width > 0 {
		line = runewidth.Truncate(line, r.width, "...")
	}

	if isSelected {
		line = r.styles.Selected.Render(line)
		if r.width > 0 {
			line = padAfterStyled(line, r.width)
		}
		return line
	}

	line = r.styles.Dimmed.Bold(true).Render(line)
	if r.width > 0 {
		line = padAfterStyled(line, r.width)
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
	haystack := strings.ToLower(resource.Address)
	if resource.ResourceType != "" || resource.ResourceName != "" {
		haystack += " " + strings.ToLower(resource.ResourceType) + " " + strings.ToLower(resource.ResourceName)
	}
	return fuzzyMatch(query, haystack)
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

func padLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	truncated := runewidth.Truncate(line, width, "...")
	pad := width - runewidth.StringWidth(truncated)
	if pad <= 0 {
		return truncated
	}
	return truncated + strings.Repeat(" ", pad)
}

func padMultiline(text string, width int) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = padLine(line, width)
	}
	return strings.Join(lines, "\n")
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
			line = padLine(line, r.width)
		}
		out = append(out, style.Render(line))
	}
	return strings.Join(out, "\n")
}

func padAfterStyled(styled string, width int) string {
	if width <= 0 {
		return styled
	}
	visible := lipgloss.Width(styled)
	if visible >= width {
		return styled
	}
	return styled + strings.Repeat(" ", width-visible)
}
