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
