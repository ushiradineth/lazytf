package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestSummarizePlan(t *testing.T) {
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{
		{Action: terraform.ActionCreate},
		{Action: terraform.ActionUpdate},
		{Action: terraform.ActionDelete},
		{Action: terraform.ActionReplace},
		{Action: terraform.ActionNoOp},
	}}

	s := summarizePlan(plan)
	if s.Create != 1 || s.Update != 1 || s.Delete != 1 || s.Replace != 1 {
		t.Fatalf("unexpected summary: %+v", s)
	}
}

func TestBuildPlanCompareView(t *testing.T) {
	current := planSummary{Create: 2, Update: 1, Delete: 0, Replace: 1}
	prev := &history.OperationEntry{FinishedAt: time.Now(), Summary: "add 1, change 2"}

	view := buildPlanCompareView(current, prev)
	if !strings.Contains(view, "Current plan") {
		t.Fatalf("expected current plan section, got %q", view)
	}
	if !strings.Contains(view, "Previous plan operation") {
		t.Fatalf("expected previous plan section, got %q", view)
	}
}
