package parser

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/terraform"
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

func TestTextParserParsesReplaceAndNested(t *testing.T) {
	input := `Terraform will perform the following actions:

  # aws_instance.web must be replaced
  -/+ resource "aws_instance" "web" {
      ~ instance_type = "t2.micro" -> "t2.small"
    }

  # aws_vpc.main will be updated in-place
  ~ resource "aws_vpc" "main" {
      ~ tags = {
          + Name = "new"
        }
    }

Plan: 1 to add, 1 to change, 1 to destroy.
`

	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(plan.Resources))
	}
	if plan.Resources[0].Action != terraform.ActionReplace {
		t.Fatalf("expected replace action, got %s", plan.Resources[0].Action)
	}
	if plan.Resources[1].Change == nil || plan.Resources[1].Change.After == nil {
		t.Fatalf("expected change data for nested tags")
	}
	tags, ok := plan.Resources[1].Change.After["tags"].(map[string]any)
	if !ok || tags["Name"] != "new" {
		t.Fatalf("expected nested tag to be parsed")
	}
}

func TestTextParserKnownAfterApply(t *testing.T) {
	input := `Terraform will perform the following actions:

  # aws_instance.web will be created
  + resource "aws_instance" "web" {
      + id = (known after apply)
    }
`
	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}
	if plan.Resources[0].Change == nil || plan.Resources[0].Change.AfterUnknown == nil {
		t.Fatalf("expected after_unknown map to be set")
	}
	if _, ok := plan.Resources[0].Change.AfterUnknown["id"]; !ok {
		t.Fatalf("expected id to be marked as unknown")
	}
}

func TestTextParserModuleForEachAddress(t *testing.T) {
	input := `Terraform will perform the following actions:

  # module.foo.aws_instance.bar["blue"] will be created
  + resource "aws_instance" "bar" {
      + ami = "ami-123"
    }
`
	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if got := plan.Resources[0].Address; got != `module.foo.aws_instance.bar["blue"]` {
		t.Fatalf("unexpected address: %q", got)
	}
}

func TestTextParserReadDuringApply(t *testing.T) {
	input := `Terraform will perform the following actions:

  # data.aws_caller_identity.current will be read during apply
  <= data "aws_caller_identity" "current" {
      + account_id = (known after apply)
    }
`
	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}
	if plan.Resources[0].Action != terraform.ActionRead {
		t.Fatalf("expected read action, got %s", plan.Resources[0].Action)
	}
}

func TestTextParserHeredocValue(t *testing.T) {
	input := `Terraform will perform the following actions:

  # aws_instance.web will be created
  + resource "aws_instance" "web" {
      + user_data = <<-EOF
line1
line2
EOF
    }
`
	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}
	value, ok := plan.Resources[0].Change.After["user_data"].(string)
	if !ok {
		t.Fatalf("expected heredoc string")
	}
	if value != "line1\nline2" {
		t.Fatalf("unexpected heredoc value: %q", value)
	}
}

func TestTextParserNestedBlocksAndListValues(t *testing.T) {
	input := `Terraform will perform the following actions:

  # aws_instance.web will be updated in-place
  ~ resource "aws_instance" "web" {
      ~ settings = {
          + nested = {
              + items = ["a", "b"]
            }
        }
    }
`
	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	nested, ok := plan.Resources[0].Change.After["settings"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map")
	}
	deeper, ok := nested["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map under settings")
	}
	if _, ok := deeper["items"]; !ok {
		t.Fatalf("expected items to be parsed")
	}
}

func TestParseTerraformValueAndComments(t *testing.T) {
	got, _ := parseTerraformValue("true")
	gotBool, ok := got.(bool)
	if !ok || !gotBool {
		t.Fatalf("expected true")
	}
	if got, _ := parseTerraformValue("null"); got != nil {
		t.Fatalf("expected nil")
	}
	if got, _ := parseTerraformValue("3.14"); got == nil {
		t.Fatalf("expected numeric value")
	}
	if got, _ := parseTerraformValue("\"value\""); got != "value" {
		t.Fatalf("expected string value")
	}
	if got := stripInlineComment("value # comment"); got != "value" {
		t.Fatalf("expected comment to be stripped")
	}
}

func TestApplyHeredocValuePrefixes(t *testing.T) {
	builder := newPlanBuilder(NewCleaner())
	builder.before = make(map[string]any)
	builder.after = make(map[string]any)
	builder.pathStack = []string{"root"}

	builder.applyHeredocValue("key", "-", "before")
	builder.applyHeredocValue("key", "+", "after")
	builder.applyHeredocValue("key", "~", "updated")

	if builder.before["root"].(map[string]any)["key"] != "before" {
		t.Fatalf("expected before value")
	}
	if builder.after["root"].(map[string]any)["key"] != "updated" {
		t.Fatalf("expected after value to be updated")
	}
}

func TestTextParserParseStreamMatchesParse(t *testing.T) {
	input := `Terraform will perform the following actions:

  # aws_instance.web will be created
  + resource "aws_instance" "web" {
      + ami = "ami-123"
    }
`
	parser := NewTextParser()
	fromReader, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse reader: %v", err)
	}

	split := strings.Split(input, "\n")
	lines := make(chan string, len(split))
	for _, line := range split {
		lines <- line
	}
	close(lines)
	fromStream, err := parser.ParseStream(lines)
	if err != nil {
		t.Fatalf("parse stream: %v", err)
	}
	if len(fromReader.Resources) != len(fromStream.Resources) {
		t.Fatalf("expected same resource count")
	}
	if fromReader.Resources[0].Address != fromStream.Resources[0].Address {
		t.Fatalf("expected same address")
	}
}
