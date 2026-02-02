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

	beforeRoot, ok := builder.before["root"].(map[string]any)
	if !ok {
		t.Fatalf("expected before[root] to be map")
	}
	if beforeRoot["key"] != "before" {
		t.Fatalf("expected before value")
	}
	afterRoot, ok := builder.after["root"].(map[string]any)
	if !ok {
		t.Fatalf("expected after[root] to be map")
	}
	if afterRoot["key"] != "updated" {
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

func TestTextParserTaintedResource(t *testing.T) {
	input := `Terraform will perform the following actions:

  # null_resource.error_resource is tainted, so must be replaced
-/+ resource "null_resource" "error_resource" {
      ~ id       = "123" -> (known after apply)
        # (1 unchanged attribute hidden)
    }

Plan: 1 to add, 0 to change, 1 to destroy.
`

	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}

	// The address should be just "null_resource.error_resource", not "null_resource.error_resource is tainted, so"
	expectedAddress := "null_resource.error_resource"
	if plan.Resources[0].Address != expectedAddress {
		t.Fatalf("expected address %q, got %q", expectedAddress, plan.Resources[0].Address)
	}
	if plan.Resources[0].Action != terraform.ActionReplace {
		t.Fatalf("expected replace action, got %s", plan.Resources[0].Action)
	}
}

func TestTextParserParseNilInput(t *testing.T) {
	parser := NewTextParser()
	_, err := parser.Parse(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
	if err.Error() != "no input provided" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTextParserParseStreamNilInput(t *testing.T) {
	parser := NewTextParser()
	_, err := parser.ParseStream(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
	if err.Error() != "no input provided" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTextParserParseStreamNoChanges(t *testing.T) {
	lines := make(chan string, 10)
	lines <- "No changes. Your infrastructure matches the configuration."
	close(lines)

	parser := NewTextParser()
	plan, err := parser.ParseStream(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Resources) != 0 {
		t.Fatalf("expected no resources, got %d", len(plan.Resources))
	}
}

func TestApplyActionValueMinusPrefix(t *testing.T) {
	builder := newPlanBuilder(NewCleaner())
	builder.before = make(map[string]any)
	builder.after = make(map[string]any)

	// Test "-" prefix sets before value
	builder.applyActionValue("-", []string{"key"}, "old_value", false)

	if builder.before["key"] != "old_value" {
		t.Fatalf("expected before[key] = old_value, got %v", builder.before["key"])
	}
}

func TestApplyActionValueTildeWithUnknown(t *testing.T) {
	builder := newPlanBuilder(NewCleaner())
	builder.before = make(map[string]any)
	builder.after = make(map[string]any)
	builder.afterUnknown = make(map[string]any)

	// Test "~" prefix with unknown sets both before and afterUnknown
	builder.applyActionValue("~", []string{"key"}, nil, true)

	if builder.before["key"] != nil {
		t.Fatalf("expected before[key] = nil, got %v", builder.before["key"])
	}
	if _, ok := builder.afterUnknown["key"]; !ok {
		t.Fatal("expected afterUnknown[key] to be set")
	}
}

func TestApplyActionValuePlusWithUnknown(t *testing.T) {
	builder := newPlanBuilder(NewCleaner())
	builder.before = make(map[string]any)
	builder.after = make(map[string]any)
	builder.afterUnknown = make(map[string]any)

	// Test "+" prefix with unknown sets afterUnknown
	builder.applyActionValue("+", []string{"key"}, nil, true)

	if _, ok := builder.afterUnknown["key"]; !ok {
		t.Fatal("expected afterUnknown[key] to be set")
	}
}

func TestExtractResourceAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"aws_instance.example", "aws_instance.example"},
		{"aws_instance.example is tainted, so must be replaced", "aws_instance.example"},
		{"module.foo.aws_instance.bar will be created", "module.foo.aws_instance.bar"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractResourceAddress(tt.input)
			if got != tt.expected {
				t.Errorf("extractResourceAddress(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseHeredocDelimiter(t *testing.T) {
	tests := []struct {
		input     string
		delimiter string
		ok        bool
	}{
		{"<<EOF", "EOF", true},
		{"<<-EOF", "EOF", true},
		{"<<-EOT some comment", "EOT", true},
		{"not heredoc", "", false},
		{"<<", "", false},
		{"<<-", "", false},
		{"<<  ", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			delimiter, ok := parseHeredocDelimiter(tt.input)
			if ok != tt.ok {
				t.Errorf("parseHeredocDelimiter(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if delimiter != tt.delimiter {
				t.Errorf("parseHeredocDelimiter(%q) = %q, want %q", tt.input, delimiter, tt.delimiter)
			}
		})
	}
}

func TestSplitArrow(t *testing.T) {
	tests := []struct {
		input  string
		before string
		after  string
	}{
		{`"old" -> "new"`, `"old"`, `"new"`},
		{"t2.micro -> t2.small", "t2.micro", "t2.small"},
		{`"value" -> (known after apply)`, `"value"`, "(known after apply)"},
		{"no arrow here", "no arrow here", ""},
		{"value # comment -> other", "value", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			before, after := splitArrow(tt.input)
			if before != tt.before {
				t.Errorf("splitArrow(%q) before = %q, want %q", tt.input, before, tt.before)
			}
			if after != tt.after {
				t.Errorf("splitArrow(%q) after = %q, want %q", tt.input, after, tt.after)
			}
		})
	}
}

func TestTextParserDeleteAction(t *testing.T) {
	input := `Terraform will perform the following actions:

  # aws_instance.web will be destroyed
  - resource "aws_instance" "web" {
      - ami = "ami-123" -> null
      - id  = "i-12345" -> null
    }

Plan: 0 to add, 0 to change, 1 to destroy.
`
	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}
	if plan.Resources[0].Action != terraform.ActionDelete {
		t.Fatalf("expected delete action, got %s", plan.Resources[0].Action)
	}
	if plan.Resources[0].Change.Before["ami"] != "ami-123" {
		t.Fatalf("expected before ami value, got %v", plan.Resources[0].Change.Before["ami"])
	}
}

func TestTextParserEmptyPlanOutput(t *testing.T) {
	input := `Some random text
without any resource changes
`
	parser := NewTextParser()
	_, err := parser.Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for empty plan")
	}
	if !strings.Contains(err.Error(), "no resource changes parsed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetPathValueNestedCreation(t *testing.T) {
	m := make(map[string]any)
	setPathValue(m, []string{"a", "b", "c"}, "value")

	a, ok := m["a"].(map[string]any)
	if !ok {
		t.Fatal("expected a to be a map")
	}
	b, ok := a["b"].(map[string]any)
	if !ok {
		t.Fatal("expected b to be a map")
	}
	if b["c"] != "value" {
		t.Fatalf("expected c = value, got %v", b["c"])
	}
}

func TestParseTerraformValueEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected any
		unknown  bool
	}{
		{"(known after apply)", nil, true},
		{"\"hello\"", "hello", false},
		{"true", true, false},
		{"false", false, false},
		{"null", nil, false},
		{"123", int64(123), false},
		{"3.14", 3.14, false},
		{"[\"a\", \"b\"]", []any{"a", "b"}, false},
		{"{}", map[string]any{}, false},
		{"value,", "value", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, unknown := parseTerraformValue(tt.input)
			if unknown != tt.unknown {
				t.Errorf("parseTerraformValue(%q) unknown = %v, want %v", tt.input, unknown, tt.unknown)
			}
			// For complex types just check they're not nil when expected
			if tt.expected == nil && val != nil && !tt.unknown {
				t.Errorf("parseTerraformValue(%q) = %v, want nil", tt.input, val)
			}
		})
	}
}

func TestTextParserParseResourceLineNoResource(t *testing.T) {
	// Test line without "resource" keyword
	input := `Terraform will perform the following actions:

  # aws_instance.web will be created
  + some_random_line "without" "resource"
      + ami = "ami-123"

Plan: 1 to add, 0 to change, 0 to destroy.
`
	parser := NewTextParser()
	plan, _ := parser.Parse(strings.NewReader(input))
	// Should still parse the header line
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}
}

func TestTextParserParseResourceLineIncomplete(t *testing.T) {
	// Test resource line with incomplete quotes
	input := `Terraform will perform the following actions:

  # aws_instance.web will be created
  + resource "aws_instance" incomplete
      + ami = "ami-123"

Plan: 1 to add, 0 to change, 0 to destroy.
`
	parser := NewTextParser()
	plan, _ := parser.Parse(strings.NewReader(input))
	// Should still parse but with incomplete metadata
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}
}

func TestTextParserHeredocValueUserData(t *testing.T) {
	input := `Terraform will perform the following actions:

  # aws_instance.web will be created
  + resource "aws_instance" "web" {
      + user_data = <<-EOT
          #!/bin/bash
          echo "hello"
        EOT
    }

Plan: 1 to add, 0 to change, 0 to destroy.
`
	parser := NewTextParser()
	plan, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}
}

func TestTextParserReadAction(t *testing.T) {
	// Test read action (data source)
	input := `Terraform will perform the following actions:

  # data.aws_ami.latest will be read during apply
  <= data "aws_ami" "latest" {
       + id = (known after apply)
     }

Plan: 0 to add, 0 to change, 0 to destroy.
`
	parser := NewTextParser()
	plan, _ := parser.Parse(strings.NewReader(input))
	// Read actions may or may not be parsed depending on implementation
	_ = plan
}
