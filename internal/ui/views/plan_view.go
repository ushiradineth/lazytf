package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/utils"
)

// PlanView renders an apply confirmation dialog.
type PlanView struct {
	summary string
	styles  *styles.Styles
	width   int
	height  int
}

// NewPlanView creates a new plan confirmation view.
func NewPlanView(summary string, s *styles.Styles) *PlanView {
	return &PlanView{
		summary: summary,
		styles:  s,
	}
}

// SetSize updates the view size for centering.
func (v *PlanView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetSummary updates the plan summary text.
func (v *PlanView) SetSummary(summary string) {
	v.summary = summary
}

// View renders the confirmation dialog.
func (v *PlanView) View() string {
	if v.styles == nil {
		return ""
	}

	lines := []string{
		v.styles.Highlight.Render("Confirm Apply"),
		"",
		"Plan summary:",
	}
	for _, line := range strings.Split(v.summary, "\n") {
		lines = append(lines, "  "+line)
	}
	lines = append(lines,
		"",
		"Do you want to apply these changes?",
		"",
		"[Y] Yes, apply    [N] No, cancel",
	)

	content := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	box := v.styles.Border.Width(utils.MinInt(50, v.width-4)).Render(content)
	if v.width == 0 || v.height == 0 {
		return box
	}
	return lipgloss.Place(v.width, v.height, lipgloss.Center, lipgloss.Center, box)
}
