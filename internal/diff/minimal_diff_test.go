package diff

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestCalculateMinimalDiff_OrderedKeys(t *testing.T) {
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

	diffs := CalculateMinimalDiff(
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

func TestCalculateMinimalDiff_SkipsKnownAfterApplyEqual(t *testing.T) {
	before := map[string]interface{}{"x": 1}
	after := map[string]interface{}{"x": 1}
	afterUnknown := map[string]interface{}{"x": true}

	diffs := CalculateMinimalDiff(before, after, afterUnknown, nil, nil, nil, "")
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs, got %d", len(diffs))
	}
}

func TestCalculateMinimalDiff_KnownAfterApplyDiff(t *testing.T) {
	before := map[string]interface{}{"x": 1}
	after := map[string]interface{}{}
	afterUnknown := map[string]interface{}{"x": true}

	diffs := CalculateMinimalDiff(before, after, afterUnknown, nil, nil, nil, "")
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

func TestCalculateMinimalDiff_StringListLCS(t *testing.T) {
	before := map[string]interface{}{
		"list": []interface{}{"a", "b", "c"},
	}
	after := map[string]interface{}{
		"list": []interface{}{"a", "c", "d"},
	}

	diffs := CalculateMinimalDiff(before, after, nil, nil, nil, nil, "")
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

func TestCalculateMinimalDiff_SingleItemStringList(t *testing.T) {
	before := map[string]interface{}{
		"values": []interface{}{"old"},
	}
	after := map[string]interface{}{
		"values": []interface{}{"new"},
	}

	diffs := CalculateMinimalDiff(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffChange || diffs[0].Path[0] != "values[0]" {
		t.Fatalf("expected change at values[0], got %s %v", diffs[0].Action, diffs[0].Path)
	}
}

func TestCalculateMinimalDiff_NestedMap(t *testing.T) {
	before := map[string]interface{}{
		"tags": map[string]interface{}{
			"Environment": "dev",
		},
	}
	after := map[string]interface{}{
		"tags": map[string]interface{}{
			"Environment": "prod",
			"Team":        "core",
		},
	}

	diffs := CalculateMinimalDiff(before, after, nil, nil, nil, nil, "")
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

func TestCalculateMinimalDiff_ListFallback(t *testing.T) {
	before := map[string]interface{}{
		"nums": []interface{}{1.0, 2.0},
	}
	after := map[string]interface{}{
		"nums": []interface{}{1.0, 2.0, 3.0},
	}

	diffs := CalculateMinimalDiff(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffChange || diffs[0].Path[0] != "nums" {
		t.Fatalf("expected change at nums, got %s %v", diffs[0].Action, diffs[0].Path)
	}
}

func TestCalculateMinimalDiff_UnknownNestedEqualSkips(t *testing.T) {
	before := map[string]interface{}{
		"obj": map[string]interface{}{"a": 1},
	}
	after := map[string]interface{}{
		"obj": map[string]interface{}{"a": 1},
	}
	afterUnknown := map[string]interface{}{
		"obj": map[string]interface{}{"a": true},
	}

	diffs := CalculateMinimalDiff(before, after, afterUnknown, nil, nil, nil, "")
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs, got %d", len(diffs))
	}
}

func TestCalculateMinimalDiff_UnknownNilAfter(t *testing.T) {
	before := map[string]interface{}{"x": 1}
	after := map[string]interface{}{"x": nil}
	afterUnknown := map[string]interface{}{"x": true}

	diffs := CalculateMinimalDiff(before, after, afterUnknown, nil, nil, nil, "")
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
	if got := formatValue(map[string]interface{}{"a": 1}); got != "{...}" {
		t.Fatalf("expected map placeholder, got %q", got)
	}
	if got := formatValue([]interface{}{"one"}); got != "\"one\"" {
		t.Fatalf("expected single string list to format to string, got %q", got)
	}
	if got := formatValue([]interface{}{"one", "two"}); got != "[...]" {
		t.Fatalf("expected list placeholder, got %q", got)
	}

	long := strings.Repeat("a", 205)
	if got := formatValue(long); !strings.HasSuffix(got, "...") {
		t.Fatalf("expected long string to be truncated, got %q", got)
	}
}

func TestOrderedKeys_RespectsOrderMaps(t *testing.T) {
	before := map[string]interface{}{"b": 1}
	after := map[string]interface{}{"a": 2, "d": 4}
	afterUnknown := map[string]interface{}{"c": true}
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
	before := map[string]interface{}{
		"values": []interface{}{"same"},
	}
	after := map[string]interface{}{
		"values": []interface{}{"same"},
	}

	diffs := CalculateMinimalDiff(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs, got %d", len(diffs))
	}
}

func TestStringListDiffs_AddRemove(t *testing.T) {
	before := map[string]interface{}{
		"list": []interface{}{"a", "b", "c"},
	}
	after := map[string]interface{}{
		"list": []interface{}{"b", "c", "d"},
	}

	diffs := CalculateMinimalDiff(before, after, nil, nil, nil, nil, "")
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
