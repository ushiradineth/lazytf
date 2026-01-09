package diff

import (
	"testing"

	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestEngine_GetResourceDiffsNilChange(t *testing.T) {
	e := NewEngine()
	rc := &terraform.ResourceChange{}
	if got := e.GetResourceDiffs(rc); got != nil {
		t.Fatalf("expected nil diffs, got %v", got)
	}
	if got := e.CountChanges(rc); got != 0 {
		t.Fatalf("expected 0 changes, got %d", got)
	}
}

func TestEngine_GetResourceDiffsNilMaps(t *testing.T) {
	e := NewEngine()
	rc := &terraform.ResourceChange{
		Change: &terraform.Change{
			Before: nil,
			After: map[string]any{
				"name": "app",
			},
			AfterUnknown: nil,
		},
	}

	diffs := e.GetResourceDiffs(rc)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffAdd || diffs[0].Path[0] != "name" {
		t.Fatalf("unexpected diff: %s %v", diffs[0].Action, diffs[0].Path)
	}
}
