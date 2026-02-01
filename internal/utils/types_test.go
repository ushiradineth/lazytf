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
