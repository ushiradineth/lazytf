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

	current, total, _, _ := state.GetProgress()
	if total != 2 || current != 0 {
		t.Fatalf("unexpected progress: %d/%d", current, total)
	}

	state.StartResource("aws_instance.web", ActionCreate)
	if op := state.GetResourceStatus("aws_instance.web"); op == nil || op.Status != StatusInProgress {
		t.Fatalf("expected in-progress resource, got %#v", op)
	}

	state.CompleteResource("aws_instance.web", "i-123")
	current, total, _, _ = state.GetProgress()
	if total != 2 || current != 1 {
		t.Fatalf("unexpected progress after completion: %d/%d", current, total)
	}
	if op := state.GetResourceStatus("aws_instance.web"); op == nil || op.Status != StatusComplete || op.IDValue != "i-123" {
		t.Fatalf("unexpected completion state: %#v", op)
	}

	state.ErrorResource("aws_s3_bucket.data", errTest("failed"))
	current, total, _, _ = state.GetProgress()
	if total != 2 || current != 2 {
		t.Fatalf("unexpected progress after error: %d/%d", current, total)
	}
}

type errTest string

func (e errTest) Error() string {
	return string(e)
}
