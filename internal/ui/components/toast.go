package components

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// ToastLevel represents the severity/type of a toast notification.
type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// ToastPosition represents where the toast should appear.
type ToastPosition int

const (
	ToastTopLeft ToastPosition = iota
	ToastTopRight
	ToastBottomLeft
	ToastBottomRight
)

// Toast renders a non-blocking notification overlay.
// It appears in a corner of the screen without blocking the main content.
type Toast struct {
	styles   *styles.Styles
	width    int
	height   int
	message  string
	level    ToastLevel
	position ToastPosition
	visible  bool
	duration time.Duration
}

// ClearToast is a message to clear the toast notification.
type ClearToast struct{}

// NewToast creates a new toast component.
func NewToast(s *styles.Styles) *Toast {
	return &Toast{
		styles:   s,
		position: ToastTopLeft,
		duration: 3 * time.Second,
	}
}

// SetSize updates the available screen size.
func (t *Toast) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetPosition sets where the toast appears on screen.
func (t *Toast) SetPosition(pos ToastPosition) {
	t.position = pos
}

// SetDuration sets how long the toast displays before auto-dismissing.
func (t *Toast) SetDuration(d time.Duration) {
	t.duration = d
}

// SetStyles updates the component styles.
func (t *Toast) SetStyles(s *styles.Styles) {
	t.styles = s
}

// Show displays a toast with the given message and level.
// Returns a command that will clear the toast after the duration.
func (t *Toast) Show(message string, level ToastLevel) tea.Cmd {
	t.message = message
	t.level = level
	t.visible = true
	return t.clearAfterDelay()
}

// ShowInfo shows an info-level toast.
func (t *Toast) ShowInfo(message string) tea.Cmd {
	return t.Show(message, ToastInfo)
}

// ShowSuccess shows a success-level toast.
func (t *Toast) ShowSuccess(message string) tea.Cmd {
	return t.Show(message, ToastSuccess)
}

// ShowWarning shows a warning-level toast.
func (t *Toast) ShowWarning(message string) tea.Cmd {
	return t.Show(message, ToastWarning)
}

// ShowError shows an error-level toast.
func (t *Toast) ShowError(message string) tea.Cmd {
	return t.Show(message, ToastError)
}

// Hide hides the toast.
func (t *Toast) Hide() {
	t.visible = false
	t.message = ""
}

// IsVisible returns whether the toast is currently visible.
func (t *Toast) IsVisible() bool {
	return t.visible
}

// Update handles tea messages for the toast.
func (t *Toast) Update(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(ClearToast); ok {
		t.Hide()
	}
	return nil
}

// Overlay renders the toast on top of the base view.
// The toast appears in the configured position without blocking other content.
func (t *Toast) Overlay(baseView string) string {
	if t.styles == nil || !t.visible || t.message == "" || t.width == 0 || t.height == 0 {
		return baseView
	}

	toastBox := t.renderBox()
	toastWidth := lipgloss.Width(toastBox)
	toastHeight := lipgloss.Height(toastBox)

	if toastWidth >= t.width || toastHeight >= t.height {
		return baseView
	}

	baseLines := strings.Split(baseView, "\n")
	toastLines := strings.Split(toastBox, "\n")

	// Ensure we have enough lines
	for len(baseLines) < t.height {
		baseLines = append(baseLines, strings.Repeat(" ", t.width))
	}

	// Calculate position based on ToastPosition
	startRow, startCol := t.calculatePosition(toastWidth, toastHeight)

	// Overlay toast on base view
	for i, toastLine := range toastLines {
		row := startRow + i
		if row < 0 || row >= len(baseLines) {
			continue
		}

		baseLine := baseLines[row]

		// Ensure base line is wide enough (visual width)
		baseLineWidth := ansi.StringWidth(baseLine)
		if baseLineWidth < t.width {
			baseLine = baseLine + strings.Repeat(" ", t.width-baseLineWidth)
		}

		// Build the new line using ANSI-aware functions:
		// [left part][toast line][right part]
		left := ansi.Truncate(baseLine, startCol, "")
		right := ANSICutLeft(baseLine, startCol+toastWidth)

		baseLines[row] = left + toastLine + right
	}

	// Return only up to t.height lines
	if len(baseLines) > t.height {
		baseLines = baseLines[:t.height]
	}

	return strings.Join(baseLines, "\n")
}

func (t *Toast) calculatePosition(toastWidth, toastHeight int) (row, col int) {
	padding := 1 // Padding from screen edges

	switch t.position {
	case ToastTopLeft:
		return padding, padding
	case ToastTopRight:
		return padding, t.width - toastWidth - padding
	case ToastBottomLeft:
		return t.height - toastHeight - padding, padding
	case ToastBottomRight:
		return t.height - toastHeight - padding, t.width - toastWidth - padding
	default:
		return padding, padding
	}
}

func (t *Toast) renderBox() string {
	// Get appropriate style based on level
	var textStyle lipgloss.Style
	var borderColor lipgloss.TerminalColor

	switch t.level {
	case ToastSuccess:
		textStyle = t.styles.DiffAdd
		borderColor = t.styles.AddColor
	case ToastWarning:
		textStyle = t.styles.DiffChange
		borderColor = t.styles.ChangeColor
	case ToastError:
		textStyle = t.styles.DiffRemove
		borderColor = t.styles.RemoveColor
	default: // ToastInfo
		textStyle = t.styles.Highlight
		borderColor = t.styles.Theme.HighlightColor
	}

	content := textStyle.Render(t.message)

	box := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(content)

	return box
}

func (t *Toast) clearAfterDelay() tea.Cmd {
	return tea.Tick(t.duration, func(_ time.Time) tea.Msg {
		return ClearToast{}
	})
}
