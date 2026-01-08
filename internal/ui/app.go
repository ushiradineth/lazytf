package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/diff"
	"github.com/ushiradineth/tftui/internal/styles"
	"github.com/ushiradineth/tftui/internal/terraform"
	"github.com/ushiradineth/tftui/internal/ui/components"
)

// Model is the main application model
type Model struct {
	plan         *terraform.Plan
	resourceList *components.ResourceList
	diffEngine   *diff.Engine
	styles       *styles.Styles
	width        int
	height       int
	ready        bool
	err          error
	quitting     bool

	// Filter state
	filterCreate  bool
	filterUpdate  bool
	filterDelete  bool
	filterReplace bool
}

// NewModel creates a new application model
func NewModel(plan *terraform.Plan) *Model {
	appStyles := styles.DefaultStyles()
	resourceList := components.NewResourceList(appStyles)

	m := &Model{
		plan:          plan,
		resourceList:  resourceList,
		diffEngine:    diff.NewEngine(),
		styles:        appStyles,
		ready:         false,
		filterCreate:  true,
		filterUpdate:  true,
		filterDelete:  true,
		filterReplace: true,
	}

	// Calculate diffs for all resources
	if plan != nil {
		m.diffEngine.CalculateResourceDiffs(plan)
		resourceList.SetResources(plan.Resources)
	}

	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.ready = true
		}

		// Update component sizes
		listHeight := m.height - 4 // Reserve space for filter bar and status bar
		m.resourceList.SetSize(m.width, listHeight)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "c":
			m.filterCreate = !m.filterCreate
			m.resourceList.SetFilter(terraform.ActionCreate, m.filterCreate)

		case "u":
			m.filterUpdate = !m.filterUpdate
			m.resourceList.SetFilter(terraform.ActionUpdate, m.filterUpdate)

		case "d":
			m.filterDelete = !m.filterDelete
			m.resourceList.SetFilter(terraform.ActionDelete, m.filterDelete)

		case "r":
			m.filterReplace = !m.filterReplace
			m.resourceList.SetFilter(terraform.ActionReplace, m.filterReplace)
		}

	case ErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	// Update resource list
	m.resourceList, cmd = m.resourceList.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the application
func (m *Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if !m.ready {
		return "Loading..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.plan == nil {
		return "No plan loaded\n"
	}

	var sections []string

	// Filter bar
	sections = append(sections, m.renderFilterBar())

	// Resource list
	sections = append(sections, m.resourceList.View())

	// Status bar
	sections = append(sections, m.renderStatusBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderFilterBar renders the action filter bar
func (m *Model) renderFilterBar() string {
	var filters []string

	createCount := m.countResourcesByAction(terraform.ActionCreate)
	updateCount := m.countResourcesByAction(terraform.ActionUpdate)
	deleteCount := m.countResourcesByAction(terraform.ActionDelete)
	replaceCount := m.countResourcesByAction(terraform.ActionReplace)

	// Create filter
	createLabel := fmt.Sprintf("Create (%d)", createCount)
	if m.filterCreate {
		filters = append(filters, m.styles.FilterBarActive.Render("[✓] "+createLabel))
	} else {
		filters = append(filters, m.styles.FilterBarInactive.Render("[ ] "+createLabel))
	}

	// Update filter
	updateLabel := fmt.Sprintf("Update (%d)", updateCount)
	if m.filterUpdate {
		filters = append(filters, m.styles.FilterBarActive.Render("[✓] "+updateLabel))
	} else {
		filters = append(filters, m.styles.FilterBarInactive.Render("[ ] "+updateLabel))
	}

	// Delete filter
	deleteLabel := fmt.Sprintf("Delete (%d)", deleteCount)
	if m.filterDelete {
		filters = append(filters, m.styles.FilterBarActive.Render("[✓] "+deleteLabel))
	} else {
		filters = append(filters, m.styles.FilterBarInactive.Render("[ ] "+deleteLabel))
	}

	// Replace filter
	replaceLabel := fmt.Sprintf("Replace (%d)", replaceCount)
	if m.filterReplace {
		filters = append(filters, m.styles.FilterBarActive.Render("[✓] "+replaceLabel))
	} else {
		filters = append(filters, m.styles.FilterBarInactive.Render("[ ] "+replaceLabel))
	}

	filterBar := lipgloss.JoinHorizontal(lipgloss.Left, filters...)

	// Add border
	return m.styles.Border.
		BorderBottom(true).
		Width(m.width - 2).
		Render(filterBar)
}

// renderStatusBar renders the bottom status bar
func (m *Model) renderStatusBar() string {
	totalResources := len(m.plan.Resources)

	helpText := "q: quit | ↑↓/jk: navigate | enter/space: expand | c/u/d/r: filter"

	statusText := fmt.Sprintf("%d resources | %s", totalResources, helpText)

	return m.styles.StatusBar.
		Width(m.width).
		Render(statusText)
}

// countResourcesByAction counts resources of a specific action type
func (m *Model) countResourcesByAction(action terraform.ActionType) int {
	count := 0
	for _, resource := range m.plan.Resources {
		if resource.Action == action {
			count++
		}
	}
	return count
}

// KeyMap defines the key bindings
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Expand   key.Binding
	Filter   key.Binding
	Quit     key.Binding
	Help     key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Expand: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "expand/collapse"),
		),
		Filter: key.NewBinding(
			key.WithKeys("c", "u", "d", "r"),
			key.WithHelp("c/u/d/r", "toggle filters"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}
