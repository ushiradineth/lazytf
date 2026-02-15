package components

import (
	"strings"
	"testing"
)

func TestGetPadding(t *testing.T) {
	tests := []struct {
		width    int
		expected int
	}{
		{0, 0},
		{1, 1},
		{10, 10},
		{80, 80},
		{200, 200},
	}

	for _, tt := range tests {
		result := GetPadding(tt.width)
		if len(result) != tt.expected {
			t.Errorf("GetPadding(%d) = len %d, want %d", tt.width, len(result), tt.expected)
		}
		// Verify all spaces
		for _, c := range result {
			if c != ' ' {
				t.Errorf("GetPadding(%d) contains non-space character", tt.width)
			}
		}
	}
}

func TestGetRepeatedChar(t *testing.T) {
	tests := []struct {
		char     string
		width    int
		expected string
	}{
		{"─", 0, ""},
		{"─", 3, "───"},
		{"│", 5, "│││││"},
	}

	for _, tt := range tests {
		result := GetRepeatedChar(tt.char, tt.width)
		if result != tt.expected {
			t.Errorf("GetRepeatedChar(%q, %d) = %q, want %q", tt.char, tt.width, result, tt.expected)
		}
	}
}

func TestPaddingCacheConsistency(t *testing.T) {
	// Multiple calls should return the same cached value
	for i := 0; i < 100; i++ {
		a := GetPadding(50)
		b := GetPadding(50)
		if a != b {
			t.Errorf("GetPadding(50) returned inconsistent results")
		}
	}
}

// Benchmarks comparing cached vs uncached padding

func BenchmarkStringsRepeat(b *testing.B) {
	widths := []int{10, 40, 80, 120, 200}
	for _, w := range widths {
		b.Run(strings.Repeat("_", len(intToString(w))), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = strings.Repeat(" ", w)
			}
		})
	}
}

func BenchmarkGetPadding(b *testing.B) {
	widths := []int{10, 40, 80, 120, 200}
	for _, w := range widths {
		b.Run(strings.Repeat("_", len(intToString(w))), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = GetPadding(w)
			}
		})
	}
}

func BenchmarkPaddingMixedWidths(b *testing.B) {
	widths := []int{5, 10, 15, 20, 30, 40, 50, 60, 80, 100, 120}

	b.Run("strings.Repeat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, w := range widths {
				_ = strings.Repeat(" ", w)
			}
		}
	})

	b.Run("GetPadding", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, w := range widths {
				_ = GetPadding(w)
			}
		}
	})
}

// Simulate a typical render cycle with many padding operations.
func BenchmarkSimulatedRender(b *testing.B) {
	// Simulate: 8 panels × 30 lines each + 50 resources
	panelWidths := []int{40, 80, 120, 60, 100, 50, 70, 90}
	linesPerPanel := 30
	resourceIndents := []int{0, 2, 4, 2, 0, 2, 4, 6, 2, 0}

	b.Run("strings.Repeat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Panel lines
			for _, w := range panelWidths {
				for j := 0; j < linesPerPanel; j++ {
					_ = strings.Repeat(" ", w)
				}
			}
			// Resource indents
			for j := 0; j < 50; j++ {
				indent := resourceIndents[j%len(resourceIndents)]
				if indent > 0 {
					_ = strings.Repeat(" ", indent)
				}
			}
		}
	})

	b.Run("GetPadding", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Panel lines
			for _, w := range panelWidths {
				for j := 0; j < linesPerPanel; j++ {
					_ = GetPadding(w)
				}
			}
			// Resource indents
			for j := 0; j < 50; j++ {
				indent := resourceIndents[j%len(resourceIndents)]
				if indent > 0 {
					_ = GetPadding(indent)
				}
			}
		}
	})
}

// Benchmark border character caching.
func BenchmarkBorderLines(b *testing.B) {
	widths := []int{40, 80, 120}

	b.Run("strings.Repeat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, w := range widths {
				_ = strings.Repeat("─", w)
				_ = strings.Repeat("│", 30)
			}
		}
	})

	b.Run("GetRepeatedChar", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, w := range widths {
				_ = GetRepeatedChar("─", w)
				_ = GetRepeatedChar("│", 30)
			}
		}
	})
}
