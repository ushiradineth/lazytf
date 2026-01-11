package parser

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestStreamParserParsesMessagesAndAccumulatesPlan(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "terraform", "streaming", "full_apply_stream.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	parser := NewStreamParser()
	msgChan := make(chan terraform.StreamMessage, 20)
	errChan := make(chan error, 1)
	go func() {
		errChan <- parser.Parse(bytes.NewReader(data), msgChan)
		close(msgChan)
	}()

	types := make(map[terraform.StreamMessageType]int)
	for msg := range msgChan {
		types[msg.Type]++
	}
	if parseErr := <-errChan; parseErr != nil {
		t.Fatalf("parse stream: %v", parseErr)
	}

	if types[terraform.MessageTypePlannedChange] == 0 {
		t.Fatalf("expected planned_change message")
	}
	if types[terraform.MessageTypeApplyStart] == 0 {
		t.Fatalf("expected apply_start message")
	}
	if types[terraform.MessageTypeApplyComplete] == 0 {
		t.Fatalf("expected apply_complete message")
	}
	if types[terraform.MessageTypeDiagnostic] == 0 {
		t.Fatalf("expected diagnostic message")
	}
	if types[terraform.MessageTypeOutputs] == 0 {
		t.Fatalf("expected outputs message")
	}

	plan := parser.GetAccumulatedPlan()
	if plan == nil || len(plan.Resources) != 1 {
		t.Fatalf("expected plan with 1 resource, got %#v", plan)
	}
	if plan.Resources[0].Address != "aws_instance.web" {
		t.Fatalf("unexpected resource address: %q", plan.Resources[0].Address)
	}
	if len(plan.OutputChanges) != 1 || plan.OutputChanges[0].Name != "public_ip" {
		t.Fatalf("unexpected output changes: %#v", plan.OutputChanges)
	}
}

func TestStreamParserPlannedChangeActionFallback(t *testing.T) {
	line := `{"type":"planned_change","change":{"resource":{"addr":"null_resource.example","module":"","resource":"null_resource.example","implied_provider":"null","resource_type":"null_resource","resource_name":"example","resource_key":null},"action":"create"}}`
	parser := NewStreamParser()
	msgChan := make(chan terraform.StreamMessage, 1)
	if err := parser.Parse(bytes.NewReader([]byte(line+"\n")), msgChan); err != nil {
		t.Fatalf("parse: %v", err)
	}
	close(msgChan)
	plan := parser.GetAccumulatedPlan()
	if plan == nil || len(plan.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %#v", plan)
	}
	if plan.Resources[0].Action != terraform.ActionCreate {
		t.Fatalf("expected create action, got %s", plan.Resources[0].Action)
	}
	if plan.Resources[0].Address != "null_resource.example" {
		t.Fatalf("unexpected address: %q", plan.Resources[0].Address)
	}
}

func TestParseStreamMessageApplyStartHook(t *testing.T) {
	line := `{"type":"apply_start","hook":{"resource":{"address":"aws_instance.web","type":"aws_instance","name":"web"},"action":"create"}}`
	msg, err := parseStreamMessage([]byte(line))
	if err != nil {
		t.Fatalf("parse stream message: %v", err)
	}
	if msg.Type != terraform.MessageTypeApplyStart || msg.Hook == nil {
		t.Fatalf("expected apply_start hook message")
	}
	if msg.Hook.Resource.Address != "aws_instance.web" {
		t.Fatalf("unexpected hook address: %q", msg.Hook.Resource.Address)
	}
}

func TestDecodeHookFallback(t *testing.T) {
	line := `{"address":"aws_instance.web","action":"create"}`
	hook, err := decodeHook([]byte(line))
	if err != nil {
		t.Fatalf("decode hook: %v", err)
	}
	if hook.Address != "aws_instance.web" {
		t.Fatalf("unexpected hook address: %q", hook.Address)
	}
}

func TestParseStreamMessageOutputsAndDrift(t *testing.T) {
	line := `{"type":"outputs","outputs":{"value":{"actions":["create"],"after":"ok"}}}`
	msg, err := parseStreamMessage([]byte(line))
	if err != nil {
		t.Fatalf("parse outputs: %v", err)
	}
	if msg.Outputs == nil || msg.Outputs["value"].After != "ok" {
		t.Fatalf("unexpected outputs message")
	}

	line = `{"type":"resource_drift","change":{"actions":["delete"]},"resource":{"addr":"aws_instance.web"}}`
	msg, err = parseStreamMessage([]byte(line))
	if err != nil {
		t.Fatalf("parse drift: %v", err)
	}
	if msg.ResourceDrift == nil || msg.ResourceDrift.Resource.Address != "aws_instance.web" {
		t.Fatalf("unexpected drift message")
	}
}
