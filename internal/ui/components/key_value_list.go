package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// KeyValueRow represents a single label/value line.
type KeyValueRow struct {
	Label      string
	Value      string
	LabelStyle lipgloss.Style
	ValueStyle lipgloss.Style
}

// KeyValueList renders rows within a fixed width, truncating values as needed.
type KeyValueList struct {
	width int
	rows  []KeyValueRow
}

// NewKeyValueList creates a new key/value list renderer.
func NewKeyValueList() *KeyValueList {
	return &KeyValueList{}
}

// SetWidth updates the available render width.
func (k *KeyValueList) SetWidth(width int) {
	k.width = width
}

// SetRows replaces the rows to render.
func (k *KeyValueList) SetRows(rows []KeyValueRow) {
	k.rows = rows
}

// View renders the key/value rows.
func (k *KeyValueList) View() string {
	return renderKeyValueRows(k.width, k.rows)
}

func renderKeyValueRows(width int, rows []KeyValueRow) string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, renderKeyValueRow(width, row))
	}
	return strings.Join(lines, "\n")
}

func renderKeyValueRow(width int, row KeyValueRow) string {
	if width <= 0 {
		return row.LabelStyle.Render(row.Label) + row.ValueStyle.Render(row.Value)
	}

	labelWidth := runewidth.StringWidth(row.Label)
	if labelWidth >= width {
		truncatedLabel := runewidth.Truncate(row.Label, width, "")
		return row.LabelStyle.Render(truncatedLabel)
	}

	available := width - labelWidth
	value := row.Value
	if runewidth.StringWidth(value) > available {
		value = runewidth.Truncate(value, available, "...")
	}

	return row.LabelStyle.Render(row.Label) + row.ValueStyle.Render(value)
}
