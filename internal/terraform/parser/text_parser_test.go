package parser

import (
	"strings"
	"testing"

	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestTextParserParsesResources(t *testing.T) {
	input := `Terraform will perform the following actions:

  # aws_instance.web will be created
  + resource "aws_instance" "web" {
      + ami = "ami-123"
      + id  = (known after apply)
    }

  # aws_instance.db will be updated in-place
  ~ resource "aws_instance" "db" {
      ~ instance_type = "t2.micro" -> "t2.small"
    }

Plan: 1 to add, 1 to change, 0 to destroy.
`

	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(plan.Resources))
	}

	if plan.Resources[0].Action != terraform.ActionCreate {
		t.Fatalf("expected create action, got %s", plan.Resources[0].Action)
	}
	if plan.Resources[0].ResourceType != "aws_instance" || plan.Resources[0].ResourceName != "web" {
		t.Fatalf("unexpected resource metadata: %#v", plan.Resources[0])
	}

	if plan.Resources[1].Action != terraform.ActionUpdate {
		t.Fatalf("expected update action, got %s", plan.Resources[1].Action)
	}
}

func TestTextParserNoChanges(t *testing.T) {
	input := `No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.`

	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 0 {
		t.Fatalf("expected no resources, got %d", len(plan.Resources))
	}
}
