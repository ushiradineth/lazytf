package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/diff"
	"github.com/ushiradineth/tftui/internal/styles"
	"github.com/ushiradineth/tftui/internal/terraform"
)

// DiffViewer renders a side-by-side diff for the selected resource.
type DiffViewer struct {
	styles     *styles.Styles
	diffEngine *diff.Engine
	width      int
	height     int
}

// NewDiffViewer creates a diff viewer.
func NewDiffViewer(s *styles.Styles, engine *diff.Engine) *DiffViewer {
	return &DiffViewer{
		styles:     s,
		diffEngine: engine,
	}
}

// SetSize updates the viewer dimensions.
func (d *DiffViewer) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View renders the diff viewer content.
func (d *DiffViewer) View(resource *terraform.ResourceChange) string {
	content := ""
	if resource == nil {
		content = d.styles.Dimmed.Render("No resource selected")
		return d.pad(content)
	}

	diffs := d.diffEngine.GetResourceDiffs(resource)
	if len(diffs) == 0 {
		content = d.styles.Dimmed.Render("No changes for selected resource")
		return d.pad(content)
	}

	header := d.renderHeader(resource, len(diffs))
	table := d.renderTable(diffs)
	content = lipgloss.JoinVertical(lipgloss.Left, header, table)
	return d.pad(content)
}

func (d *DiffViewer) renderHeader(resource *terraform.ResourceChange, changeCount int) string {
	icon := resource.Action.GetActionIcon()
	title := fmt.Sprintf("%s %s", icon, resource.Address)
	if changeCount == 1 {
		title += "  (1 change)"
	} else {
		title += fmt.Sprintf("  (%d changes)", changeCount)
	}
	return d.styles.Title.Width(d.width).Render(title)
}

func (d *DiffViewer) renderTable(diffs []diff.MinimalDiff) string {
	columns := d.columnWidths()
	header := d.renderRow(columns, d.styles.HelpKey, "Path", "Before", "After")

	var rows []string
	rows = append(rows, header)
	for _, item := range diffs {
		rows = append(rows, d.renderDiffRow(columns, item))
	}

	return strings.Join(rows, "\n")
}

func (d *DiffViewer) renderDiffRow(columns []int, item diff.MinimalDiff) string {
	path := strings.Join(item.Path, ".")

	before := ""
	after := ""
	switch item.Action {
	case diff.DiffAdd:
		after = formatSingleLineValue(item.NewValue)
		return d.renderRow(columns, d.styles.DiffAdd, path, before, after)
	case diff.DiffRemove:
		before = formatSingleLineValue(item.OldValue)
		return d.renderRow(columns, d.styles.DiffRemove, path, before, after)
	case diff.DiffChange:
		before = formatSingleLineValue(item.OldValue)
		after = formatSingleLineValue(item.NewValue)
		return d.renderRow(columns, d.styles.DiffChange, path, before, after)
	default:
		return d.renderRow(columns, d.styles.Dimmed, path, before, after)
	}
}

func (d *DiffViewer) renderRow(columns []int, style lipgloss.Style, path, before, after string) string {
	path = truncateLine(path, columns[0]-1)
	before = truncateLine(before, columns[1]-1)
	after = truncateLine(after, columns[2]-1)

	pathCell := style.Width(columns[0]).MaxWidth(columns[0]).Render(path)
	beforeCell := style.Width(columns[1]).MaxWidth(columns[1]).Render(before)
	afterCell := style.Width(columns[2]).MaxWidth(columns[2]).Render(after)

	return lipgloss.JoinHorizontal(lipgloss.Left, pathCell, beforeCell, afterCell)
}

func (d *DiffViewer) columnWidths() []int {
	if d.width <= 0 {
		return []int{20, 20, 20}
	}

	pathWidth := int(float64(d.width) * 0.35)
	beforeWidth := int(float64(d.width) * 0.32)
	afterWidth := d.width - pathWidth - beforeWidth

	if pathWidth < 16 {
		pathWidth = 16
	}
	if beforeWidth < 14 {
		beforeWidth = 14
	}
	if afterWidth < 14 {
		afterWidth = 14
	}

	remaining := d.width - pathWidth - beforeWidth - afterWidth
	if remaining != 0 {
		afterWidth += remaining
	}

	return []int{pathWidth, beforeWidth, afterWidth}
}

func (d *DiffViewer) pad(content string) string {
	if d.width <= 0 || d.height <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(d.width).Height(d.height).Render(content)
}

func formatSingleLineValue(val interface{}) string {
	if s, ok := val.(string); ok {
		if strings.Contains(s, "\n") {
			s = strings.ReplaceAll(s, "\n", `\n`)
			s = truncateMiddle(s, 240)
			return fmt.Sprintf(`"%s"`, s)
		}
	}
	return formatValue(val)
}

func truncateMiddle(s string, max int) string {
	if len(s) <= max || max <= 3 {
		return s
	}
	head := max / 2
	tail := max - head - 3
	if tail < 1 {
		tail = 1
	}
	return s[:head] + "..." + s[len(s)-tail:]
}
