package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

// StateResourceItem implements ListPanelItem for state resources.
type StateResourceItem struct {
	resource terraform.StateResource
}

// Render renders the state resource item.
func (s StateResourceItem) Render(st *styles.Styles, width int, selected bool) string {
	address := s.resource.Address
	maxWidth := max(10, width)
	if len(address) > maxWidth {
		address = address[:maxWidth-3] + "..."
	}

	// Use consistent selection styling like resource list and history panel
	if selected {
		bg := st.SelectedLineBackground
		text := st.LineItemText.Background(bg).Bold(true).Render(address)
		return PadLineWithBg(text, width, bg)
	}

	text := st.LineItemText.Render(address)
	return PadLine(text, width)
}

// StateListContent renders the terraform state list using ListPanel.
type StateListContent struct {
	listPanel *ListPanel
	styles    *styles.Styles
	resources []terraform.StateResource
	loading   bool
	errorMsg  string

	// Callback for when a resource is selected (enter pressed)
	OnSelect func(address string) tea.Cmd
}

// NewStateListContent creates a new state list content.
func NewStateListContent(s *styles.Styles) *StateListContent {
	if s == nil {
		s = styles.DefaultStyles()
	}
	panel := NewListPanel("[2]", s)
	panel.SetTabs([]string{"State"})
	return &StateListContent{
		listPanel: panel,
		styles:    s,
		loading:   false,
	}
}

// SetSize updates the layout size.
func (s *StateListContent) SetSize(width, height int) {
	s.listPanel.SetSize(width, height)
}

// SetFocused sets the focus state.
func (s *StateListContent) SetFocused(focused bool) {
	s.listPanel.SetFocused(focused)
}

// IsFocused returns whether the panel is focused.
func (s *StateListContent) IsFocused() bool {
	return s.listPanel.IsFocused()
}

// SetStyles updates the component styles.
func (s *StateListContent) SetStyles(st *styles.Styles) {
	s.styles = st
	if s.listPanel != nil {
		s.listPanel.SetStyles(st)
	}
}

// SetResources sets the list of resources.
func (s *StateListContent) SetResources(resources []terraform.StateResource) {
	s.resources = resources
	s.loading = false
	s.errorMsg = ""

	items := make([]ListPanelItem, len(resources))
	for i, res := range resources {
		items[i] = StateResourceItem{resource: res}
	}
	s.listPanel.SetItems(items)
}

// SetLoading sets the loading state.
func (s *StateListContent) SetLoading(loading bool) {
	s.loading = loading
}

// SetError sets an error message.
func (s *StateListContent) SetError(err string) {
	s.errorMsg = err
	s.loading = false
}

// GetSelected returns the currently selected resource.
func (s *StateListContent) GetSelected() *terraform.StateResource {
	idx := s.listPanel.GetSelectedIndex()
	if idx >= 0 && idx < len(s.resources) {
		return &s.resources[idx]
	}
	return nil
}

// MoveUp moves the selection up.
func (s *StateListContent) MoveUp() {
	s.listPanel.MoveUp()
}

// MoveDown moves the selection down.
func (s *StateListContent) MoveDown() {
	s.listPanel.MoveDown()
}

// HandleKey handles key events.
func (s *StateListContent) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		s.MoveUp()
		return true, nil
	case keybinds.KeyDown, "j":
		s.MoveDown()
		return true, nil
	case "enter":
		if res := s.GetSelected(); res != nil && s.OnSelect != nil {
			return true, s.OnSelect(res.Address)
		}
		return true, nil
	}
	return false, nil
}

// View renders the state list content.
// NOTE: This renders just the content lines, NOT a full panel with frame.
// The caller is responsible for wrapping in a PanelFrame if needed.
func (s *StateListContent) View() string {
	if s.styles == nil {
		return ""
	}

	// Handle special states
	if s.loading {
		return s.styles.Dimmed.Render("Loading state...")
	}

	if s.errorMsg != "" {
		return s.styles.DiffRemove.Render("Error: " + s.errorMsg)
	}

	if len(s.resources) == 0 {
		return s.styles.Dimmed.Render("No resources in state. Press 'r' to refresh.")
	}

	// Get content lines from list panel and join them
	lines := s.listPanel.RenderContentLines(s.listPanel.width, s.listPanel.height)
	return strings.Join(lines, "\n")
}

// GetScrollInfo returns scroll information for external scrollbar rendering.
func (s *StateListContent) GetScrollInfo(height int) (scrollPos, thumbSize float64, hasScrollbar bool) {
	return s.listPanel.GetScrollInfo(height)
}

// GetFooterText returns the footer text for the panel.
func (s *StateListContent) GetFooterText() string {
	if len(s.resources) == 0 {
		return ""
	}
	return FormatItemCount(s.listPanel.GetSelectedIndex()+1, len(s.resources))
}

// ResourceCount returns the number of resources.
func (s *StateListContent) ResourceCount() int {
	return len(s.resources)
}

// Clear clears the resources.
func (s *StateListContent) Clear() {
	s.resources = nil
	s.loading = false
	s.errorMsg = ""
	s.listPanel.SetItems(nil)
}
