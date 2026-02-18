package ui

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestBuildDependencyGraphView(t *testing.T) {
	resources := []terraform.ResourceChange{
		{
			Address: "aws_instance.web",
			Action:  terraform.ActionCreate,
			Change: &terraform.Change{
				After: map[string]any{
					"depends_on": []any{"aws_security_group.web", "aws_vpc.main"},
				},
			},
		},
		{
			Address: "aws_vpc.main",
			Action:  terraform.ActionNoOp,
			Change: &terraform.Change{
				After: map[string]any{},
			},
		},
	}

	view := BuildDependencyGraphView(resources)
	if !strings.Contains(view, "aws_instance.web [create]") {
		t.Fatalf("expected instance entry, got %q", view)
	}
	if !strings.Contains(view, "-> aws_security_group.web") {
		t.Fatalf("expected dependency entry, got %q", view)
	}
	if !strings.Contains(view, "aws_vpc.main [no-op]") {
		t.Fatalf("expected vpc entry, got %q", view)
	}
	if !strings.Contains(view, "(no dependencies)") {
		t.Fatalf("expected no dependencies marker, got %q", view)
	}
}
