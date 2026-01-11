package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestProgressCompactWaitingState(t *testing.T) {
	view := NewProgressCompact(nil, styles.DefaultStyles())
	view.SetSize(40, 5)

	out := view.View()
	if !strings.Contains(out, "Waiting for terraform updates") {
		t.Fatalf("expected waiting message")
	}
}

func TestProgressCompactWithProgress(t *testing.T) {
	state := terraform.NewOperationState()
	state.StartResource("aws_instance.web", terraform.ActionCreate)
	state.CompleteResource("aws_instance.web", "id")
	state.StartResource("aws_instance.db", terraform.ActionUpdate)

	view := NewProgressCompact(state, styles.DefaultStyles())
	view.SetSize(60, 6)

	out := view.View()
	if !strings.Contains(out, "Progress:") {
		t.Fatalf("expected progress line")
	}
	if !strings.Contains(out, "Status:") {
		t.Fatalf("expected status line")
	}
}

func TestProgressCompactShowsDiagnostics(t *testing.T) {
	state := terraform.NewOperationState()
	state.AddDiagnostic(terraform.Diagnostic{
		Severity: "error",
		Summary:  "Plan failed",
		Detail:   "exit status 1",
	})

	view := NewProgressCompact(state, styles.DefaultStyles())
	view.SetSize(60, 5)

	out := view.View()
	if !strings.Contains(out, "Plan failed") {
		t.Fatalf("expected diagnostic summary")
	}
	if strings.Contains(out, "Waiting for terraform updates") {
		t.Fatalf("did not expect waiting message")
	}
}
