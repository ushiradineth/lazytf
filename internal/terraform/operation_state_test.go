package terraform

import "testing"

func TestOperationStateLifecycle(t *testing.T) {
	state := NewOperationState()
	state.InitializeFromPlan(&Plan{
		Resources: []ResourceChange{
			{Address: "aws_instance.web", Action: ActionCreate},
			{Address: "aws_s3_bucket.data", Action: ActionUpdate},
		},
	})

	current, total, currentAddress, currentAction := state.GetProgress()
	if total != 2 || current != 0 {
		t.Fatalf("unexpected progress: %d/%d", current, total)
	}
	if currentAddress != "" || currentAction != ActionNoOp {
		t.Fatalf("expected no current resource, got %q/%s", currentAddress, currentAction)
	}

	state.StartResource("aws_instance.web", ActionCreate)
	if op := state.GetResourceStatus("aws_instance.web"); op == nil || op.Status != StatusInProgress {
		t.Fatalf("expected in-progress resource, got %#v", op)
	}
	current, total, currentAddress, currentAction = state.GetProgress()
	if total != 2 || current != 0 {
		t.Fatalf("unexpected progress after start: %d/%d", current, total)
	}
	if currentAddress != "aws_instance.web" || currentAction != ActionCreate {
		t.Fatalf("unexpected current resource: %q/%s", currentAddress, currentAction)
	}

	state.CompleteResource("aws_instance.web", "i-123")
	current, total, currentAddress, currentAction = state.GetProgress()
	if total != 2 || current != 1 {
		t.Fatalf("unexpected progress after completion: %d/%d", current, total)
	}
	if currentAddress != "" || currentAction != ActionNoOp {
		t.Fatalf("unexpected current resource after completion: %q/%s", currentAddress, currentAction)
	}
	if op := state.GetResourceStatus("aws_instance.web"); op == nil || op.Status != StatusComplete || op.IDValue != "i-123" {
		t.Fatalf("unexpected completion state: %#v", op)
	}

	state.ErrorResource("aws_s3_bucket.data", errTest("failed"))
	current, total, currentAddress, currentAction = state.GetProgress()
	if total != 2 || current != 2 {
		t.Fatalf("unexpected progress after error: %d/%d", current, total)
	}
	if currentAddress != "" || currentAction != ActionNoOp {
		t.Fatalf("unexpected current resource after error: %q/%s", currentAddress, currentAction)
	}
}

type errTest string

func (e errTest) Error() string {
	return string(e)
}

func TestOperationStateDiagnostics(t *testing.T) {
	state := NewOperationState()
	state.AddDiagnostic(Diagnostic{Severity: "error", Summary: "bad"})
	diags := state.GetDiagnostics()
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic")
	}
	if diags[0].Summary != "bad" {
		t.Fatalf("unexpected diagnostic summary")
	}
}

func TestOperationStateErrorResourceIgnoresEmpty(t *testing.T) {
	state := NewOperationState()
	state.ErrorResource("", errTest("bad"))
	if _, total, _, _ := state.GetProgress(); total != 0 {
		t.Fatalf("expected no resources for empty address")
	}
}

func TestParseApplyLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantAddress string
		wantStatus  OperationStatus
		wantAction  ActionType
		wantIDValue string
	}{
		{
			name:        "creating start",
			line:        "null_resource.example: Creating...",
			wantAddress: "null_resource.example",
			wantStatus:  StatusInProgress,
			wantAction:  ActionCreate,
		},
		{
			name:        "destroying start",
			line:        "null_resource.foo: Destroying...",
			wantAddress: "null_resource.foo",
			wantStatus:  StatusInProgress,
			wantAction:  ActionDelete,
		},
		{
			name:        "destroying start with id",
			line:        "null_resource.error_resource: Destroying... [id=8120145320752542536]",
			wantAddress: "null_resource.error_resource",
			wantStatus:  StatusInProgress,
			wantAction:  ActionDelete,
		},
		{
			name:        "modifying start",
			line:        "aws_instance.web: Modifying...",
			wantAddress: "aws_instance.web",
			wantStatus:  StatusInProgress,
			wantAction:  ActionUpdate,
		},
		{
			name:        "creation complete with id",
			line:        "null_resource.example: Creation complete after 0s [id=1234567890]",
			wantAddress: "null_resource.example",
			wantStatus:  StatusComplete,
			wantIDValue: "1234567890",
		},
		{
			name:        "destruction complete",
			line:        "null_resource.foo: Destruction complete after 1s",
			wantAddress: "null_resource.foo",
			wantStatus:  StatusComplete,
		},
		{
			name:        "modifications complete",
			line:        "aws_instance.web: Modifications complete after 5s [id=i-abc123]",
			wantAddress: "aws_instance.web",
			wantStatus:  StatusComplete,
			wantIDValue: "i-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewOperationState()
			state.ParseApplyLine(tt.line)

			op := state.GetResourceStatus(tt.wantAddress)
			if op == nil {
				t.Fatalf("expected resource %q to be tracked", tt.wantAddress)
			}
			if op.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", op.Status, tt.wantStatus)
			}
			if tt.wantAction != ActionNoOp && op.Action != tt.wantAction {
				t.Errorf("action = %v, want %v", op.Action, tt.wantAction)
			}
			if tt.wantIDValue != "" && op.IDValue != tt.wantIDValue {
				t.Errorf("idValue = %q, want %q", op.IDValue, tt.wantIDValue)
			}
		})
	}
}

func TestParseApplyLineIgnoresIrrelevant(t *testing.T) {
	state := NewOperationState()
	state.ParseApplyLine("Terraform will perform the following actions:")
	state.ParseApplyLine("")
	state.ParseApplyLine("  # null_resource.example will be created")

	if _, total, _, _ := state.GetProgress(); total != 0 {
		t.Fatalf("expected no resources from irrelevant lines, got %d", total)
	}
}

func TestParseApplyLineWithANSI(t *testing.T) {
	state := NewOperationState()

	// Line with ANSI color codes
	state.ParseApplyLine("\x1b[0m\x1b[1mnull_resource.example\x1b[0m: Creating...")
	op := state.GetResourceStatus("null_resource.example")
	if op == nil || op.Status != StatusInProgress {
		t.Fatalf("expected in-progress status with ANSI stripped, got %v", op)
	}

	// Complete with ANSI
	state.ParseApplyLine("\x1b[32mnull_resource.example: Creation complete after 1s [id=123]\x1b[0m")
	op = state.GetResourceStatus("null_resource.example")
	if op == nil || op.Status != StatusComplete {
		t.Fatalf("expected complete status with ANSI stripped, got %v", op)
	}
}

func TestParseApplyLineError(t *testing.T) {
	state := NewOperationState()

	// Start a resource
	state.ParseApplyLine("null_resource.error_resource: Creating...")
	op := state.GetResourceStatus("null_resource.error_resource")
	if op == nil || op.Status != StatusInProgress {
		t.Fatalf("expected in-progress status, got %v", op)
	}

	// Error occurs
	state.ParseApplyLine("Error: local-exec provisioner error")
	op = state.GetResourceStatus("null_resource.error_resource")
	if op == nil || op.Status != StatusErrored {
		t.Fatalf("expected errored status, got %v", op)
	}
}

func TestParseApplyLineApplyFailed(t *testing.T) {
	state := NewOperationState()

	// Start a resource
	state.ParseApplyLine("null_resource.error_resource: Creating...")
	op := state.GetResourceStatus("null_resource.error_resource")
	if op == nil || op.Status != StatusInProgress {
		t.Fatalf("expected in-progress status, got %v", op)
	}

	// Apply failed
	state.ParseApplyLine("Apply failed: exit status 1")
	op = state.GetResourceStatus("null_resource.error_resource")
	if op == nil || op.Status != StatusErrored {
		t.Fatalf("expected errored status after Apply failed, got %v", op)
	}
}

func TestParseApplyLineErrorWithBoxDrawing(t *testing.T) {
	state := NewOperationState()

	// Start a resource
	state.ParseApplyLine("null_resource.error_resource: Creating...")
	op := state.GetResourceStatus("null_resource.error_resource")
	if op == nil || op.Status != StatusInProgress {
		t.Fatalf("expected in-progress status, got %v", op)
	}

	// Error with box drawing character prefix (terraform formats errors this way)
	state.ParseApplyLine("│ Error: local-exec provisioner error")
	op = state.GetResourceStatus("null_resource.error_resource")
	if op == nil || op.Status != StatusErrored {
		t.Fatalf("expected errored status after Error with box drawing, got %v", op)
	}
}
