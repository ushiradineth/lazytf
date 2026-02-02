package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
)

// StateListView renders the terraform state list.
type StateListView struct {
	styles    *styles.Styles
	resources []terraform.StateResource
	selected  int
	width     int
	height    int
	offset    int
}

// NewStateListView creates a new state list view.
func NewStateListView(s *styles.Styles) *StateListView {
	return &StateListView{
		styles: s,
	}
}

// SetStyles updates the component styles.
func (v *StateListView) SetStyles(s *styles.Styles) {
	v.styles = s
}

// SetSize updates the layout size.
func (v *StateListView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetResources sets the list of resources.
func (v *StateListView) SetResources(resources []terraform.StateResource) {
	v.resources = resources
	v.selected = 0
	v.offset = 0
}

// GetSelected returns the currently selected resource.
func (v *StateListView) GetSelected() *terraform.StateResource {
	if len(v.resources) == 0 || v.selected < 0 || v.selected >= len(v.resources) {
		return nil
	}
	return &v.resources[v.selected]
}

// MoveUp moves the selection up.
func (v *StateListView) MoveUp() {
	if v.selected > 0 {
		v.selected--
		if v.selected < v.offset {
			v.offset = v.selected
		}
	}
}

// MoveDown moves the selection down.
func (v *StateListView) MoveDown() {
	if v.selected < len(v.resources)-1 {
		v.selected++
		visibleRows := v.visibleRows()
		if v.selected >= v.offset+visibleRows {
			v.offset = v.selected - visibleRows + 1
		}
	}
}

func (v *StateListView) visibleRows() int {
	// Header (1) + Footer (1) = 2 lines reserved
	rows := v.height - 2
	if rows < 1 {
		rows = 1
	}
	return rows
}

// Update handles input messages.
func (v *StateListView) Update(msg tea.Msg) (*StateListView, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			v.MoveUp()
		case "down", "j":
			v.MoveDown()
		}
	}
	return v, nil
}

// View renders the state list.
func (v *StateListView) View() string {
	if v.styles == nil {
		return ""
	}

	header := v.styles.Title.Width(v.width).Render("Terraform State")

	var lines []string
	visibleRows := v.visibleRows()
	end := v.offset + visibleRows
	if end > len(v.resources) {
		end = len(v.resources)
	}

	for i := v.offset; i < end; i++ {
		res := v.resources[i]
		line := res.Address
		if i == v.selected {
			line = v.styles.Selected.Width(v.width - 2).Render("> " + line)
		} else {
			line = v.styles.ListItem.Width(v.width - 2).Render("  " + line)
		}
		lines = append(lines, line)
	}

	// Pad if needed
	emptyLine := components.GetPadding(v.width)
	for len(lines) < visibleRows {
		lines = append(lines, emptyLine)
	}

	body := strings.Join(lines, "\n")
	footer := v.styles.StatusBar.Width(v.width).Render("↑↓/jk: navigate | enter: show details | esc: back | q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
