package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

// ProgressCompact renders a compact progress view for streaming operations.
type ProgressCompact struct {
	state  *terraform.OperationState
	styles *styles.Styles
	width  int
	height int
}

// NewProgressCompact creates a compact progress component.
func NewProgressCompact(state *terraform.OperationState, styles *styles.Styles) *ProgressCompact {
	return &ProgressCompact{
		state:  state,
		styles: styles,
	}
}

// SetSize updates the component size.
func (p *ProgressCompact) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// View renders the compact progress display.
func (p *ProgressCompact) View() string {
	if p == nil || p.styles == nil {
		return ""
	}

	current, total, address, action := 0, 0, "", terraform.ActionNoOp
	if p.state != nil {
		current, total, address, action = p.state.GetProgress()
	}

	if total == 0 {
		return p.renderNoProgress()
	}

	barWidth := p.width - 24
	if barWidth < 10 {
		barWidth = 10
	}
	if barWidth > 60 {
		barWidth = 60
	}
	percent := int(float64(current) / float64(total) * 100)
	fill := int(float64(barWidth) * float64(current) / float64(total))
	bar := fmt.Sprintf("[%s%s]", strings.Repeat("#", fill), strings.Repeat("-", barWidth-fill))
	progressLine := fmt.Sprintf("Progress: %s %3d%%", bar, percent)

	currentLine := "Current: waiting for resource updates"
	if address != "" {
		currentLine = fmt.Sprintf("Current: %s %s", action.GetActionIcon(), address)
	}
	statusLine := fmt.Sprintf("Status: %d/%d resources complete", current, total)

	content := strings.Join([]string{
		p.styles.Title.Render("Terraform apply progress"),
		progressLine,
		currentLine,
		statusLine,
	}, "\n")

	return lipgloss.NewStyle().Width(p.width).Height(p.height).Render(content)
}

func (p *ProgressCompact) renderNoProgress() string {
	if p.state == nil {
		return p.renderBox(p.styles.Dimmed.Render("Waiting for terraform updates..."))
	}

	diags := p.state.GetDiagnostics()
	if len(diags) == 0 {
		return p.renderBox(p.styles.Dimmed.Render("Waiting for terraform updates..."))
	}

	latest := latestDiagnostic(diags)
	if latest == nil {
		return p.renderBox(p.styles.Dimmed.Render("Waiting for terraform updates..."))
	}

	summary := strings.TrimSpace(latest.Summary)
	if summary == "" {
		summary = "Unknown error"
	}

	content := strings.Join([]string{
		p.styles.Title.Render("Terraform error"),
		p.styles.DiffRemove.Render("Error: " + summary),
		p.styles.Dimmed.Render("Check logs or diagnostics for details."),
	}, "\n")

	return p.renderBox(content)
}

func (p *ProgressCompact) renderBox(content string) string {
	return lipgloss.NewStyle().Width(p.width).Height(p.height).Render(content)
}

func latestDiagnostic(diags []terraform.Diagnostic) *terraform.Diagnostic {
	for i := len(diags) - 1; i >= 0; i-- {
		if diags[i].Severity == "error" {
			return &diags[i]
		}
	}
	if len(diags) == 0 {
		return nil
	}
	return &diags[len(diags)-1]
}
