package ui

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestBuildDriftAnalysisView(t *testing.T) {
	resources := []terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionUpdate, ActionReason: "updated outside terraform"},
		{Address: "aws_s3_bucket.logs", Action: terraform.ActionNoOp},
		{Address: "aws_db_instance.main", Action: terraform.ActionReplace},
	}

	view := BuildDriftAnalysisView(resources)
	if !strings.Contains(view, "Drift candidates: 2") {
		t.Fatalf("expected drift count, got %q", view)
	}
	if !strings.Contains(view, "aws_instance.web [update]") {
		t.Fatalf("expected update candidate, got %q", view)
	}
	if !strings.Contains(view, "aws_db_instance.main [replace]") {
		t.Fatalf("expected replace candidate, got %q", view)
	}
}
