package diff

import "testing"

func TestGetActionSymbol(t *testing.T) {
	tests := []struct {
		action   DiffAction
		expected string
	}{
		{DiffAdd, "+"},
		{DiffRemove, "-"},
		{DiffChange, "~"},
		{DiffAction("unknown"), "?"},
		{DiffAction(""), "?"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			result := tt.action.GetActionSymbol()
			if result != tt.expected {
				t.Errorf("GetActionSymbol() = %q; want %q", result, tt.expected)
			}
		})
	}
}
