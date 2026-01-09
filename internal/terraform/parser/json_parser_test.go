package parser

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestParseBytes(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "plans", "sample.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sample plan: %v", err)
	}

	p := NewJSONParser()
	plan, err := p.ParseBytes(data)
	if err != nil {
		t.Fatalf("parse bytes: %v", err)
	}

	if plan.FormatVersion != "1.2" {
		t.Fatalf("unexpected format version: %s", plan.FormatVersion)
	}
	if plan.Metadata.TerraformVersion != "1.5.0" {
		t.Fatalf("unexpected terraform version: %s", plan.Metadata.TerraformVersion)
	}
	if len(plan.Resources) != 6 {
		t.Fatalf("expected 6 resources, got %d", len(plan.Resources))
	}
}

func TestParseFile(t *testing.T) {
	p := NewJSONParser()
	path := filepath.Join("..", "..", "..", "testdata", "plans", "sample.json")
	plan, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	if plan == nil || len(plan.Resources) == 0 {
		t.Fatalf("expected resources in plan")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	p := NewJSONParser()
	if _, err := p.ParseBytes([]byte(`{invalid`)); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

func TestParse_ReaderFiltersAndOutputs(t *testing.T) {
	raw := []byte(`{
		"format_version":"1.2",
		"terraform_version":"1.5.0",
		"resource_changes":[
			{"address":"aws_vpc.main","change":{"actions":["update"],"before":{},"after":{}}},
			{"address":"aws_iam_role.noop","change":{"actions":["no-op"],"before":{},"after":{}}},
			{"address":"aws_null.nil","change":null}
		],
		"output_changes":{
			"bucket_id":{"actions":["create"],"before":null,"after":"id-123","sensitive":true},
			"region":{"actions":["read"],"before":null,"after":"us-west-2","sensitive":false}
		}
	}`)

	p := NewJSONParser()
	plan, err := p.Parse(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse reader: %v", err)
	}
	if len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(plan.Resources))
	}
	if plan.Resources[0].Address != "aws_vpc.main" {
		t.Fatalf("unexpected resource address: %q", plan.Resources[0].Address)
	}

	if len(plan.OutputChanges) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(plan.OutputChanges))
	}
	outputs := make(map[string]bool)
	for _, out := range plan.OutputChanges {
		outputs[out.Name] = out.Sensitive
	}
	if got := outputs["bucket_id"]; !got {
		t.Fatalf("expected bucket_id to be sensitive")
	}
	if got := outputs["region"]; got {
		t.Fatalf("expected region to be not sensitive")
	}
}

func TestParseFile_Missing(t *testing.T) {
	p := NewJSONParser()
	if _, err := p.ParseFile(filepath.Join(os.TempDir(), "missing-plan.json")); err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestParseBytes_VariablesAndEmptyOutputs(t *testing.T) {
	raw := []byte(`{
		"format_version":"1.2",
		"terraform_version":"1.5.0",
		"resource_changes":[
			{"address":"aws_vpc.main","change":{"actions":["update"],"before":{},"after":{}}}
		],
		"output_changes":{},
		"variables":{"env":"dev"}
	}`)

	p := NewJSONParser()
	plan, err := p.ParseBytes(raw)
	if err != nil {
		t.Fatalf("parse bytes: %v", err)
	}
	if got := plan.Variables["env"]; got != "dev" {
		t.Fatalf("unexpected variables value: %v", got)
	}
	if len(plan.OutputChanges) != 0 {
		t.Fatalf("expected no output changes, got %d", len(plan.OutputChanges))
	}
}

func TestParse_InvalidReader(t *testing.T) {
	p := NewJSONParser()
	if _, err := p.Parse(bytes.NewReader([]byte(`{invalid`))); err == nil {
		t.Fatalf("expected error for invalid json reader")
	}
}

func TestParse_ModuleAddressAndActions(t *testing.T) {
	raw := []byte(`{
		"format_version":"1.2",
		"terraform_version":"1.5.0",
		"resource_changes":[
			{
				"address":"module.foo.aws_instance.one",
				"module_address":"module.foo",
				"type":"aws_instance",
				"name":"one",
				"change":{"actions":["create"],"before":null,"after":{}}
			},
			{
				"address":"module.bar.aws_instance.two",
				"module_address":["module","bar"],
				"type":"aws_instance",
				"name":"two",
				"change":{"actions":["delete","create"],"before":{},"after":{}}
			},
			{
				"address":"data.aws_caller_identity.current",
				"type":"aws_caller_identity",
				"name":"current",
				"change":{"actions":["read"],"before":null,"after":{}}
			}
		],
		"output_changes":{}
	}`)

	p := NewJSONParser()
	plan, err := p.ParseBytes(raw)
	if err != nil {
		t.Fatalf("parse bytes: %v", err)
	}
	if len(plan.Resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(plan.Resources))
	}
	if plan.Resources[0].Action != terraform.ActionCreate {
		t.Fatalf("expected create action")
	}
	if plan.Resources[1].Action != terraform.ActionReplace {
		t.Fatalf("expected replace action")
	}
	if plan.Resources[2].Action != terraform.ActionRead {
		t.Fatalf("expected read action")
	}
	if got := plan.Resources[0].ModulePath; len(got) != 1 || got[0] != "module.foo" {
		t.Fatalf("unexpected module path: %#v", got)
	}
	if got := plan.Resources[1].ModulePath; len(got) != 2 {
		t.Fatalf("unexpected module path: %#v", got)
	}
}
