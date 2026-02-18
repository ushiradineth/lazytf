package ui

import (
	"fmt"
	"strings"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

// BuildDriftAnalysisView renders a compact drift summary from plan resources.
func BuildDriftAnalysisView(resources []terraform.ResourceChange) string {
	if len(resources) == 0 {
		return "No resources in plan."
	}

	candidates := driftCandidates(resources)
	if len(candidates) == 0 {
		return "No drift candidates detected.\n\nTip: run refresh-only plan to verify external drift."
	}

	lines := make([]string, 0, len(candidates)+3)
	lines = append(lines, fmt.Sprintf("Drift candidates: %d", len(candidates)), "")
	for i := range candidates {
		r := candidates[i]
		reason := strings.TrimSpace(r.ActionReason)
		if reason == "" {
			reason = "resource differs from last known state"
		}
		lines = append(lines, fmt.Sprintf("- %s [%s]", r.Address, r.Action))
		lines = append(lines, "  "+reason)
	}

	return strings.Join(lines, "\n")
}

func driftCandidates(resources []terraform.ResourceChange) []terraform.ResourceChange {
	result := make([]terraform.ResourceChange, 0, len(resources))
	for i := range resources {
		r := resources[i]
		switch r.Action {
		case terraform.ActionUpdate, terraform.ActionDelete, terraform.ActionReplace:
			result = append(result, r)
		case terraform.ActionCreate, terraform.ActionNoOp, terraform.ActionRead:
			continue
		}
	}
	return result
}
