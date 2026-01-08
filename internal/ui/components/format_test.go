package components

import "testing"

func TestFormatPathForDisplay(t *testing.T) {
	tests := []struct {
		path []string
		want string
	}{
		{[]string{}, ""},
		{[]string{"data", "app.yaml"}, `data."app.yaml"`},
		{[]string{"tags", "Cost Center"}, `tags."Cost Center"`},
		{[]string{"data", "config", "app.yaml"}, `data.config."app.yaml"`},
		{[]string{`path`, `with"quote`}, `path."with\"quote"`},
		{[]string{`path`, `with\\slash`}, `path."with\\\\slash"`},
		{[]string{`spec[0]`, `foo.bar`}, `spec[0]."foo.bar"`},
		{[]string{"values[0]"}, "values[0]"},
	}

	for _, tt := range tests {
		if got := formatPathForDisplay(tt.path); got != tt.want {
			t.Fatalf("path %v: expected %q, got %q", tt.path, tt.want, got)
		}
	}
}
