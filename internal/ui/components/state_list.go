package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

// StateListContent renders the terraform state list as a tab content.
type StateListContent struct {
	styles    *styles.Styles
	resources []terraform.StateResource
	selected  int
	width     int
	height    int
	offset    int
	loading   bool
	errorMsg  string

	// Callback for when a resource is selected (enter pressed)
	OnSelect func(address string) tea.Cmd
}

// NewStateListContent creates a new state list content.
func NewStateListContent(s *styles.Styles) *StateListContent {
	return &StateListContent{
		styles:  s,
		loading: false,
	}
}

// SetSize updates the layout size.
func (s *StateListContent) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetResources sets the list of resources.
func (s *StateListContent) SetResources(resources []terraform.StateResource) {
	s.resources = resources
	s.selected = 0
	s.offset = 0
	s.loading = false
	s.errorMsg = ""
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
	if len(s.resources) == 0 || s.selected < 0 || s.selected >= len(s.resources) {
		return nil
	}
	return &s.resources[s.selected]
}

// MoveUp moves the selection up.
func (s *StateListContent) MoveUp() {
	if s.selected > 0 {
		s.selected--
		if s.selected < s.offset {
			s.offset = s.selected
		}
	}
}

// MoveDown moves the selection down.
func (s *StateListContent) MoveDown() {
	if s.selected < len(s.resources)-1 {
		s.selected++
		visibleRows := s.visibleRows()
		if s.selected >= s.offset+visibleRows {
			s.offset = s.selected - visibleRows + 1
		}
	}
}

func (s *StateListContent) visibleRows() int {
	rows := s.height
	if rows < 1 {
		rows = 1
	}
	return rows
}

// HandleKey handles key events.
func (s *StateListContent) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		s.MoveUp()
		return true, nil
	case "down", "j":
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
func (s *StateListContent) View() string {
	if s.styles == nil {
		return ""
	}

	if s.loading {
		return s.styles.Dimmed.Render("Loading state...")
	}

	if s.errorMsg != "" {
		return s.styles.Delete.Render("Error: " + s.errorMsg)
	}

	if len(s.resources) == 0 {
		return s.styles.Dimmed.Render("No resources in state. Press 'r' to refresh.")
	}

	var lines []string
	visibleRows := s.visibleRows()
	end := s.offset + visibleRows
	if end > len(s.resources) {
		end = len(s.resources)
	}

	for i := s.offset; i < end; i++ {
		res := s.resources[i]
		line := res.Address
		maxWidth := s.width - 4
		if maxWidth < 10 {
			maxWidth = 10
		}
		if len(line) > maxWidth {
			line = line[:maxWidth-3] + "..."
		}
		if i == s.selected {
			line = s.styles.Selected.Width(s.width - 2).Render("> " + line)
		} else {
			line = s.styles.ListItem.Width(s.width - 2).Render("  " + line)
		}
		lines = append(lines, line)
	}

	// Pad if needed
	for len(lines) < visibleRows {
		lines = append(lines, strings.Repeat(" ", s.width))
	}

	// Add count info at bottom if we have resources
	if len(s.resources) > 0 {
		countInfo := fmt.Sprintf("%d/%d", s.selected+1, len(s.resources))
		if len(lines) > 0 {
			// Replace last line with count info
			lastIdx := len(lines) - 1
			lines[lastIdx] = lipgloss.NewStyle().
				Foreground(s.styles.Theme.DimmedColor).
				Align(lipgloss.Right).
				Width(s.width).
				Render(countInfo)
		}
	}

	return strings.Join(lines, "\n")
}

// ResourceCount returns the number of resources.
func (s *StateListContent) ResourceCount() int {
	return len(s.resources)
}

// Clear clears the resources.
func (s *StateListContent) Clear() {
	s.resources = nil
	s.selected = 0
	s.offset = 0
	s.loading = false
	s.errorMsg = ""
}
