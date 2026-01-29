package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// ApplyStatus tracks the apply state.
type ApplyStatus int

const (
	ApplyPending ApplyStatus = iota
	ApplyRunning
	ApplySuccess
	ApplyFailed
)

// ApplyView renders streaming terraform output.
type ApplyView struct {
	viewport    viewport.Model
	outputLines []string
	status      ApplyStatus
	spinner     spinner.Model
	progress    string
	styles      *styles.Styles
	title       string
	statusText  statusText
	width       int
	maxLines    int
}

type statusText struct {
	running string
	success string
	failure string
	pending string
}

// NewApplyView creates a new apply view.
func NewApplyView(s *styles.Styles) *ApplyView {
	sp := spinner.New()
	sp.Spinner = spinner.Spinner{
		Frames: []string{"-", "\\", "|", "/"},
		FPS:    120000000,
	}

	vp := viewport.New(0, 0)
	return &ApplyView{
		viewport: vp,
		spinner:  sp,
		status:   ApplyPending,
		styles:   s,
		maxLines: 10000,
		statusText: statusText{
			running: "Running...",
			success: "Apply complete",
			failure: "Apply failed",
			pending: "Waiting",
		},
	}
}

// SetStyles updates the component styles.
func (v *ApplyView) SetStyles(s *styles.Styles) {
	v.styles = s
}

// SetSize updates the layout size.
func (v *ApplyView) SetSize(width, height int) {
	v.width = width
	headerHeight := 1
	footerHeight := 1
	bodyHeight := height - headerHeight - footerHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	v.viewport.Width = width
	v.viewport.Height = bodyHeight
	v.refreshViewport()
}

// SetTitle updates the header title.
func (v *ApplyView) SetTitle(title string) {
	v.title = title
}

// SetStatusText customizes footer labels.
func (v *ApplyView) SetStatusText(running, success, failure string) {
	if running != "" {
		v.statusText.running = running
	}
	if success != "" {
		v.statusText.success = success
	}
	if failure != "" {
		v.statusText.failure = failure
	}
}

// SetStatus updates the apply status.
func (v *ApplyView) SetStatus(status ApplyStatus) {
	v.status = status
}

// SetProgress updates the progress footer.
func (v *ApplyView) SetProgress(progress string) {
	v.progress = progress
}

// Reset clears output and resets status.
func (v *ApplyView) Reset() {
	v.outputLines = nil
	v.progress = ""
	v.status = ApplyPending
	v.viewport.SetContent("")
}

// SetOutput replaces the output content.
func (v *ApplyView) SetOutput(output string) {
	output = strings.TrimRight(output, "\n")
	if output == "" {
		v.outputLines = nil
		v.viewport.SetContent("")
		return
	}
	v.outputLines = strings.Split(output, "\n")
	if v.maxLines > 0 && len(v.outputLines) > v.maxLines {
		v.outputLines = v.outputLines[len(v.outputLines)-v.maxLines:]
	}
	v.refreshViewport()
}

// AppendLine adds a new output line.
func (v *ApplyView) AppendLine(line string) {
	if v.maxLines > 0 && len(v.outputLines) >= v.maxLines {
		v.outputLines = v.outputLines[len(v.outputLines)-v.maxLines+1:]
	}
	v.outputLines = append(v.outputLines, line)
	v.refreshViewport()
}

// Update handles spinner and viewport messages.
func (v *ApplyView) Update(msg tea.Msg) (*ApplyView, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if v.status == ApplyRunning {
			v.spinner, cmd = v.spinner.Update(msg)
		}
	case tea.KeyMsg:
		v.viewport, cmd = v.viewport.Update(msg)
	default:
		v.viewport, cmd = v.viewport.Update(msg)
	}
	return v, cmd
}

// Tick returns the spinner tick command.
func (v *ApplyView) Tick() tea.Cmd {
	if v == nil {
		return nil
	}
	return v.spinner.Tick
}

// View renders the apply output.
func (v *ApplyView) View() string {
	if v.styles == nil {
		return ""
	}
	header := v.renderHeader()
	body := v.viewport.View()
	if v.width > 0 {
		body = lipgloss.NewStyle().Width(v.width).Height(v.viewport.Height).Render(body)
	}
	footer := v.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (v *ApplyView) renderHeader() string {
	title := v.title
	if title == "" {
		title = "Applying changes..."
	}
	label := title
	switch v.status {
	case ApplyRunning:
		label = fmt.Sprintf("%s %s", v.spinner.View(), title)
	case ApplySuccess:
		label = "OK " + title
	case ApplyFailed:
		label = "ERR " + title
	case ApplyPending:
		// Keep default label
	}
	if v.width > 0 {
		return v.styles.Title.Width(v.width).Render(label)
	}
	return v.styles.Title.Render(label)
}

func (v *ApplyView) renderFooter() string {
	var footer string
	switch v.status {
	case ApplyRunning:
		if v.progress != "" {
			footer = "Progress: " + v.progress
		} else {
			footer = v.statusText.running
		}
	case ApplySuccess:
		footer = v.statusText.success
	case ApplyFailed:
		footer = v.statusText.failure
	case ApplyPending:
		footer = v.statusText.pending
	}
	if v.status == ApplySuccess || v.status == ApplyFailed {
		if footer != "" {
			footer += " | esc: back | q: quit"
		} else {
			footer = "esc: back | q: quit"
		}
	}
	if v.width > 0 {
		return v.styles.StatusBar.Width(v.width).Render(footer)
	}
	return v.styles.StatusBar.Render(footer)
}

func (v *ApplyView) refreshViewport() {
	content := strings.Join(v.outputLines, "\n")
	v.viewport.SetContent(content)
	v.viewport.GotoBottom()
}
