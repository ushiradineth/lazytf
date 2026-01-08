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
		return d.pad("")
	}

	diffs := d.diffEngine.GetResourceDiffs(resource)
	if len(diffs) == 0 {
		content = d.styles.Dimmed.Render("No changes for selected resource")
		return d.pad(content)
	}

	header := d.renderHeader(resource, diffs)
	var body string
	if resource.Action == terraform.ActionCreate || resource.Action == terraform.ActionDelete {
		body = d.renderCompactList(diffs, resource.Change)
	} else if hasMultilineDiff(diffs) {
		body = d.renderBlocks(diffs, resource.Change)
	} else {
		body = d.renderCompactList(diffs, resource.Change)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, header, body)
	return d.pad(content)
}

func (d *DiffViewer) renderHeader(resource *terraform.ResourceChange, diffs []diff.MinimalDiff) string {
	changeCount := len(diffs)
	icon := resource.Action.GetActionIcon()
	actionLabel := actionLabel(resource.Action)
	if actionLabel != "" {
		actionLabel = " [" + actionLabel + "]"
	}
	title := fmt.Sprintf("%s %s%s", icon, resource.Address, actionLabel)
	if changeCount == 1 {
		title += "  (1 change)"
	} else {
		title += fmt.Sprintf("  (%d changes)", changeCount)
	}
	return d.styles.Title.Width(d.width).Render(title)
}

func (d *DiffViewer) renderTable(diffs []diff.MinimalDiff, change *terraform.Change) string {
	var rows []string
	for _, item := range diffs {
		rows = append(rows, d.renderInlineChange(item, change))
	}
	return strings.Join(rows, "\n")
}

func (d *DiffViewer) renderBlocks(diffs []diff.MinimalDiff, change *terraform.Change) string {
	var rows []string
	prevRoot := ""
	lastSpacer := false
	for i, item := range diffs {
		root := ""
		if len(item.Path) > 0 {
			root = item.Path[0]
		}
		if prevRoot != "" && root != prevRoot && !lastSpacer {
			rows = append(rows, "")
			lastSpacer = true
		}
		switch {
		case isMultilineChange(item):
			rows = append(rows, d.renderMultilineBlock(item, change)...)
			if i < len(diffs)-1 {
				rows = append(rows, "")
				lastSpacer = true
			}
		default:
			rows = append(rows, d.renderInlineChange(item, change))
			lastSpacer = false
		}
		prevRoot = root
	}
	return strings.Join(rows, "\n")
}

func (d *DiffViewer) renderCompactList(diffs []diff.MinimalDiff, change *terraform.Change) string {
	var rows []string
	for _, item := range diffs {
		rows = append(rows, d.renderInlineChange(item, change))
	}
	return strings.Join(rows, "\n")
}

func (d *DiffViewer) renderInlineChange(item diff.MinimalDiff, change *terraform.Change) string {
	path := formatPathForDisplay(item.Path)
	symbol := item.Action.GetActionSymbol()
	var style lipgloss.Style
	line := ""
	marker := replaceMarker(item.Path, change)
	markerPlain := ""
	markerStyled := ""
	if marker != "" {
		markerPlain = "  " + marker
		markerStyled = d.styles.Comment.Render(markerPlain)
	}

	switch item.Action {
	case diff.DiffAdd:
		style = d.styles.DiffAdd
		line = fmt.Sprintf("%s %s: %s", symbol, path, formatSingleLineValue(item.NewValue))
	case diff.DiffRemove:
		style = d.styles.DiffRemove
		line = fmt.Sprintf("%s %s: %s", symbol, path, formatSingleLineValue(item.OldValue))
	case diff.DiffChange:
		style = d.styles.DiffChange
		line = fmt.Sprintf("%s %s: %s → %s", symbol, path, formatSingleLineValue(item.OldValue), formatSingleLineValue(item.NewValue))
	default:
		style = d.styles.Dimmed
		line = fmt.Sprintf("? %s", path)
	}

	if d.width > 0 {
		line = padLine(line, d.width)
	}
	if markerStyled != "" {
		return style.Render(strings.TrimRight(line, " ")) + markerStyled
	}
	return style.Render(line)
}

func (d *DiffViewer) renderMultilineBlock(item diff.MinimalDiff, change *terraform.Change) []string {
	path := formatPathForDisplay(item.Path)
	symbol := item.Action.GetActionSymbol()
	marker := replaceMarker(item.Path, change)
	header := fmt.Sprintf("%s %s", symbol, path)
	if d.width > 0 {
		header = padLine(header, d.width)
	}
	header = d.styles.DiffChange.Render(header)
	if marker != "" {
		header = header + d.styles.Comment.Render("  "+marker)
	}

	oldStr, _ := item.OldValue.(string)
	newStr, _ := item.NewValue.(string)
	lines := buildContextDiff(oldStr, newStr, 2)

	var output []string
	output = append(output, header)
	for _, line := range lines {
		prefix := linePrefix(line)
		line = "  " + line
		if d.width > 0 {
			line = padLine(line, d.width)
		}
		switch prefix {
		case "-":
			output = append(output, d.styles.DiffRemove.Render(line))
		case "+":
			output = append(output, d.styles.DiffAdd.Render(line))
		case ".":
			output = append(output, d.styles.Dimmed.Render(line))
		default:
			output = append(output, d.styles.Dimmed.Render(line))
		}
	}
	return output
}

func (d *DiffViewer) renderDiffRow(columns []int, item diff.MinimalDiff, change *terraform.Change) string {
	path := formatPathForDisplay(item.Path) + replaceMarker(item.Path, change)
	symbol := item.Action.GetActionSymbol()

	before := ""
	after := ""
	switch item.Action {
	case diff.DiffAdd:
		after = formatSingleLineValue(item.NewValue)
		return d.renderRow(columns, d.styles.DiffAdd, symbol, path, before, after)
	case diff.DiffRemove:
		before = formatSingleLineValue(item.OldValue)
		return d.renderRow(columns, d.styles.DiffRemove, symbol, path, before, after)
	case diff.DiffChange:
		before = formatSingleLineValue(item.OldValue)
		after = formatSingleLineValue(item.NewValue)
		return d.renderRow(columns, d.styles.DiffChange, symbol, path, before, after)
	default:
		return d.renderRow(columns, d.styles.Dimmed, symbol, path, before, after)
	}
}

func (d *DiffViewer) renderRow(columns []int, style lipgloss.Style, symbol, path, before, after string) string {
	symbol = truncateLine(symbol, columns[0]-1)
	path = truncateLine(path, columns[1]-1)
	before = truncateLine(before, columns[2]-1)
	after = truncateLine(after, columns[3]-1)

	symbolCell := style.Width(columns[0]).MaxWidth(columns[0]).Render(symbol)
	pathCell := style.Width(columns[1]).MaxWidth(columns[1]).Render(path)
	beforeCell := style.Width(columns[2]).MaxWidth(columns[2]).Render(before)
	afterCell := style.Width(columns[3]).MaxWidth(columns[3]).Render(after)

	return lipgloss.JoinHorizontal(lipgloss.Left, symbolCell, pathCell, beforeCell, afterCell)
}

func (d *DiffViewer) columnWidths() []int {
	if d.width <= 0 {
		return []int{2, 18, 18, 18}
	}

	symbolWidth := 2
	pathWidth := int(float64(d.width) * 0.32)
	beforeWidth := int(float64(d.width) * 0.31)
	afterWidth := d.width - symbolWidth - pathWidth - beforeWidth

	if pathWidth < 16 {
		pathWidth = 16
	}
	if beforeWidth < 14 {
		beforeWidth = 14
	}
	if afterWidth < 14 {
		afterWidth = 14
	}

	remaining := d.width - symbolWidth - pathWidth - beforeWidth - afterWidth
	if remaining != 0 {
		afterWidth += remaining
	}

	return []int{symbolWidth, pathWidth, beforeWidth, afterWidth}
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

func hasMultilineDiff(diffs []diff.MinimalDiff) bool {
	for _, item := range diffs {
		if isMultilineChange(item) {
			return true
		}
	}
	return false
}

func isMultilineChange(item diff.MinimalDiff) bool {
	if item.Action != diff.DiffChange {
		return false
	}
	oldStr, okOld := item.OldValue.(string)
	newStr, okNew := item.NewValue.(string)
	return okOld && okNew && strings.Contains(oldStr, "\n") && strings.Contains(newStr, "\n")
}

func buildContextDiff(before, after string, context int) []string {
	beforeLines := strings.Split(strings.TrimSuffix(before, "\n"), "\n")
	afterLines := strings.Split(strings.TrimSuffix(after, "\n"), "\n")
	maxLen := len(beforeLines)
	if len(afterLines) > maxLen {
		maxLen = len(afterLines)
	}

	diffIdx := []int{}
	for i := 0; i < maxLen; i++ {
		oldLine := lineAt(beforeLines, i)
		newLine := lineAt(afterLines, i)
		if oldLine != newLine {
			diffIdx = append(diffIdx, i)
		}
	}
	if len(diffIdx) == 0 {
		return []string{"  (no diff)"}
	}

	type span struct{ start, end int }
	spans := []span{}
	for _, idx := range diffIdx {
		start := idx - context
		if start < 0 {
			start = 0
		}
		end := idx + context
		if end > maxLen-1 {
			end = maxLen - 1
		}
		if len(spans) == 0 || start > spans[len(spans)-1].end+1 {
			spans = append(spans, span{start: start, end: end})
		} else if end > spans[len(spans)-1].end {
			spans[len(spans)-1].end = end
		}
	}

	var out []string
	lastEnd := -1
	for _, sp := range spans {
		if lastEnd >= 0 && sp.start > lastEnd+1 {
			out = append(out, "  ...")
		}
		for i := sp.start; i <= sp.end; i++ {
			oldLine := lineAt(beforeLines, i)
			newLine := lineAt(afterLines, i)
			if oldLine == newLine {
				out = append(out, "  "+oldLine)
				continue
			}
			if oldLine != "" {
				out = append(out, "- "+oldLine)
			}
			if newLine != "" {
				out = append(out, "+ "+newLine)
			}
		}
		lastEnd = sp.end
	}
	return out
}

func lineAt(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return ""
	}
	return lines[index]
}

func linePrefix(line string) string {
	if strings.HasPrefix(line, "- ") {
		return "-"
	}
	if strings.HasPrefix(line, "+ ") {
		return "+"
	}
	if strings.HasPrefix(line, "  ...") {
		return "."
	}
	return ""
}

func actionLabel(action terraform.ActionType) string {
	switch action {
	case terraform.ActionCreate:
		return "create"
	case terraform.ActionUpdate:
		return "update"
	case terraform.ActionDelete:
		return "destroy"
	case terraform.ActionReplace:
		return "replace"
	default:
		return ""
	}
}

func replaceMarker(path []string, change *terraform.Change) string {
	if change == nil || len(change.ReplacePaths) == 0 {
		return ""
	}
	for _, rp := range change.ReplacePaths {
		if pathMatchesReplace(path, rp) {
			return " # forces replacement"
		}
	}
	return ""
}

func pathMatchesReplace(path, replace []string) bool {
	if len(replace) == 0 || len(path) < len(replace) {
		return false
	}
	for i := range replace {
		if path[i] != replace[i] {
			return false
		}
	}
	return true
}
