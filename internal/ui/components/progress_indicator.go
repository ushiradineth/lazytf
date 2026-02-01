package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// ProgressState represents the current state of the progress indicator.
type ProgressState int

const (
	// ProgressIdle means no operation is running (not rendered).
	ProgressIdle ProgressState = iota
	// ProgressRunning means an operation is in progress (animated).
	ProgressRunning
	// ProgressFailed means the last operation failed (static red).
	ProgressFailed
)

// ProgressOperation represents the type of operation being tracked.
type ProgressOperation int

const (
	// OperationNone means no operation.
	OperationNone ProgressOperation = iota
	// OperationPlan means terraform plan.
	OperationPlan
	// OperationApply means terraform apply.
	OperationApply
	// OperationRefresh means terraform refresh.
	OperationRefresh
	// OperationValidate means terraform validate.
	OperationValidate
	// OperationFormat means terraform fmt.
	OperationFormat
	// OperationStateList means terraform state list.
	OperationStateList
)

// ProgressIndicator displays command execution status in the footer.
type ProgressIndicator struct {
	styles    *styles.Styles
	state     ProgressState
	operation ProgressOperation
	spinner   spinner.Model
}

// NewProgressIndicator creates a new progress indicator.
func NewProgressIndicator(s *styles.Styles) *ProgressIndicator {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return &ProgressIndicator{
		styles:    s,
		state:     ProgressIdle,
		operation: OperationNone,
		spinner:   sp,
	}
}

// Start begins tracking an operation and returns a tick command.
func (p *ProgressIndicator) Start(op ProgressOperation) tea.Cmd {
	p.state = ProgressRunning
	p.operation = op
	return p.spinner.Tick
}

// Fail marks the current operation as failed.
func (p *ProgressIndicator) Fail() {
	p.state = ProgressFailed
}

// Reset clears the indicator (returns to idle).
func (p *ProgressIndicator) Reset() {
	p.state = ProgressIdle
	p.operation = OperationNone
}

// Update handles tick messages for animation.
func (p *ProgressIndicator) Update(msg tea.Msg) tea.Cmd {
	if p.state != ProgressRunning {
		return nil
	}
	if _, ok := msg.(spinner.TickMsg); ok {
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		return cmd
	}
	return nil
}

// View renders the progress indicator.
func (p *ProgressIndicator) View() string {
	if p.styles == nil || p.state == ProgressIdle {
		return ""
	}

	icon, text := p.getIconAndText()
	if text == "" {
		return ""
	}

	switch p.state {
	case ProgressRunning:
		return p.styles.DiffChange.Render(p.spinner.View() + text)
	case ProgressFailed:
		return p.styles.DiffRemove.Render(icon + " " + text)
	default:
		return ""
	}
}

// SetStyles updates the styles.
func (p *ProgressIndicator) SetStyles(s *styles.Styles) {
	p.styles = s
}

func (p *ProgressIndicator) getIconAndText() (string, string) {
	var runningText, failedText string

	switch p.operation {
	case OperationPlan:
		runningText = "Running Plan"
		failedText = "Plan Failed"
	case OperationApply:
		runningText = "Applying"
		failedText = "Apply Failed"
	case OperationRefresh:
		runningText = "Refreshing State"
		failedText = "Refresh Failed"
	case OperationValidate:
		runningText = "Validating"
		failedText = "Validation Failed"
	case OperationFormat:
		runningText = "Formatting"
		failedText = "Format Failed"
	case OperationStateList:
		runningText = "Loading State"
		failedText = "State Load Failed"
	default:
		return "", ""
	}

	switch p.state {
	case ProgressRunning:
		return "", runningText
	case ProgressFailed:
		return "●", failedText
	default:
		return "", ""
	}
}
