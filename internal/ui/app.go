package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
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
	diffViewer   *components.DiffViewer
	styles       *styles.Styles
	width        int
	height       int
	ready        bool
	err          error
	quitting     bool
	showHelp     bool
	showSplit    bool

	searchInput textinput.Model
	searching   bool

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
	diffEngine := diff.NewEngine()
	diffViewer := components.NewDiffViewer(appStyles, diffEngine)
	searchInput := textinput.New()
	searchInput.Placeholder = "press / to search"
	searchInput.Prompt = "Search: "
	searchInput.CharLimit = 80
	searchBg := lipgloss.AdaptiveColor{Light: "#F2F2F2", Dark: "#262626"}
	searchInput.TextStyle = lipgloss.NewStyle().
		Foreground(appStyles.Theme.ForegroundColor).
		Background(searchBg)
	searchInput.PromptStyle = lipgloss.NewStyle().
		Foreground(appStyles.Theme.ForegroundColor).
		Background(searchBg)
	searchInput.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(appStyles.Theme.ForegroundColor).
		Background(searchBg)

	m := &Model{
		plan:          plan,
		resourceList:  resourceList,
		diffEngine:    diffEngine,
		diffViewer:    diffViewer,
		styles:        appStyles,
		ready:         false,
		showSplit:     true,
		searchInput:   searchInput,
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
	return textinput.Blink
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

		m.updateLayout()

	case tea.KeyMsg:
		if m.showHelp {
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "?", "esc":
				m.showHelp = false
				return m, nil
			default:
				return m, nil
			}
		}

		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.resourceList.SetSearchQuery("")
				return m, nil
			case "enter":
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.resourceList.SetSearchQuery(m.searchInput.Value())
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "/":
			m.searching = true
			m.searchInput.Focus()
			return m, nil
		case "esc":
			if m.searchInput.Value() != "" {
				m.searchInput.SetValue("")
				m.resourceList.SetSearchQuery("")
				return m, nil
			}

		case "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "v":
			m.showSplit = !m.showSplit
			m.updateLayout()
			return m, nil

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

	if m.showHelp {
		return m.renderHelp()
	}

	var sections []string

	// Filter bar
	sections = append(sections, m.renderFilterBar())

	// Search bar
	sections = append(sections, m.renderSearchBar())

	// Resource list
	sections = append(sections, m.renderMainContent())

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
	totalResources := 0
	if m.plan != nil {
		totalResources = len(m.plan.Resources)
	}

	helpText := "q: quit | ↑↓/jk: navigate | c/u/d/r: filter | /: search | v: diff | ?: help"

	statusText := fmt.Sprintf("%d resources | %s", totalResources, helpText)

	return m.styles.StatusBar.
		Width(m.width).
		Render(statusText)
}

// countResourcesByAction counts resources of a specific action type
func (m *Model) countResourcesByAction(action terraform.ActionType) int {
	if m.plan == nil {
		return 0
	}
	count := 0
	for _, resource := range m.plan.Resources {
		if resource.Action == action {
			count++
		}
	}
	return count
}

func (m *Model) renderSearchBar() string {
	m.searchInput.Width = maxInt(20, m.width-20)

	if m.searching || m.searchInput.Value() != "" {
		searchView := m.searchInput.View()
		return m.styles.SearchBar.Width(m.width).Render(searchView)
	}

	searchView := m.searchInput.Prompt + m.searchInput.Placeholder
	searchView = m.searchInput.TextStyle.Render(searchView)

	return m.styles.SearchBar.Width(m.width).Render(searchView)
}

func (m *Model) renderMainContent() string {
	if m.showSplit && m.width >= 100 {
		left := m.resourceList.View()
		right := lipgloss.NewStyle().MarginLeft(1).Render(
			m.diffViewer.View(m.resourceList.GetSelectedResource()),
		)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	return m.resourceList.View()
}

func (m *Model) renderHelp() string {
	keys := []string{
		"Navigation: ↑/↓ or j/k",
		"Filters: c/u/d/r",
		"Search: / to focus, esc to clear",
		"Diff panel: v to toggle",
		"Help: ? to close",
		"Quit: q or ctrl+c",
	}

	content := m.styles.Title.Render("tftui help")
	content += "\n"
	for _, line := range keys {
		content += m.styles.HelpValue.Render(line) + "\n"
	}

	box := m.styles.Border.
		Width(minInt(60, m.width-4)).
		Render(strings.TrimRight(content, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *Model) updateLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	reserved := lipgloss.Height(m.renderFilterBar()) +
		lipgloss.Height(m.renderSearchBar()) +
		lipgloss.Height(m.renderStatusBar())
	listHeight := m.height - reserved
	if listHeight < 1 {
		listHeight = 1
	}

	if m.showSplit && m.width >= 100 {
		listWidth := maxInt(40, int(float64(m.width)*0.45))
		diffWidth := m.width - listWidth - 1
		if diffWidth < 20 {
			diffWidth = 20
			listWidth = m.width - diffWidth - 1
		}
		m.resourceList.SetSize(listWidth, listHeight)
		m.diffViewer.SetSize(diffWidth, listHeight)
	} else {
		m.resourceList.SetSize(m.width, listHeight)
		m.diffViewer.SetSize(m.width, listHeight)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// KeyMap defines the key bindings
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Expand key.Binding
	Filter key.Binding
	Quit   key.Binding
	Help   key.Binding
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
