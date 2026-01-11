package diff

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestCalculateDiffs_OrderedKeys(t *testing.T) {
	raw := []byte(`{
		"actions": ["update"],
		"before": {"b": 1, "a": 1},
		"after": {"b": 1, "a": 2, "c": 3},
		"after_unknown": {}
	}`)

	var change terraform.Change
	if err := json.Unmarshal(raw, &change); err != nil {
		t.Fatalf("unmarshal change: %v", err)
	}

	diffs := CalculateDiffs(
		change.Before,
		change.After,
		change.AfterUnknown,
		change.BeforeOrder,
		change.AfterOrder,
		change.AfterUnknownOrder,
		"",
	)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}
	if got := diffs[0].Path[0]; got != "a" {
		t.Fatalf("expected first diff to be 'a', got %q", got)
	}
	if got := diffs[1].Path[0]; got != "c" {
		t.Fatalf("expected second diff to be 'c', got %q", got)
	}
}

func TestCalculateDiffs_SkipsKnownAfterApplyEqual(t *testing.T) {
	before := map[string]any{"x": 1}
	after := map[string]any{"x": 1}
	afterUnknown := map[string]any{"x": true}

	diffs := CalculateDiffs(before, after, afterUnknown, nil, nil, nil, "")
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs, got %d", len(diffs))
	}
}

func TestCalculateDiffs_KnownAfterApplyDiff(t *testing.T) {
	before := map[string]any{"x": 1}
	after := map[string]any{}
	afterUnknown := map[string]any{"x": true}

	diffs := CalculateDiffs(before, after, afterUnknown, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffChange {
		t.Fatalf("expected change action, got %s", diffs[0].Action)
	}
	if _, ok := diffs[0].NewValue.(UnknownValue); !ok {
		t.Fatalf("expected new value to be UnknownValue, got %T", diffs[0].NewValue)
	}

	formatted := FormatDiff(diffs[0])
	if formatted == "" || !strings.Contains(formatted, "(known after apply)") {
		t.Fatalf("unexpected formatted diff: %q", formatted)
	}
}

func TestCalculateDiffs_StringListLCS(t *testing.T) {
	before := map[string]any{
		"list": []any{"a", "b", "c"},
	}
	after := map[string]any{
		"list": []any{"a", "c", "d"},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}
	if diffs[0].Action != DiffRemove || diffs[0].Path[0] != "list[1]" {
		t.Fatalf("expected remove at list[1], got %s %v", diffs[0].Action, diffs[0].Path)
	}
	if diffs[1].Action != DiffAdd || diffs[1].Path[0] != "list[2]" {
		t.Fatalf("expected add at list[2], got %s %v", diffs[1].Action, diffs[1].Path)
	}
}

func TestCalculateDiffs_SingleItemStringList(t *testing.T) {
	before := map[string]any{
		"values": []any{"old"},
	}
	after := map[string]any{
		"values": []any{"new"},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffChange || diffs[0].Path[0] != "values[0]" {
		t.Fatalf("expected change at values[0], got %s %v", diffs[0].Action, diffs[0].Path)
	}
}

func TestCalculateDiffs_NestedMap(t *testing.T) {
	before := map[string]any{
		"tags": map[string]any{
			"Environment": "dev",
		},
	}
	after := map[string]any{
		"tags": map[string]any{
			"Environment": "prod",
			"Team":        "core",
		},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}
	if got := strings.Join(diffs[0].Path, "."); got != "tags.Environment" {
		t.Fatalf("expected tags.Environment diff first, got %q", got)
	}
	if diffs[0].Action != DiffChange {
		t.Fatalf("expected change action, got %s", diffs[0].Action)
	}
	if got := strings.Join(diffs[1].Path, "."); got != "tags.Team" {
		t.Fatalf("expected tags.Team diff second, got %q", got)
	}
	if diffs[1].Action != DiffAdd {
		t.Fatalf("expected add action, got %s", diffs[1].Action)
	}
}

func TestCalculateDiffs_ListFallback(t *testing.T) {
	before := map[string]any{
		"nums": []any{1.0, 2.0},
	}
	after := map[string]any{
		"nums": []any{1.0, 2.0, 3.0},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffChange || diffs[0].Path[0] != "nums" {
		t.Fatalf("expected change at nums, got %s %v", diffs[0].Action, diffs[0].Path)
	}
}

func TestCalculateDiffs_AfterUnknownNestedMapsAndLists(t *testing.T) {
	before := map[string]any{
		"config": map[string]any{
			"items": []any{"a", "b"},
			"meta":  map[string]any{"enabled": true},
		},
	}
	after := map[string]any{
		"config": map[string]any{
			"items": []any{"a", "b"},
			"meta":  map[string]any{"enabled": true},
		},
	}
	afterUnknown := map[string]any{
		"config": map[string]any{
			"items": true,
			"meta":  map[string]any{"enabled": true},
		},
	}

	diffs := CalculateDiffs(before, after, afterUnknown, nil, nil, nil, "")
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs when unknown matches existing values")
	}
}

func TestCalculateDiffs_ListMixedTypes(t *testing.T) {
	before := map[string]any{
		"mixed": []any{"a", 1},
	}
	after := map[string]any{
		"mixed": []any{"a", 2},
	}
	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected single diff for mixed list types, got %d", len(diffs))
	}
	if diffs[0].Path[0] != "mixed" {
		t.Fatalf("unexpected diff path: %#v", diffs[0].Path)
	}
}

func TestCalculateDiffs_UnknownNestedEqualSkips(t *testing.T) {
	before := map[string]any{
		"obj": map[string]any{"a": 1},
	}
	after := map[string]any{
		"obj": map[string]any{"a": 1},
	}
	afterUnknown := map[string]any{
		"obj": map[string]any{"a": true},
	}

	diffs := CalculateDiffs(before, after, afterUnknown, nil, nil, nil, "")
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs, got %d", len(diffs))
	}
}

func TestCalculateDiffs_UnknownNilAfter(t *testing.T) {
	before := map[string]any{"x": 1}
	after := map[string]any{"x": nil}
	afterUnknown := map[string]any{"x": true}

	diffs := CalculateDiffs(before, after, afterUnknown, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if _, ok := diffs[0].NewValue.(UnknownValue); !ok {
		t.Fatalf("expected UnknownValue, got %T", diffs[0].NewValue)
	}
}

func TestFormatValue_PrimitivesAndContainers(t *testing.T) {
	if got := formatValue(nil); got != "(null)" {
		t.Fatalf("expected null, got %q", got)
	}
	if got := formatValue("hi"); got != "\"hi\"" {
		t.Fatalf("expected quoted string, got %q", got)
	}
	if got := formatValue(UnknownValue{}); got != "(known after apply)" {
		t.Fatalf("expected known after apply, got %q", got)
	}
	if got := formatValue(map[string]any{"a": 1}); got != "{...}" {
		t.Fatalf("expected map placeholder, got %q", got)
	}
	if got := formatValue([]any{"one"}); got != "\"one\"" {
		t.Fatalf("expected single string list to format to string, got %q", got)
	}
	if got := formatValue([]any{"one", "two"}); got != "[...]" {
		t.Fatalf("expected list placeholder, got %q", got)
	}

	long := strings.Repeat("a", 205)
	if got := formatValue(long); !strings.HasSuffix(got, "...") {
		t.Fatalf("expected long string to be truncated, got %q", got)
	}
}

func TestOrderedKeys_RespectsOrderMaps(t *testing.T) {
	before := map[string]any{"b": 1}
	after := map[string]any{"a": 2, "d": 4}
	afterUnknown := map[string]any{"c": true}
	beforeOrder := map[string][]string{"": {"b", "a"}}
	afterOrder := map[string][]string{"": {"a", "c"}}
	afterUnknownOrder := map[string][]string{"": {"c", "b"}}

	keys := orderedKeys(before, after, afterUnknown, beforeOrder, afterOrder, afterUnknownOrder, "")
	want := []string{"b", "a", "c", "d"}
	if !deepEqual(keys, want) {
		t.Fatalf("expected keys %v, got %v", want, keys)
	}
}

func TestCalculateListDiff_SingleStringNoChange(t *testing.T) {
	before := map[string]any{
		"values": []any{"same"},
	}
	after := map[string]any{
		"values": []any{"same"},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs, got %d", len(diffs))
	}
}

func TestStringListDiffs_AddRemove(t *testing.T) {
	before := map[string]any{
		"list": []any{"a", "b", "c"},
	}
	after := map[string]any{
		"list": []any{"b", "c", "d"},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}
	if diffs[0].Action != DiffRemove || diffs[0].Path[0] != "list[0]" {
		t.Fatalf("expected remove at list[0], got %s %v", diffs[0].Action, diffs[0].Path)
	}
	if diffs[1].Action != DiffAdd || diffs[1].Path[0] != "list[2]" {
		t.Fatalf("expected add at list[2], got %s %v", diffs[1].Action, diffs[1].Path)
	}
}

func TestFormatPath(t *testing.T) {
	if got := formatPath([]string{"a", "b", "c"}); got != "a.b.c" {
		t.Fatalf("unexpected path: %q", got)
	}
	if got := formatPath([]string{"root", "list[0]"}); got != "root.list[0]" {
		t.Fatalf("unexpected path: %q", got)
	}
}

func TestJoinJSONPointer(t *testing.T) {
	if got := joinJSONPointer("", "a/b~c"); got != "/a~1b~0c" {
		t.Fatalf("unexpected pointer: %q", got)
	}
	if got := joinJSONPointer("/root", "k"); got != "/root/k" {
		t.Fatalf("unexpected pointer: %q", got)
	}
}
