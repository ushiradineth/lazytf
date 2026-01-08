package components

import (
	"fmt"
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
	filtered := r.getFilteredResources()
	if r.selectedIndex < len(filtered)-1 {
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
		r.viewport.SetContent(r.styles.Dimmed.Render("No resources to display"))
		return
	}

	if r.selectedIndex >= len(filtered) {
		r.selectedIndex = len(filtered) - 1
	}

	var content strings.Builder

	for i, resource := range filtered {
		isSelected := i == r.selectedIndex

		// Render the resource
		content.WriteString(r.renderResource(resource, isSelected))
		content.WriteString("\n")
	}

	r.viewport.SetContent(content.String())
}

// renderResource renders a single resource line.
func (r *ResourceList) renderResource(resource terraform.ResourceChange, isSelected bool) string {
	var output strings.Builder

	// Get action style and icon
	actionIcon := resource.Action.GetActionIcon()
	actionStyle := r.getActionStyle(resource.Action)

	// Calculate change count
	changeCount := r.diffEngine.CountChanges(&resource)

	// Render the header line
	headerBase := fmt.Sprintf("%s %s", actionIcon, resource.Address)
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
	filtered := r.getFilteredResources()
	if r.selectedIndex >= 0 && r.selectedIndex < len(filtered) {
		return &filtered[r.selectedIndex]
	}
	return nil
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
