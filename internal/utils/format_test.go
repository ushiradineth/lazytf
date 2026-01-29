package utils

import (
	"strings"
	"testing"
)

func TestFormatValue(t *testing.T) {
	unknownChecker := func(val any) bool {
		if m, ok := val.(map[string]any); ok {
			_, hasUnknown := m["__unknown__"]
			return hasUnknown
		}
		return false
	}

	tests := []struct {
		name     string
		val      any
		checker  func(any) bool
		expected string
	}{
		{"nil value", nil, nil, "(null)"},
		{"short string", "hello", nil, `"hello"`},
		{"long string truncates", strings.Repeat("a", 250), nil, `"` + strings.Repeat("a", 197) + `"...`},
		{"integer", 42, nil, "42"},
		{"float", 3.14, nil, "3.14"},
		{"boolean true", true, nil, "true"},
		{"boolean false", false, nil, "false"},
		{"map", map[string]any{"key": "value"}, nil, "{...}"},
		{"list multi items", []any{1, 2, 3}, nil, "[...]"},
		{"list single string", []any{"single"}, nil, `"single"`},
		{"list single int", []any{42}, nil, "[...]"},
		{"unknown value", map[string]any{"__unknown__": true}, unknownChecker, "(known after apply)"},
		{"nil checker unknown map", map[string]any{"__unknown__": true}, nil, "{...}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValue(tt.val, tt.checker)
			if result != tt.expected {
				t.Errorf("FormatValue(%v) = %q; want %q", tt.val, result, tt.expected)
			}
		})
	}
}

func TestFormatPath(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		expected string
	}{
		{"empty path", []string{}, ""},
		{"single segment", []string{"root"}, "root"},
		{"two segments", []string{"root", "child"}, "root.child"},
		{"multiple segments", []string{"a", "b", "c", "d"}, "a.b.c.d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPath(tt.path)
			if result != tt.expected {
				t.Errorf("FormatPath(%v) = %q; want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestTruncateMiddle(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate even", "hello world", 8, "he...ld"},
		{"truncate odd", "hello world", 9, "hel...rld"},
		{"maxLen 3", "hello", 3, "hel"},
		{"maxLen 2", "hello", 2, "he"},
		{"maxLen 1", "hello", 1, "h"},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateMiddle(tt.s, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateMiddle(%q, %d) = %q; want %q", tt.s, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestTruncateEnd(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate", "hello world", 8, "hello..."},
		{"maxLen 4", "hello", 4, "h..."},
		{"maxLen 3", "hello", 3, "hel"},
		{"maxLen 2", "hello", 2, "he"},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateEnd(tt.s, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateEnd(%q, %d) = %q; want %q", tt.s, tt.maxLen, result, tt.expected)
			}
		})
	}
}
