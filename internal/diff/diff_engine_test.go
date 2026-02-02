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

func TestJoinJSONPointer(t *testing.T) {
	if got := joinJSONPointer("", "a/b~c"); got != "/a~1b~0c" {
		t.Fatalf("unexpected pointer: %q", got)
	}
	if got := joinJSONPointer("/root", "k"); got != "/root/k" {
		t.Fatalf("unexpected pointer: %q", got)
	}
}

func TestIsUnknown(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string", "hello", false},
		{"int", 42, false},
		{"float", 3.14, false},
		{"nil", nil, false},
		{"empty map", map[string]any{}, false},
		{"map with true bool", map[string]any{"a": true}, true},
		{"map with false bool", map[string]any{"a": false}, false},
		{"map with string", map[string]any{"a": "value"}, false},
		{"nested map with true", map[string]any{"outer": map[string]any{"inner": true}}, true},
		{"nested map all false", map[string]any{"outer": map[string]any{"inner": false}}, false},
		{"empty slice", []any{}, false},
		{"slice with true", []any{true}, true},
		{"slice with false", []any{false}, false},
		{"slice with string", []any{"hello"}, false},
		{"slice with nested map containing true", []any{map[string]any{"a": true}}, true},
		{"mixed slice with true", []any{"a", 1, true}, true},
		{"mixed slice no bool", []any{"a", 1, 3.14}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnknown(tt.val)
			if got != tt.want {
				t.Errorf("isUnknown(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestIsUnknownValue(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"UnknownValue type", UnknownValue{}, true},
		{"string", "hello", false},
		{"int", 42, false},
		{"nil", nil, false},
		{"map", map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnknownValue(tt.val)
			if got != tt.want {
				t.Errorf("isUnknownValue(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestIsMap(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"map[string]any", map[string]any{"a": 1}, true},
		{"empty map", map[string]any{}, true},
		{"nil", nil, false},
		{"string", "hello", false},
		{"slice", []any{1, 2}, false},
		{"int", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMap(tt.val)
			if got != tt.want {
				t.Errorf("isMap(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestIsList(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"slice any", []any{1, 2}, true},
		{"slice string", []string{"a", "b"}, true},
		{"empty slice", []any{}, true},
		{"array", [3]int{1, 2, 3}, true},
		{"nil", nil, false},
		{"string", "hello", false},
		{"map", map[string]any{}, false},
		{"int", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isList(tt.val)
			if got != tt.want {
				t.Errorf("isList(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestToMap(t *testing.T) {
	tests := []struct {
		name    string
		val     any
		wantNil bool
	}{
		{"valid map", map[string]any{"a": 1}, false},
		{"empty map", map[string]any{}, false},
		{"nil", nil, true},
		{"string", "hello", true},
		{"slice", []any{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toMap(tt.val)
			if tt.wantNil && got != nil {
				t.Errorf("toMap(%v) = %v, want nil", tt.val, got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("toMap(%v) = nil, want non-nil", tt.val)
			}
		})
	}
}

func TestInterfaceToList(t *testing.T) {
	tests := []struct {
		name    string
		val     any
		wantLen int
		wantNil bool
	}{
		{"slice any", []any{1, 2, 3}, 3, false},
		{"empty slice", []any{}, 0, false},
		{"array", [2]string{"a", "b"}, 2, false},
		{"not a slice", "hello", 0, true},
		{"map", map[string]any{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interfaceToList(tt.val)
			if tt.wantNil && got != nil {
				t.Errorf("interfaceToList(%v) = %v, want nil", tt.val, got)
			}
			if !tt.wantNil && len(got) != tt.wantLen {
				t.Errorf("interfaceToList(%v) len = %d, want %d", tt.val, len(got), tt.wantLen)
			}
		})
	}
}

func TestInterfaceToStrings(t *testing.T) {
	tests := []struct {
		name string
		list []any
		want []string
	}{
		{"all strings", []any{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"empty", []any{}, []string{}},
		{"mixed types skips non-strings", []any{"a", 1, "b"}, []string{"a", "b"}},
		{"no strings", []any{1, 2, 3}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interfaceToStrings(tt.list)
			if len(got) != len(tt.want) {
				t.Errorf("interfaceToStrings(%v) = %v, want %v", tt.list, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("interfaceToStrings(%v)[%d] = %q, want %q", tt.list, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestAllStrings(t *testing.T) {
	tests := []struct {
		name string
		list []any
		want bool
	}{
		{"all strings", []any{"a", "b", "c"}, true},
		{"empty", []any{}, true},
		{"mixed types", []any{"a", 1, "b"}, false},
		{"no strings", []any{1, 2, 3}, false},
		{"single string", []any{"a"}, true},
		{"single non-string", []any{1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allStrings(tt.list)
			if got != tt.want {
				t.Errorf("allStrings(%v) = %v, want %v", tt.list, got, tt.want)
			}
		})
	}
}

func TestShouldSkipUnknown(t *testing.T) {
	tests := []struct {
		name          string
		unknownExists bool
		unknownVal    any
		beforeVal     any
		afterVal      any
		want          bool
	}{
		{"unknown exists and values equal", true, true, "a", "a", true},
		{"unknown exists but values different", true, true, "a", "b", false},
		{"unknown doesn't exist", false, nil, "a", "a", false},
		{"unknown false", true, false, "a", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipUnknown(tt.unknownExists, tt.unknownVal, tt.beforeVal, tt.afterVal)
			if got != tt.want {
				t.Errorf("shouldSkipUnknown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeUnknownValue(t *testing.T) {
	tests := []struct {
		name          string
		unknownExists bool
		unknownVal    any
		afterVal      any
		afterExists   bool
		wantUnknown   bool
	}{
		{"unknown exists and after nil", true, true, nil, false, true},
		{"unknown exists and after empty", true, true, nil, true, true},
		{"unknown exists but after has value", true, true, "value", true, false},
		{"unknown doesn't exist", false, nil, "value", true, false},
		{"unknown false", true, false, nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, _ := normalizeUnknownValue(tt.unknownExists, tt.unknownVal, tt.afterVal, tt.afterExists)
			_, isUnknownVal := val.(UnknownValue)
			if isUnknownVal != tt.wantUnknown {
				t.Errorf("normalizeUnknownValue() isUnknown = %v, want %v", isUnknownVal, tt.wantUnknown)
			}
		})
	}
}

func TestStringListDiffsAllRemoved(t *testing.T) {
	before := map[string]any{
		"list": []any{"a", "b", "c"},
	}
	after := map[string]any{
		"list": []any{},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	// When all items removed, the function returns a single change diff for the whole list
	if len(diffs) == 0 {
		t.Fatal("expected at least 1 diff")
	}
}

func TestStringListDiffsEmptyToValues(t *testing.T) {
	before := map[string]any{
		"list": []any{},
	}
	after := map[string]any{
		"list": []any{"x", "y", "z"},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	// When going from empty to values, might get single change diff
	if len(diffs) < 1 {
		t.Fatalf("expected at least 1 diff, got %d", len(diffs))
	}
}

func TestStringListDiffsWithSameValues(t *testing.T) {
	before := map[string]any{
		"list": []any{"a", "b", "c"},
	}
	after := map[string]any{
		"list": []any{"a", "b", "c"},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	// Same values, no diffs expected
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs, got %d", len(diffs))
	}
}

func TestCalculateDiffs_DeletedKey(t *testing.T) {
	before := map[string]any{
		"a": 1,
		"b": 2,
	}
	after := map[string]any{
		"a": 1,
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffRemove || diffs[0].Path[0] != "b" {
		t.Errorf("expected remove of 'b', got %s %v", diffs[0].Action, diffs[0].Path)
	}
}

func TestCalculateDiffs_AddedKey(t *testing.T) {
	before := map[string]any{
		"a": 1,
	}
	after := map[string]any{
		"a": 1,
		"b": 2,
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffAdd || diffs[0].Path[0] != "b" {
		t.Errorf("expected add of 'b', got %s %v", diffs[0].Action, diffs[0].Path)
	}
}

func TestCalculateDiffs_NestedDeletion(t *testing.T) {
	before := map[string]any{
		"config": map[string]any{
			"a": 1,
			"b": 2,
		},
	}
	after := map[string]any{
		"config": map[string]any{
			"a": 1,
		},
	}

	diffs := CalculateDiffs(before, after, nil, nil, nil, nil, "")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Action != DiffRemove {
		t.Errorf("expected remove action, got %s", diffs[0].Action)
	}
}
