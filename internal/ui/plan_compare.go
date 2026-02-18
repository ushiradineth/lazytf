package ui

import (
	"fmt"
	"strings"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

type planSummary struct {
	Create  int
	Update  int
	Delete  int
	Replace int
}

func summarizePlan(plan *terraform.Plan) planSummary {
	if plan == nil {
		return planSummary{}
	}
	var s planSummary
	for i := range plan.Resources {
		switch plan.Resources[i].Action {
		case terraform.ActionCreate:
			s.Create++
		case terraform.ActionUpdate:
			s.Update++
		case terraform.ActionDelete:
			s.Delete++
		case terraform.ActionReplace:
			s.Replace++
		case terraform.ActionNoOp, terraform.ActionRead:
			continue
		}
	}
	return s
}

func buildPlanCompareView(current planSummary, previous *history.OperationEntry) string {
	lines := []string{
		"Current plan",
		fmt.Sprintf("  create: %d", current.Create),
		fmt.Sprintf("  update: %d", current.Update),
		fmt.Sprintf("  delete: %d", current.Delete),
		fmt.Sprintf("  replace: %d", current.Replace),
	}

	if previous == nil {
		lines = append(lines, "", "No previous plan operation found for comparison.")
		return strings.Join(lines, "\n")
	}

	lines = append(lines,
		"",
		"Previous plan operation",
		"  time: "+previous.FinishedAt.Local().Format("2006-01-02 15:04:05"),
		"  summary: "+strings.TrimSpace(previous.Summary),
	)

	return strings.Join(lines, "\n")
}

func (m *Model) latestPlanOperation() (*history.OperationEntry, bool, error) {
	if m.historyStore == nil {
		return nil, false, nil
	}
	filter := history.OperationFilter{Action: "plan", Limit: 1}
	if m.envCurrent != "" {
		filter.Environment = m.envCurrent
	}
	entries, err := m.historyStore.QueryOperations(filter)
	if err != nil {
		return nil, false, err
	}
	if len(entries) == 0 {
		return nil, false, nil
	}
	return &entries[0], true, nil
}
