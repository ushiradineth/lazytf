package utils

import "testing"

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
