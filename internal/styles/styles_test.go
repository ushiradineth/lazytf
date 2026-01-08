package styles

import "testing"

func TestDefaultStyles(t *testing.T) {
	s := DefaultStyles()
	if s == nil {
		t.Fatalf("expected styles")
	}
	if s.Theme.Name != "default" {
		t.Fatalf("unexpected theme name: %s", s.Theme.Name)
	}
	if got := s.DiffAdd.Render("+ ok"); got == "" {
		t.Fatalf("expected rendered string")
	}
	if got := s.FilterBarActive.Render("Active"); got == "" {
		t.Fatalf("expected rendered string")
	}
}
