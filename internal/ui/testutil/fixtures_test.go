package testutil

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestSampleResources(t *testing.T) {
	if len(SampleResources) != 5 {
		t.Errorf("expected 5 sample resources, got %d", len(SampleResources))
	}

	// Check that we have a variety of actions
	actions := make(map[terraform.ActionType]int)
	for _, r := range SampleResources {
		actions[r.Action]++
	}

	if actions[terraform.ActionCreate] < 1 {
		t.Error("expected at least one create action")
	}
	if actions[terraform.ActionUpdate] < 1 {
		t.Error("expected at least one update action")
	}
	if actions[terraform.ActionDelete] < 1 {
		t.Error("expected at least one delete action")
	}
	if actions[terraform.ActionReplace] < 1 {
		t.Error("expected at least one replace action")
	}
}

func TestFewResources(t *testing.T) {
	if len(FewResources) != 3 {
		t.Errorf("expected 3 few resources, got %d", len(FewResources))
	}

	for _, r := range FewResources {
		if r.Address == "" {
			t.Error("resource has empty address")
		}
		if r.ResourceType == "" {
			t.Error("resource has empty type")
		}
	}
}

func TestManyResources(t *testing.T) {
	if len(ManyResources) < 50 {
		t.Errorf("expected at least 50 many resources, got %d", len(ManyResources))
	}

	// Check that addresses are unique
	seen := make(map[string]bool)
	for _, r := range ManyResources {
		if seen[r.Address] {
			t.Errorf("duplicate address: %s", r.Address)
		}
		seen[r.Address] = true
	}
}

func TestModuleResources(t *testing.T) {
	if len(ModuleResources) < 3 {
		t.Errorf("expected at least 3 module resources, got %d", len(ModuleResources))
	}

	// Check that all have module. prefix
	for _, r := range ModuleResources {
		if len(r.Address) < 7 || r.Address[:7] != "module." {
			t.Errorf("expected module prefix, got %s", r.Address)
		}
	}
}

func TestSampleHistory(t *testing.T) {
	if len(SampleHistory) < 3 {
		t.Errorf("expected at least 3 history entries, got %d", len(SampleHistory))
	}

	// Check variety of statuses
	statuses := make(map[history.Status]int)
	for _, e := range SampleHistory {
		statuses[e.Status]++
	}

	if statuses[history.StatusSuccess] < 1 {
		t.Error("expected at least one success status")
	}
	if statuses[history.StatusFailed] < 1 {
		t.Error("expected at least one failed status")
	}
}

func TestResourceWithAction(t *testing.T) {
	tests := []terraform.ActionType{
		terraform.ActionCreate,
		terraform.ActionUpdate,
		terraform.ActionDelete,
		terraform.ActionReplace,
	}

	for _, action := range tests {
		t.Run(string(action), func(t *testing.T) {
			r := ResourceWithAction(action)
			if r.Action != action {
				t.Errorf("expected action %s, got %s", action, r.Action)
			}
			if r.Address == "" {
				t.Error("resource has empty address")
			}
			if r.ResourceType == "" {
				t.Error("resource has empty type")
			}
			if r.Change == nil {
				t.Error("resource has nil change")
			}
		})
	}
}

func TestModuleResource(t *testing.T) {
	r := ModuleResource("module.vpc", "aws_vpc", "main", terraform.ActionCreate)

	expectedAddress := "module.vpc.aws_vpc.main"
	if r.Address != expectedAddress {
		t.Errorf("expected address %s, got %s", expectedAddress, r.Address)
	}
	if r.ResourceType != "aws_vpc" {
		t.Errorf("expected type aws_vpc, got %s", r.ResourceType)
	}
	if r.ResourceName != "main" {
		t.Errorf("expected name main, got %s", r.ResourceName)
	}
	if r.Action != terraform.ActionCreate {
		t.Errorf("expected action create, got %s", r.Action)
	}
}

func TestResourceWithChange(t *testing.T) {
	before := map[string]any{"key": "old"}
	after := map[string]any{"key": "new"}

	r := ResourceWithChange("test.resource", terraform.ActionUpdate, before, after)

	if r.Address != "test.resource" {
		t.Errorf("expected address test.resource, got %s", r.Address)
	}
	if r.Action != terraform.ActionUpdate {
		t.Errorf("expected action update, got %s", r.Action)
	}
	if r.Change == nil {
		t.Fatal("expected non-nil change")
	}
	if r.Change.Before["key"] != "old" {
		t.Errorf("expected before key 'old', got %v", r.Change.Before["key"])
	}
	if r.Change.After["key"] != "new" {
		t.Errorf("expected after key 'new', got %v", r.Change.After["key"])
	}
}

func TestHistoryEntry(t *testing.T) {
	e := HistoryEntry(5, history.StatusSuccess, "Test summary")

	if e.ID != 5 {
		t.Errorf("expected ID 5, got %d", e.ID)
	}
	if e.Status != history.StatusSuccess {
		t.Errorf("expected status success, got %s", e.Status)
	}
	if e.Summary != "Test summary" {
		t.Errorf("expected summary 'Test summary', got %s", e.Summary)
	}
	if e.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}
	if e.Duration == 0 {
		t.Error("expected non-zero Duration")
	}
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-5, "-5"},
	}

	for _, tt := range tests {
		got := intToString(tt.input)
		if got != tt.expected {
			t.Errorf("intToString(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
