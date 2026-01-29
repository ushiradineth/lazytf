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

func TestClamp(t *testing.T) {
	tests := []struct {
		name            string
		val, minV, maxV int
		expected        int
	}{
		{"within range", 5, 0, 10, 5},
		{"below min", -5, 0, 10, 0},
		{"above max", 15, 0, 10, 10},
		{"at min", 0, 0, 10, 0},
		{"at max", 10, 0, 10, 10},
		{"negative range", -5, -10, -1, -5},
		{"negative below", -15, -10, -1, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Clamp(tt.val, tt.minV, tt.maxV)
			if result != tt.expected {
				t.Errorf("Clamp(%d, %d, %d) = %d; want %d", tt.val, tt.minV, tt.maxV, result, tt.expected)
			}
		})
	}
}
