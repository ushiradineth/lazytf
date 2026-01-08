package components

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/diff"
	"github.com/ushiradineth/tftui/internal/styles"
	"github.com/ushiradineth/tftui/internal/terraform"
)

// ResourceList displays a list of resources with expand/collapse functionality
type ResourceList struct {
	viewport      viewport.Model
	resources     []terraform.ResourceChange
	expandedMap   map[string]bool // resource address -> expanded state
	selectedIndex int
	diffEngine    *diff.Engine
	styles        *styles.Styles
	width         int
	height        int
	filterActions map[terraform.ActionType]bool
}

// NewResourceList creates a new resource list component
func NewResourceList(s *styles.Styles) *ResourceList {
	vp := viewport.New(0, 0)

	return &ResourceList{
		viewport:      vp,
		resources:     []terraform.ResourceChange{},
		expandedMap:   make(map[string]bool),
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

// ToggleSelected toggles the expanded state of the currently selected resource
func (r *ResourceList) ToggleSelected() {
	filtered := r.getFilteredResources()
	if r.selectedIndex >= 0 && r.selectedIndex < len(filtered) {
		resource := filtered[r.selectedIndex]
		r.expandedMap[resource.Address] = !r.expandedMap[resource.Address]
		r.updateViewport()
	}
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			r.ToggleSelected()
		}
	}

	r.viewport, cmd = r.viewport.Update(msg)
	return r, cmd
}

// View renders the resource list
func (r *ResourceList) View() string {
	return r.viewport.View()
}

// getFilteredResources returns resources that pass the current filter
func (r *ResourceList) getFilteredResources() []terraform.ResourceChange {
	filtered := []terraform.ResourceChange{}
	for _, resource := range r.resources {
		if r.filterActions[resource.Action] {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// updateViewport regenerates the viewport content based on current state
func (r *ResourceList) updateViewport() {
	filtered := r.getFilteredResources()
	if len(filtered) == 0 {
		r.viewport.SetContent(r.styles.Dimmed.Render("No resources to display"))
		return
	}

	var content strings.Builder

	for i, resource := range filtered {
		isSelected := i == r.selectedIndex
		isExpanded := r.expandedMap[resource.Address]

		// Render the resource
		content.WriteString(r.renderResource(resource, isSelected, isExpanded))
		content.WriteString("\n")
	}

	r.viewport.SetContent(content.String())
}

// renderResource renders a single resource (collapsed or expanded)
func (r *ResourceList) renderResource(resource terraform.ResourceChange, isSelected, isExpanded bool) string {
	var output strings.Builder

	// Get action style and icon
	actionIcon := resource.Action.GetActionIcon()
	actionStyle := r.getActionStyle(resource.Action)

	// Calculate change count
	changeCount := r.diffEngine.CountChanges(&resource)

	// Render the header line
	headerLine := fmt.Sprintf("%s %s", actionIcon, resource.Address)
	if !isExpanded && changeCount > 0 {
		headerLine += r.styles.Dimmed.Render(fmt.Sprintf("  (%d changes)", changeCount))
	}

	if isSelected {
		headerLine = r.styles.Selected.Render(headerLine)
	} else {
		headerLine = actionStyle.Render(headerLine)
	}

	output.WriteString(headerLine)

	// If expanded, show the minimal diff
	if isExpanded {
		diffs := r.diffEngine.GetResourceDiffs(&resource)
		if len(diffs) > 0 {
			output.WriteString("\n")
			for _, d := range diffs {
				diffLine := r.renderDiff(d)
				output.WriteString(diffLine)
				output.WriteString("\n")
			}
		}
	}

	return output.String()
}

// renderDiff renders a single diff line
func (r *ResourceList) renderDiff(d diff.MinimalDiff) string {
	symbol := d.Action.GetActionSymbol()
	path := strings.Join(d.Path, ".")

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
					return style.Render(multi)
				}
			}
		}
		line = fmt.Sprintf("  %s %s: %v → %v", symbol, path, formatValue(d.OldValue), formatValue(d.NewValue))
	default:
		style = r.styles.Dimmed
		line = fmt.Sprintf("  ? %s", path)
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

// formatValue formats a value for display
func formatValue(val interface{}) string {
	if val == nil {
		return "(null)"
	}

	if s, ok := val.(string); ok {
		if len(s) > 200 {
			return fmt.Sprintf("%q...", s[:197])
		}
		return fmt.Sprintf(`"%s"`, s)
	}

	if _, ok := val.(diff.UnknownValue); ok {
		return "(known after apply)"
	}

	// For complex types, use a compact representation.
	if isMap(val) {
		return "{...}"
	}
	if isList(val) {
		if asList := interfaceToList(val); len(asList) == 1 {
			if s, ok := asList[0].(string); ok {
				return formatValue(s)
			}
		}
		return "[...]"
	}

	return fmt.Sprintf("%v", val)
}

func isMap(val interface{}) bool {
	if val == nil {
		return false
	}
	return reflect.TypeOf(val).Kind() == reflect.Map
}

func isList(val interface{}) bool {
	if val == nil {
		return false
	}
	kind := reflect.TypeOf(val).Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

func interfaceToList(val interface{}) []interface{} {
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}

	result := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}
	return result
}

func formatMultilineStringDiff(path, before, after string) string {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	if len(beforeLines) != len(afterLines) {
		return ""
	}

	diffIndexes := make([]int, 0, 4)
	for i := range beforeLines {
		if beforeLines[i] != afterLines[i] {
			diffIndexes = append(diffIndexes, i)
			if len(diffIndexes) >= 4 {
				break
			}
		}
	}
	if len(diffIndexes) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  ~ %s:", path))
	for _, idx := range diffIndexes {
		oldLine := stripListMarker(strings.TrimSpace(beforeLines[idx]))
		newLine := stripListMarker(strings.TrimSpace(afterLines[idx]))
		b.WriteString("\n")
		b.WriteString("    - ")
		b.WriteString(truncateLine(oldLine, 140))
		b.WriteString("\n")
		b.WriteString("    + ")
		b.WriteString(truncateLine(newLine, 140))
	}
	return b.String()
}

func truncateLine(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func stripListMarker(line string) string {
	if strings.HasPrefix(line, "- ") {
		return strings.TrimSpace(line[2:])
	}
	return line
}

// GetSelectedResource returns the currently selected resource
func (r *ResourceList) GetSelectedResource() *terraform.ResourceChange {
	filtered := r.getFilteredResources()
	if r.selectedIndex >= 0 && r.selectedIndex < len(filtered) {
		return &filtered[r.selectedIndex]
	}
	return nil
}
