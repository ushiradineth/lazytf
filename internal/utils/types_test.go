package utils

import "testing"

func TestIsMap(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		expected bool
	}{
		{"map[string]any", map[string]any{"key": "value"}, true},
		{"map[string]string", map[string]string{"key": "value"}, true},
		{"map[int]string", map[int]string{1: "value"}, true},
		{"empty map", map[string]any{}, true},
		{"nil", nil, false},
		{"string", "not a map", false},
		{"int", 42, false},
		{"slice", []string{"a", "b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMap(tt.val)
			if result != tt.expected {
				t.Errorf("IsMap(%v) = %v; want %v", tt.val, result, tt.expected)
			}
		})
	}
}

func TestIsList(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		expected bool
	}{
		{"slice of strings", []string{"a", "b"}, true},
		{"slice of any", []any{1, "two", 3.0}, true},
		{"array", [3]int{1, 2, 3}, true},
		{"empty slice", []string{}, true},
		{"nil", nil, false},
		{"string", "not a list", false},
		{"int", 42, false},
		{"map", map[string]string{"key": "value"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsList(tt.val)
			if result != tt.expected {
				t.Errorf("IsList(%v) = %v; want %v", tt.val, result, tt.expected)
			}
		})
	}
}

func TestToMap(t *testing.T) {
	tests := []struct {
		name      string
		val       any
		expectNil bool
	}{
		{"map[string]any", map[string]any{"key": "value"}, false},
		{"empty map", map[string]any{}, false},
		{"nil", nil, true},
		{"map[string]string", map[string]string{"key": "value"}, true},
		{"slice", []string{"a"}, true},
		{"string", "not a map", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToMap(tt.val)
			if tt.expectNil && result != nil {
				t.Errorf("ToMap(%v) = %v; want nil", tt.val, result)
			}
			if !tt.expectNil && result == nil {
				t.Errorf("ToMap(%v) = nil; want non-nil", tt.val)
			}
		})
	}
}

func TestInterfaceToList(t *testing.T) {
	tests := []struct {
		name        string
		val         any
		expectedLen int
		expectedNil bool
	}{
		{"slice of strings", []string{"a", "b", "c"}, 3, false},
		{"slice of any", []any{1, "two"}, 2, false},
		{"array", [2]int{1, 2}, 2, false},
		{"empty slice", []string{}, 0, false},
		{"string", "not a list", 0, true},
		{"map", map[string]string{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InterfaceToList(tt.val)
			if tt.expectedNil {
				if result != nil {
					t.Errorf("InterfaceToList(%v) = %v; want nil", tt.val, result)
				}
				return
			}
			if result == nil {
				t.Errorf("InterfaceToList(%v) = nil; want non-nil", tt.val)
				return
			}
			if len(result) != tt.expectedLen {
				t.Errorf("InterfaceToList(%v) has len %d; want %d", tt.val, len(result), tt.expectedLen)
			}
		})
	}
}

func TestDeepEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     any
		expected bool
	}{
		{"equal ints", 5, 5, true},
		{"different ints", 5, 6, false},
		{"equal strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},
		{"equal slices", []int{1, 2, 3}, []int{1, 2, 3}, true},
		{"different slices", []int{1, 2, 3}, []int{1, 2, 4}, false},
		{"equal maps", map[string]int{"a": 1}, map[string]int{"a": 1}, true},
		{"different maps", map[string]int{"a": 1}, map[string]int{"a": 2}, false},
		{"nil values", nil, nil, true},
		{"nil and non-nil", nil, "something", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeepEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("DeepEqual(%v, %v) = %v; want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
