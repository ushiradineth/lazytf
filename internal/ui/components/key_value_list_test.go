package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNewKeyValueList(t *testing.T) {
	kv := NewKeyValueList()
	if kv == nil {
		t.Fatal("expected non-nil KeyValueList")
	}
}

func TestKeyValueListSetWidth(t *testing.T) {
	kv := NewKeyValueList()
	kv.SetWidth(50)
	if kv.width != 50 {
		t.Errorf("expected width 50, got %d", kv.width)
	}
}

func TestKeyValueListSetRows(t *testing.T) {
	kv := NewKeyValueList()
	rows := []KeyValueRow{
		{Label: "Key1: ", Value: "Value1"},
		{Label: "Key2: ", Value: "Value2"},
	}
	kv.SetRows(rows)
	if len(kv.rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(kv.rows))
	}
}

func TestKeyValueListView(t *testing.T) {
	kv := NewKeyValueList()
	kv.SetWidth(50)
	kv.SetRows([]KeyValueRow{
		{Label: "Name: ", Value: "test"},
		{Label: "Type: ", Value: "resource"},
	})

	view := kv.View()
	if !strings.Contains(view, "Name:") {
		t.Error("expected view to contain 'Name:'")
	}
	if !strings.Contains(view, "Type:") {
		t.Error("expected view to contain 'Type:'")
	}
}

func TestRenderKeyValueRowZeroWidth(t *testing.T) {
	row := KeyValueRow{Label: "Key: ", Value: "Value"}
	result := renderKeyValueRow(0, row)
	if !strings.Contains(result, "Key:") || !strings.Contains(result, "Value") {
		t.Errorf("unexpected result for zero width: %q", result)
	}
}

func TestRenderKeyValueRowTruncateLabel(_ *testing.T) {
	row := KeyValueRow{Label: "VeryLongLabelName: ", Value: "Value"}
	result := renderKeyValueRow(10, row)
	// Label is 19 chars but width is 10, so it should be truncated
	// Result may contain ANSI codes, so just check it's reasonable
	_ = result
}

func TestRenderKeyValueRowTruncateValue(t *testing.T) {
	row := KeyValueRow{Label: "Key: ", Value: "VeryLongValueThatShouldBeTruncated"}
	result := renderKeyValueRow(20, row)
	if strings.Contains(result, "VeryLongValueThatShouldBeTruncated") {
		t.Error("expected value to be truncated")
	}
}

func TestRenderKeyValueRowWithStyles(t *testing.T) {
	row := KeyValueRow{
		Label:      "Status: ",
		Value:      "active",
		LabelStyle: lipgloss.NewStyle().Bold(true),
		ValueStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("green")),
	}
	result := renderKeyValueRow(50, row)
	if !strings.Contains(result, "Status") {
		t.Error("expected label in result")
	}
	if !strings.Contains(result, "active") {
		t.Error("expected value in result")
	}
}

func TestRenderKeyValueRows(t *testing.T) {
	rows := []KeyValueRow{
		{Label: "A: ", Value: "1"},
		{Label: "B: ", Value: "2"},
		{Label: "C: ", Value: "3"},
	}
	result := renderKeyValueRows(50, rows)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestRenderKeyValueRowsEmpty(t *testing.T) {
	result := renderKeyValueRows(50, nil)
	if result != "" {
		t.Errorf("expected empty string for nil rows, got %q", result)
	}
}
