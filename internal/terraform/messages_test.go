package terraform

import "testing"

func TestParseActionType(t *testing.T) {
	tests := []struct {
		input string
		want  ActionType
	}{
		{"create", ActionCreate},
		{"delete", ActionDelete},
		{"destroy", ActionDelete},
		{"remove", ActionDelete},
		{"change", ActionUpdate},
		{"replace", ActionReplace},
		{"read", ActionRead},
		{"no-op", ActionNoOp},
		{"noop", ActionNoOp},
		{"unknown", ActionNoOp},
	}

	for _, tt := range tests {
		if got := ParseActionType(tt.input); got != tt.want {
			t.Fatalf("input %q: expected %s, got %s", tt.input, tt.want, got)
		}
	}
}
