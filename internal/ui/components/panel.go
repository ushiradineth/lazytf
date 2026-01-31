package components

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// Panel is the base interface that all panels implement.
// It provides the core rendering and sizing capabilities.
type Panel interface {
	// View renders the panel content.
	View() string

	// SetSize updates the panel dimensions.
	SetSize(width, height int)

	// SetFocused sets the focus state of the panel.
	SetFocused(focused bool)

	// IsFocused returns whether the panel is focused.
	IsFocused() bool

	// SetStyles updates the component styles.
	SetStyles(s *styles.Styles)
}

// InteractivePanel extends Panel with keyboard handling capability.
type InteractivePanel interface {
	Panel

	// HandleKey handles key events and returns whether the event was handled
	// and any command to execute.
	HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)
}

// SelectablePanel extends Panel with item selection capability.
type SelectablePanel interface {
	Panel

	// GetSelectedIndex returns the currently selected item index.
	GetSelectedIndex() int

	// SetSelectedIndex sets the selected item index.
	SetSelectedIndex(index int)

	// ItemCount returns the total number of items in the panel.
	ItemCount() int
}

// FullPanel combines all panel interfaces for components that support
// both keyboard handling and item selection.
type FullPanel interface {
	InteractivePanel
	SelectablePanel
}
