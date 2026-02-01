package utils

import "testing"

func TestMinInt(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a less than b", 1, 5, 1},
		{"b less than a", 5, 1, 1},
		{"equal values", 3, 3, 3},
		{"negative values", -5, -1, -5},
		{"zero and positive", 0, 5, 0},
		{"zero and negative", 0, -5, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MinInt(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("MinInt(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a greater than b", 5, 1, 5},
		{"b greater than a", 1, 5, 5},
		{"equal values", 3, 3, 3},
		{"negative values", -5, -1, -1},
		{"zero and positive", 0, 5, 5},
		{"zero and negative", 0, -5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaxInt(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("MaxInt(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
