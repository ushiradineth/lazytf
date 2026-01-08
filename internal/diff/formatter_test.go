package diff

import "testing"

func TestFormatResourceSummary(t *testing.T) {
	f := NewFormatter()

	if got := f.FormatResourceSummary("aws_vpc.main", "+", 0); got != "+ aws_vpc.main" {
		t.Fatalf("unexpected summary: %q", got)
	}
	if got := f.FormatResourceSummary("aws_vpc.main", "+", 1); got != "+ aws_vpc.main  (1 change)" {
		t.Fatalf("unexpected summary: %q", got)
	}
	if got := f.FormatResourceSummary("aws_vpc.main", "+", 2); got != "+ aws_vpc.main  (2 changes)" {
		t.Fatalf("unexpected summary: %q", got)
	}
}

func TestFormatDiffs(t *testing.T) {
	f := NewFormatter()

	if got := f.FormatDiffs(nil); got != "  (no changes)" {
		t.Fatalf("unexpected empty format: %q", got)
	}

	diffs := []MinimalDiff{
		{Path: []string{"tags", "env"}, OldValue: "dev", NewValue: "prod", Action: DiffChange},
		{Path: []string{"name"}, OldValue: nil, NewValue: "app", Action: DiffAdd},
	}
	got := f.FormatDiffs(diffs)
	want := "  ~ tags.env: \"dev\" → \"prod\"\n  + name: \"app\""
	if got != want {
		t.Fatalf("unexpected formatted diffs:\n%q", got)
	}
}

func TestFormatDiff_UnknownAction(t *testing.T) {
	diff := MinimalDiff{
		Path:   []string{"a"},
		Action: DiffAction("weird"),
	}
	if got := FormatDiff(diff); got != "  ? a" {
		t.Fatalf("unexpected format for unknown action: %q", got)
	}
}
