package terraform

import (
	"encoding/json"
	"testing"
)

func TestModuleAddressUnmarshal_String(t *testing.T) {
	var addr ModuleAddress
	if err := json.Unmarshal([]byte(`"module.foo"`), &addr); err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}
	if len(addr) != 1 || addr[0] != "module.foo" {
		t.Fatalf("unexpected address: %#v", addr)
	}
}

func TestModuleAddressUnmarshal_List(t *testing.T) {
	var addr ModuleAddress
	if err := json.Unmarshal([]byte(`["module.foo","module.bar"]`), &addr); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(addr) != 2 || addr[0] != "module.foo" || addr[1] != "module.bar" {
		t.Fatalf("unexpected address: %#v", addr)
	}
}

func TestModuleAddressUnmarshal_Null(t *testing.T) {
	var addr ModuleAddress
	if err := json.Unmarshal([]byte(`null`), &addr); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if addr != nil {
		t.Fatalf("expected nil, got %#v", addr)
	}
}

func TestModuleAddressUnmarshal_EmptyString(t *testing.T) {
	var addr ModuleAddress
	if err := json.Unmarshal([]byte(`""`), &addr); err != nil {
		t.Fatalf("unmarshal empty string: %v", err)
	}
	if addr != nil {
		t.Fatalf("expected nil, got %#v", addr)
	}
}

func TestModuleAddressUnmarshal_InvalidType(t *testing.T) {
	var addr ModuleAddress
	if err := json.Unmarshal([]byte(`123`), &addr); err == nil {
		t.Fatalf("expected error for invalid type")
	}
}

func TestGetActionType(t *testing.T) {
	tests := []struct {
		actions []string
		want    ActionType
	}{
		{[]string{"create"}, ActionCreate},
		{[]string{"update"}, ActionUpdate},
		{[]string{"delete"}, ActionDelete},
		{[]string{"no-op"}, ActionNoOp},
		{[]string{"read"}, ActionRead},
		{[]string{"delete", "create"}, ActionReplace},
		{[]string{"update", "read"}, ActionUpdate},
		{nil, ActionNoOp},
	}

	for _, tt := range tests {
		if got := GetActionType(tt.actions); got != tt.want {
			t.Fatalf("actions %v: expected %s, got %s", tt.actions, tt.want, got)
		}
	}
}

func TestChangeUnmarshal_Order(t *testing.T) {
	raw := []byte(`{
		"actions": ["update"],
		"after": {"outer": {"b": 1, "a": 2}, "z": 1}
	}`)

	var change Change
	if err := json.Unmarshal(raw, &change); err != nil {
		t.Fatalf("unmarshal change: %v", err)
	}

	rootOrder := change.AfterOrder[""]
	if len(rootOrder) != 2 || rootOrder[0] != "outer" || rootOrder[1] != "z" {
		t.Fatalf("unexpected root order: %#v", rootOrder)
	}
	childOrder := change.AfterOrder["/outer"]
	if len(childOrder) != 2 || childOrder[0] != "b" || childOrder[1] != "a" {
		t.Fatalf("unexpected child order: %#v", childOrder)
	}
}

func TestChangeUnmarshal_Nulls(t *testing.T) {
	raw := []byte(`{
		"actions": ["no-op"],
		"before": null,
		"after": null,
		"after_unknown": null,
		"before_sensitive": {"a": true},
		"after_sensitive": {"b": true}
	}`)

	var change Change
	if err := json.Unmarshal(raw, &change); err != nil {
		t.Fatalf("unmarshal change: %v", err)
	}
	if change.Before != nil || change.After != nil || change.AfterUnknown != nil {
		t.Fatalf("expected nil maps, got before=%v after=%v unknown=%v", change.Before, change.After, change.AfterUnknown)
	}
	if change.BeforeSensitive["a"] != true || change.AfterSensitive["b"] != true {
		t.Fatalf("expected sensitive maps to be set")
	}
}

func TestChangeUnmarshal_ReplacePaths(t *testing.T) {
	raw := []byte(`{
		"actions": ["delete","create"],
		"replace_paths": [["allocated_storage"], ["network", "self_link"]]
	}`)

	var change Change
	if err := json.Unmarshal(raw, &change); err != nil {
		t.Fatalf("unmarshal change: %v", err)
	}
	if len(change.ReplacePaths) != 2 || change.ReplacePaths[0][0] != "allocated_storage" {
		t.Fatalf("unexpected replace paths: %#v", change.ReplacePaths)
	}
}

func TestBuildOrderMap_InvalidJSON(t *testing.T) {
	if got := buildOrderMap([]byte(`{"a":`)); got != nil {
		t.Fatalf("expected nil for invalid json, got %#v", got)
	}
}

func TestBuildOrderMap_PrimitivesAndNested(t *testing.T) {
	raw := []byte(`{"a":1,"b":{"c":2}}`)
	order := buildOrderMap(raw)
	root := order[""]
	if len(root) != 2 || root[0] != "a" || root[1] != "b" {
		t.Fatalf("unexpected root order: %#v", root)
	}
	child := order["/b"]
	if len(child) != 1 || child[0] != "c" {
		t.Fatalf("unexpected child order: %#v", child)
	}
}

func TestBuildOrderMap_SkipsArrays(t *testing.T) {
	raw := []byte(`{"list":[{"a":1},{"b":2}],"z":3}`)
	order := buildOrderMap(raw)
	root := order[""]
	if len(root) != 2 || root[0] != "list" || root[1] != "z" {
		t.Fatalf("unexpected root order: %#v", root)
	}
}

func TestBuildOrderMap_NonObject(t *testing.T) {
	if order := buildOrderMap([]byte(`[]`)); order != nil {
		t.Fatalf("expected nil order for array, got %#v", order)
	}
}

func TestEscapeJSONPointer(t *testing.T) {
	if got := escapeJSONPointer("a/b~c"); got != "a~1b~0c" {
		t.Fatalf("unexpected escape: %q", got)
	}
}

func TestActionTypeDisplayHelpers(t *testing.T) {
	if got := ActionCreate.GetActionIcon(); got != "[+]" {
		t.Fatalf("unexpected icon: %q", got)
	}
	if got := ActionUpdate.GetActionVerb(); got != "will be updated" {
		t.Fatalf("unexpected verb: %q", got)
	}
	if got := ActionType("bogus").GetActionIcon(); got != "[?]" {
		t.Fatalf("unexpected icon for unknown: %q", got)
	}
	if got := ActionType("bogus").GetActionVerb(); got != "unknown action" {
		t.Fatalf("unexpected verb for unknown: %q", got)
	}
}
