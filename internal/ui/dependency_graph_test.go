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

func TestBuildDependencyGraphViewFromStateJSON(t *testing.T) {
	stateJSON := `{
	  "resources": [
	    {
	      "mode": "managed",
	      "type": "null_resource",
	      "name": "example",
	      "instances": [
	        {
	          "dependencies": ["null_resource.dependency"]
	        }
	      ]
	    },
	    {
	      "mode": "managed",
	      "module": "module.app",
	      "type": "aws_instance",
	      "name": "web",
	      "instances": [
	        {
	          "index_key": 0,
	          "dependencies": []
	        }
	      ]
	    }
	  ]
	}`

	view, err := BuildDependencyGraphViewFromStateJSON(stateJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(view, "null_resource.example") {
		t.Fatalf("expected null_resource.example entry, got %q", view)
	}
	if !strings.Contains(view, "-> null_resource.dependency") {
		t.Fatalf("expected dependency entry, got %q", view)
	}
	if !strings.Contains(view, "module.app.aws_instance.web[0]") {
		t.Fatalf("expected module instance entry, got %q", view)
	}
}

func TestBuildDependencyGraphViewFromStateJSONInvalid(t *testing.T) {
	_, err := BuildDependencyGraphViewFromStateJSON("not-json")
	if err == nil {
		t.Fatal("expected parse error")
	}
}
