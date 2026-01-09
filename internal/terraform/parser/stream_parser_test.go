package parser

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestStreamParserParsesMessagesAndAccumulatesPlan(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "terraform", "streaming", "full_apply_stream.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	parser := NewStreamParser()
	msgChan := make(chan terraform.StreamMessage, 20)
	go func() {
		_ = parser.Parse(bytes.NewReader(data), msgChan)
		close(msgChan)
	}()

	types := make(map[terraform.StreamMessageType]int)
	for msg := range msgChan {
		types[msg.Type]++
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
