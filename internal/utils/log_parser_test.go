package utils

import (
	"strings"
	"testing"
)

func TestFormatLogOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"whitespace only", "   \n\t\n  ", ""},
		{"plain text single line", "hello world", "hello world"},
		{"plain text multi line", "line1\nline2\nline3", "line1\nline2\nline3"},
		{"skip empty lines", "line1\n\n\nline2", "line1\nline2"},
		{"trailing newlines trimmed", "line1\nline2\n\n\n", "line1\nline2"},
		{
			"json log with @message",
			`{"@timestamp":"2024-01-15T10:30:00Z","@message":"Planning complete"}`,
			"[2024-01-15 10:30:00 +00:00] Planning complete",
		},
		{
			"json log with message",
			`{"timestamp":"2024-01-15T10:30:00Z","message":"Apply started"}`,
			"[2024-01-15 10:30:00 +00:00] Apply started",
		},
		{
			"json log without timestamp",
			`{"@message":"No timestamp here"}`,
			"No timestamp here",
		},
		{
			"json log with RFC3339Nano timestamp",
			`{"@timestamp":"2024-01-15T10:30:00.123456789Z","@message":"Nano time"}`,
			"[2024-01-15 10:30:00 +00:00] Nano time",
		},
		{
			"mixed json and plain text",
			"Plain line\n{\"@message\":\"JSON line\"}\nAnother plain",
			"Plain line\nJSON line\nAnother plain",
		},
		{
			"invalid json treated as plain text",
			`{"invalid json`,
			`{"invalid json`,
		},
		{
			"json without message field",
			`{"other":"data","value":123}`,
			`{"other":"data","value":123}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLogOutput(tt.input)
			if result != tt.expected {
				t.Errorf("FormatLogOutput(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatLogTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"RFC3339",
			"2024-01-15T10:30:00Z",
			"2024-01-15 10:30:00 +00:00",
		},
		{
			"RFC3339 with timezone",
			"2024-01-15T10:30:00-05:00",
			"2024-01-15 10:30:00 -05:00",
		},
		{
			"RFC3339Nano",
			"2024-01-15T10:30:00.123456789Z",
			"2024-01-15 10:30:00 +00:00",
		},
		{
			"invalid format returns original",
			"not-a-timestamp",
			"not-a-timestamp",
		},
		{
			"partial timestamp",
			"2024-01-15",
			"2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLogTimestamp(tt.input)
			if result != tt.expected {
				t.Errorf("FormatLogTimestamp(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatLogOutputLargeInput(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "test line"
	}
	input := strings.Join(lines, "\n")

	result := FormatLogOutput(input)

	resultLines := strings.Split(result, "\n")
	if len(resultLines) != 100 {
		t.Errorf("Expected 100 lines, got %d", len(resultLines))
	}
}
