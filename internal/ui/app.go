package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/diff"
	"github.com/ushiradineth/tftui/internal/history"
	"github.com/ushiradineth/tftui/internal/styles"
	"github.com/ushiradineth/tftui/internal/terraform"
	tfparser "github.com/ushiradineth/tftui/internal/terraform/parser"
	"github.com/ushiradineth/tftui/internal/ui/components"
	"github.com/ushiradineth/tftui/internal/ui/views"
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

	// Execution mode
	executionMode       bool
	executor            *terraform.Executor
	applyView           *views.ApplyView
	planView            *views.PlanView
	autoPlan            bool
	planFlags           []string
	applyFlags          []string
	planRunning         bool
	applyRunning        bool
	outputChan          <-chan string
	cancelFunc          context.CancelFunc
	execView            executionView
	planStartedAt       time.Time
	applyStartedAt      time.Time
	useJSON             bool
	streamMsgChan       <-chan terraform.StreamMessage
	streamParser        *tfparser.StreamParser
	operationState      *terraform.OperationState
	progressCompact     *components.ProgressCompact
	diagnosticsPanel    *components.DiagnosticsPanel
	showDiagnostics     bool
	showCompactProgress bool

	historyStore    *history.Store
	historyPanel    *components.HistoryPanel
	historyEntries  []history.Entry
	showHistory     bool
	historyHeight   int
	historySelected int
	historyFocused  bool
	historyView     *views.HistoryView
	historyDetail   *history.Entry
	lastPlanOutput  string

	toastMessage string
	toastIsError bool
}

type executionView int

const (
	viewMain executionView = iota
	viewPlanOutput
	viewApplyOutput
	viewPlanConfirm
	viewHistoryDetail
	viewCompactProgress
	viewDiagnostics
)

// ExecutionConfig configures execution mode behavior.
type ExecutionConfig struct {
	Executor  *terraform.Executor
	AutoPlan  bool
	Flags     []string
	UseJSON   bool
	ForceJSON bool
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
		execView:      viewMain,
	}

	// Calculate diffs for all resources
	if plan != nil {
		if err := m.diffEngine.CalculateResourceDiffs(plan); err != nil {
			m.err = err
		}
		resourceList.SetResources(plan.Resources)
	}

	return m
}

// NewExecutionModel creates a model configured for terraform execution.
func NewExecutionModel(plan *terraform.Plan, cfg ExecutionConfig) *Model {
	m := NewModel(plan)
	m.executionMode = true
	m.executor = cfg.Executor
	m.autoPlan = cfg.AutoPlan
	m.planFlags = append([]string{}, cfg.Flags...)
	m.applyFlags = append([]string{}, cfg.Flags...)
	m.planView = views.NewPlanView("", m.styles)
	m.applyView = views.NewApplyView(m.styles)
	m.applyView.SetStatusText("Running...", "Apply complete", "Apply failed - press esc to return")
	m.operationState = terraform.NewOperationState()
	m.progressCompact = components.NewProgressCompact(m.operationState, m.styles)
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.resourceList.SetOperationState(m.operationState)
	m.showCompactProgress = cfg.UseJSON
	if plan != nil {
		m.operationState.InitializeFromPlan(plan)
	}

	if cfg.UseJSON && !cfg.ForceJSON && m.executor != nil {
		if supported, err := m.executor.SupportsJSON(); err != nil || !supported {
			m.showCompactProgress = false
			m.useJSON = false
		} else {
			m.useJSON = true
		}
	} else {
		m.useJSON = cfg.UseJSON
	}
	if m.useJSON {
		m.resourceList.SetShowStatus(true)
	}
	m.historyPanel = components.NewHistoryPanel(m.styles)
	m.historyHeight = 6
	m.showHistory = false
	m.historyView = views.NewHistoryView(m.styles)
	store, err := history.OpenDefault()
	if err != nil {
		m.err = err
	} else {
		m.historyStore = store
		if entries, err := store.ListRecent(5); err == nil {
			m.historyEntries = entries
			m.historyPanel.SetEntries(entries)
			m.syncHistorySelection()
		}
	}
	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}
	if m.executionMode && m.autoPlan {
		if cmd := m.beginPlan(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
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
		if m.applyView != nil {
			m.applyView.SetSize(m.width, m.height)
		}
		if m.planView != nil {
			m.planView.SetSize(m.width, m.height)
		}
		if m.historyPanel != nil {
			m.historyPanel.SetSize(m.width, m.historyHeight)
		}
		if m.historyView != nil {
			m.historyView.SetSize(m.width, m.height)
		}
		if m.progressCompact != nil {
			m.progressCompact.SetSize(m.width, m.height)
		}
		if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetSize(m.width, m.height)
		}

	case PlanStartMsg:
		return m.handlePlanStart(msg)

	case PlanOutputMsg:
		if !m.useJSON && m.applyView != nil {
			m.applyView.AppendLine(msg.Line)
		}
		if m.useJSON {
			return m, nil
		}
		return m, m.streamPlanOutputCmd()

	case PlanCompleteMsg:
		return m.handlePlanComplete(msg)

	case ApplyStartMsg:
		return m.handleApplyStart(msg)

	case ApplyOutputMsg:
		if !m.useJSON && m.applyView != nil {
			m.applyView.AppendLine(msg.Line)
		}
		if m.useJSON {
			return m, nil
		}
		return m, m.streamApplyOutputCmd()

	case ApplyCompleteMsg:
		return m.handleApplyComplete(msg)
	case HistoryLoadedMsg:
		if msg.Error != nil {
			m.err = msg.Error
		} else if m.historyPanel != nil {
			m.historyEntries = msg.Entries
			m.historyPanel.SetEntries(msg.Entries)
			m.syncHistorySelection()
		}
		return m, nil
	case HistoryDetailMsg:
		if msg.Error != nil {
			m.toastMessage = fmt.Sprintf("History error: %v", msg.Error)
			m.toastIsError = true
			return m, m.clearToastCmd(3 * time.Second)
		}
		m.historyDetail = &msg.Entry
		if m.historyView != nil {
			title := "Apply details"
			if msg.Entry.WorkDir != "" {
				title = "Apply details - " + msg.Entry.WorkDir
			}
			m.historyView.SetTitle(title)
			content := strings.TrimRight(msg.Entry.Output, "\n")
			if content == "" {
				content = "No stored output for this apply."
			}
			m.historyView.SetContent(content)
		}
		return m, nil
	case ClearToastMsg:
		m.toastMessage = ""
		m.toastIsError = false
		return m, nil
	case StreamMessageMsg:
		return m, m.handleStreamMessage(msg.Message)
	case OperationStateUpdateMsg:
		if m.diagnosticsPanel != nil && m.operationState != nil {
			m.diagnosticsPanel.SetDiagnostics(m.operationState.GetDiagnostics())
		}
		if m.resourceList != nil && m.resourceList.ShowStatus() {
			m.resourceList.Refresh()
		}
		return m, nil

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

		if m.executionMode {
			if handled, cmd := m.handleExecutionKey(msg); handled {
				return m, cmd
			}
		}

		if m.execView != viewMain {
			if m.execView == viewPlanOutput || m.execView == viewApplyOutput {
				m.applyView, cmd = m.applyView.Update(msg)
				return m, cmd
			}
			if m.execView == viewDiagnostics && m.diagnosticsPanel != nil {
				m.diagnosticsPanel, cmd = m.diagnosticsPanel.Update(msg)
				return m, cmd
			}
			if m.execView == viewHistoryDetail && m.historyView != nil {
				m.historyView, cmd = m.historyView.Update(msg)
				return m, cmd
			}
			return m, nil
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

		case "c":
			m.filterCreate = !m.filterCreate
			m.resourceList.SetFilter(terraform.ActionCreate, m.filterCreate)

		case "t":
			m.resourceList.ToggleAllGroups()

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

	if m.executionMode {
		switch m.execView {
		case viewPlanOutput, viewApplyOutput:
			m.applyView, cmd = m.applyView.Update(msg)
			return m, cmd
		case viewDiagnostics:
			if m.diagnosticsPanel != nil {
				m.diagnosticsPanel, cmd = m.diagnosticsPanel.Update(msg)
				return m, cmd
			}
		case viewHistoryDetail:
			if m.historyView != nil {
				m.historyView, cmd = m.historyView.Update(msg)
				return m, cmd
			}
		}
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

	if m.showHelp {
		return m.renderHelp()
	}

	if m.executionMode {
		switch m.execView {
		case viewPlanConfirm:
			if m.planView != nil {
				return m.planView.View()
			}
		case viewPlanOutput, viewApplyOutput:
			if m.applyView != nil {
				return m.applyView.View()
			}
		case viewCompactProgress:
			if m.progressCompact != nil {
				return m.progressCompact.View()
			}
		case viewDiagnostics:
			if m.diagnosticsPanel != nil {
				return m.diagnosticsPanel.View()
			}
		case viewHistoryDetail:
			if m.historyView != nil {
				return m.historyView.View()
			}
		}
	}

	if m.toastMessage != "" {
		return m.renderToast(m.toastMessage, m.toastIsError)
	}

	if m.plan == nil {
		return "No plan loaded\n"
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

	helpText := "q: quit | ↑↓/jk: navigate | enter/space: toggle group | t: toggle all | c/u/d/r: filter | /: search | ?: help"
	if m.executionMode {
		execHelp := "p: plan | a: apply | h: history | tab: focus history | ctrl+c: cancel"
		if m.useJSON {
			execHelp += " | s: status | c: compact | d: diagnostics"
		}
		helpText = execHelp + " | " + helpText
	}

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
	leftContent := m.resourceList.View()
	if m.executionMode && m.showHistory && m.historyPanel != nil && m.historyPanel.View() != "" {
		leftContent = lipgloss.JoinVertical(lipgloss.Left, leftContent, m.historyPanel.View())
	}
	if m.showSplit && m.width >= 100 {
		right := lipgloss.NewStyle().MarginLeft(1).Render(
			m.diffViewer.View(m.resourceList.GetSelectedResource()),
		)
		return lipgloss.JoinHorizontal(lipgloss.Top, leftContent, right)
	}
	return leftContent
}

func (m *Model) renderHelp() string {
	type helpRow struct {
		keys string
		desc string
	}
	type helpSection struct {
		title string
		rows  []helpRow
	}

	sections := []helpSection{
		{
			title: "Navigation",
			rows: []helpRow{
				{keys: "↑/↓ or j/k", desc: "move selection"},
				{keys: "enter/space", desc: "toggle group"},
				{keys: "t", desc: "toggle all groups"},
			},
		},
		{
			title: "Filters",
			rows: []helpRow{
				{keys: "c", desc: "toggle create"},
				{keys: "u", desc: "toggle update"},
				{keys: "d", desc: "toggle delete"},
				{keys: "r", desc: "toggle replace"},
			},
		},
		{
			title: "Search",
			rows: []helpRow{
				{keys: "/", desc: "focus search"},
				{keys: "esc", desc: "clear search"},
			},
		},
		{
			title: "General",
			rows: []helpRow{
				{keys: "?", desc: "toggle help"},
				{keys: "q or ctrl+c", desc: "quit"},
			},
		},
	}
	if m.executionMode {
		sections = append(sections, helpSection{
			title: "Execution",
			rows: []helpRow{
				{keys: "p", desc: "run terraform plan"},
				{keys: "a", desc: "confirm apply"},
				{keys: "h", desc: "toggle history panel"},
				{keys: "tab", desc: "focus history panel"},
				{keys: "ctrl+c", desc: "cancel running command"},
				{keys: "s", desc: "toggle status column"},
				{keys: "c", desc: "toggle compact progress view"},
				{keys: "d", desc: "toggle diagnostics panel"},
			},
		})
	}

	keyWidth := 0
	for _, section := range sections {
		for _, row := range section.rows {
			if len(row.keys) > keyWidth {
				keyWidth = len(row.keys)
			}
		}
	}
	if keyWidth < 8 {
		keyWidth = 8
	}

	var lines []string
	lines = append(lines, m.styles.Title.Render("tftui help"))
	for _, section := range sections {
		lines = append(lines, m.styles.Highlight.Render(section.title))
		for _, row := range section.rows {
			keyText := fmt.Sprintf("%-*s", keyWidth, row.keys)
			left := m.styles.HelpKey.Render(keyText)
			right := m.styles.HelpValue.Render(row.desc)
			lines = append(lines, left+"  "+right)
		}
		lines = append(lines, "")
	}

	content := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	box := m.styles.Border.
		Width(minInt(64, m.width-4)).
		Render(content)

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
	historyHeight := 0
	if m.executionMode && m.showHistory && m.historyPanel != nil {
		historyHeight = m.historyHeight
		if historyHeight >= listHeight {
			historyHeight = 0
		}
	}
	listHeight -= historyHeight
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
		if m.historyPanel != nil {
			m.historyPanel.SetSize(listWidth, historyHeight)
		}
		m.diffViewer.SetSize(diffWidth, listHeight)
	} else {
		m.resourceList.SetSize(m.width, listHeight)
		if m.historyPanel != nil {
			m.historyPanel.SetSize(m.width, historyHeight)
		}
		m.diffViewer.SetSize(m.width, listHeight)
	}
}

func (m *Model) handleExecutionKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	key := msg.String()
	switch m.execView {
	case viewPlanConfirm:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "y", "Y":
			return true, m.beginApply()
		case "n", "N", "esc":
			m.execView = viewMain
			return true, nil
		case "ctrl+c":
			return true, m.cancelExecution()
		}
	case viewPlanOutput, viewApplyOutput:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "ctrl+c":
			return true, m.cancelExecution()
		case "c":
			if m.useJSON && (m.planRunning || m.applyRunning) {
				m.showCompactProgress = !m.showCompactProgress
				m.showDiagnostics = false
				if !m.showCompactProgress && m.applyView != nil {
					m.applyView.AppendLine("Streaming JSON output enabled. Press 'c' for compact progress.")
				}
				m.updateExecutionViewForStreaming()
				return true, nil
			}
		case "d":
			if m.useJSON {
				m.showDiagnostics = !m.showDiagnostics
				m.updateExecutionViewForStreaming()
				return true, nil
			}
		case "esc":
			if !m.planRunning && !m.applyRunning {
				m.execView = viewMain
				return true, nil
			}
		}
	case viewCompactProgress, viewDiagnostics:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "ctrl+c":
			return true, m.cancelExecution()
		case "c":
			if m.useJSON && (m.planRunning || m.applyRunning) {
				m.showCompactProgress = !m.showCompactProgress
				m.showDiagnostics = false
				if !m.showCompactProgress && m.applyView != nil {
					m.applyView.AppendLine("Streaming JSON output enabled. Press 'c' for compact progress.")
				}
				m.updateExecutionViewForStreaming()
				return true, nil
			}
		case "d":
			if m.useJSON {
				m.showDiagnostics = !m.showDiagnostics
				m.updateExecutionViewForStreaming()
				return true, nil
			}
		case "esc":
			if !m.planRunning && !m.applyRunning {
				m.execView = viewMain
				m.showDiagnostics = false
				return true, nil
			}
		}
	case viewHistoryDetail:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "esc":
			m.execView = viewMain
			return true, nil
		}
	default:
		switch key {
		case "p":
			return true, m.beginPlan()
		case "a":
			if m.plan == nil {
				m.err = errors.New("no plan loaded; run terraform plan first")
				return true, nil
			}
			if m.planView != nil {
				m.planView.SetSummary(m.planSummary())
			}
			m.execView = viewPlanConfirm
			return true, nil
		case "h":
			m.showHistory = !m.showHistory
			if !m.showHistory {
				m.historyFocused = false
			}
			m.updateLayout()
			return true, nil
		case "s":
			m.resourceList.SetShowStatus(!m.resourceList.ShowStatus())
			return true, nil
		case "tab":
			if m.showHistory && len(m.historyEntries) > 0 {
				m.historyFocused = !m.historyFocused
				m.syncHistorySelection()
				return true, nil
			}
		case "ctrl+c":
			if m.planRunning || m.applyRunning {
				return true, m.cancelExecution()
			}
		}
		if m.showHistory && m.historyFocused {
			handled, cmd := m.handleHistoryKeys(key)
			if handled {
				return true, cmd
			}
		}
	}
	return false, nil
}

func (m *Model) beginPlan() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.planRunning || m.applyRunning {
		return nil
	}
	m.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	m.planRunning = true
	m.execView = viewPlanOutput
	m.planStartedAt = time.Now()
	m.showDiagnostics = false
	if m.applyView != nil {
		m.applyView.Reset()
		m.applyView.SetTitle("Running terraform plan...")
		m.applyView.SetStatusText("Running...", "Plan complete", "Plan failed - press esc to return")
		m.applyView.SetStatus(views.ApplyRunning)
		if m.useJSON && !m.showCompactProgress {
			m.applyView.AppendLine("Streaming JSON output enabled. Press 'c' for compact progress.")
		}
	}
	m.updateExecutionViewForStreaming()

	return func() tea.Msg {
		result, output, err := m.executor.Plan(ctx, terraform.PlanOptions{Flags: m.planFlags, UseJSON: m.useJSON})
		return PlanStartMsg{Result: result, Output: output, Error: err}
	}
}

func (m *Model) beginApply() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.planRunning || m.applyRunning {
		return nil
	}
	m.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	m.applyRunning = true
	m.execView = viewApplyOutput
	m.applyStartedAt = time.Now()
	m.showDiagnostics = false
	if m.applyView != nil {
		m.applyView.Reset()
		m.applyView.SetTitle("Applying changes...")
		m.applyView.SetStatusText("Running...", "Apply complete", "Apply failed - press esc to return")
		m.applyView.SetStatus(views.ApplyRunning)
		if m.useJSON && !m.showCompactProgress {
			m.applyView.AppendLine("Streaming JSON output enabled. Press 'c' for compact progress.")
		}
	}
	m.updateExecutionViewForStreaming()

	return func() tea.Msg {
		result, output, err := m.executor.Apply(ctx, terraform.ApplyOptions{
			Flags:       m.applyFlags,
			AutoApprove: true,
			UseJSON:     m.useJSON,
		})
		return ApplyStartMsg{Result: result, Output: output, Error: err}
	}
}

func (m *Model) cancelExecution() tea.Cmd {
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}
	return nil
}

func (m *Model) handlePlanStart(msg PlanStartMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.planRunning = false
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Failed to start terraform plan: %v", msg.Error))
		}
		return m, nil
	}

	m.outputChan = msg.Output
	cmds := []tea.Cmd{
		m.waitPlanCompleteCmd(msg.Result),
	}
	if m.useJSON {
		if m.operationState != nil {
			m.operationState.InitializeFromPlan(nil)
		}
		cmds = append(cmds, m.processJSONStream(msg.Output))
	} else {
		cmds = append(cmds, m.streamPlanOutputCmd())
	}
	if m.applyView != nil {
		cmds = append(cmds, m.applyView.Tick())
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handlePlanComplete(msg PlanCompleteMsg) (tea.Model, tea.Cmd) {
	m.planRunning = false
	m.cancelFunc = nil
	m.outputChan = nil
	m.streamMsgChan = nil

	if msg.Error != nil {
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Plan failed: %v", msg.Error))
		}
		return m, nil
	}

	if msg.Plan != nil {
		m.setPlan(msg.Plan)
		if m.operationState != nil {
			m.operationState.InitializeFromPlan(msg.Plan)
		}
		if m.planView != nil {
			m.planView.SetSummary(m.planSummary())
		}
	}
	if msg.Output != "" {
		m.lastPlanOutput = msg.Output
	}
	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}
	m.streamParser = nil
	m.showDiagnostics = false
	m.updateExecutionViewForStreaming()
	return m, nil
}

func (m *Model) handleApplyStart(msg ApplyStartMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.applyRunning = false
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Failed to start terraform apply: %v", msg.Error))
		}
		return m, nil
	}

	m.outputChan = msg.Output
	cmds := []tea.Cmd{
		m.waitApplyCompleteCmd(msg.Result),
	}
	if m.useJSON {
		if m.operationState != nil {
			m.operationState.InitializeFromPlan(m.plan)
		}
		cmds = append(cmds, m.processJSONStream(msg.Output))
	} else {
		cmds = append(cmds, m.streamApplyOutputCmd())
	}
	if m.applyView != nil {
		cmds = append(cmds, m.applyView.Tick())
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleApplyComplete(msg ApplyCompleteMsg) (tea.Model, tea.Cmd) {
	m.applyRunning = false
	m.cancelFunc = nil
	m.outputChan = nil
	m.streamMsgChan = nil
	if msg.Error != nil || !msg.Success {
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			if msg.Error != nil {
				m.applyView.AppendLine(fmt.Sprintf("Apply failed: %v", msg.Error))
			}
		}
		status := history.StatusFailed
		if errors.Is(msg.Error, context.Canceled) {
			status = history.StatusCanceled
		}
		m.streamParser = nil
		return m, m.recordHistoryCmd(status, m.flattenSummary(m.planSummary()), m.lastPlanOutput, msg.Result, msg.Error)
	}

	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}
	summary := m.planSummary()
	m.showDiagnostics = false
	m.updateExecutionViewForStreaming()
	m.setPlan(&terraform.Plan{Resources: nil})
	m.toastMessage = "Apply complete"
	m.toastIsError = false
	m.streamParser = nil
	return m, tea.Batch(m.clearToastCmd(3*time.Second), m.recordHistoryCmd(history.StatusSuccess, m.flattenSummary(summary), m.lastPlanOutput, msg.Result, nil))
}

func (m *Model) handleStreamMessage(msg terraform.StreamMessage) tea.Cmd {
	if m == nil {
		return nil
	}

	updated := false
	if m.operationState != nil {
		switch msg.Type {
		case terraform.MessageTypeApplyStart, terraform.MessageTypeApplyProgress:
			if msg.Hook != nil {
				address := msg.Hook.Address
				if address == "" {
					address = msg.Hook.Resource.Address
				}
				if address != "" {
					m.operationState.StartResource(address, terraform.ParseActionType(msg.Hook.Action))
					updated = true
				}
			}
		case terraform.MessageTypeApplyComplete:
			if msg.Hook != nil {
				address := msg.Hook.Address
				if address == "" {
					address = msg.Hook.Resource.Address
				}
				if address != "" {
					m.operationState.CompleteResource(address, msg.Hook.IDValue)
					updated = true
				}
			}
		case terraform.MessageTypeApplyErrored:
			if msg.Hook != nil {
				address := msg.Hook.Address
				if address == "" {
					address = msg.Hook.Resource.Address
				}
				if address != "" {
					err := errors.New(msg.Hook.Error)
					if msg.Hook.Error == "" {
						err = errors.New("resource apply error")
					}
					m.operationState.ErrorResource(address, err)
					updated = true
				}
			}
		case terraform.MessageTypeDiagnostic:
			if msg.Diagnostic != nil {
				m.operationState.AddDiagnostic(*msg.Diagnostic)
				updated = true
			}
		}
	}

	if updated {
		return tea.Batch(m.streamJSONMessagesCmd(), func() tea.Msg { return OperationStateUpdateMsg{} })
	}
	return m.streamJSONMessagesCmd()
}

func (m *Model) updateExecutionViewForStreaming() {
	if m.execView == viewPlanConfirm || m.execView == viewHistoryDetail {
		return
	}
	if m.showDiagnostics {
		m.execView = viewDiagnostics
		return
	}
	if m.showCompactProgress && (m.planRunning || m.applyRunning) {
		m.execView = viewCompactProgress
		return
	}
	if m.planRunning {
		m.execView = viewPlanOutput
		return
	}
	if m.applyRunning {
		m.execView = viewApplyOutput
		return
	}
	if m.execView != viewMain {
		m.execView = viewMain
	}
}

func (m *Model) streamPlanOutputCmd() tea.Cmd {
	return func() tea.Msg {
		if m.outputChan == nil {
			return nil
		}
		line, ok := <-m.outputChan
		if !ok {
			return nil
		}
		return PlanOutputMsg{Line: line}
	}
}

func (m *Model) streamApplyOutputCmd() tea.Cmd {
	return func() tea.Msg {
		if m.outputChan == nil {
			return nil
		}
		line, ok := <-m.outputChan
		if !ok {
			return nil
		}
		return ApplyOutputMsg{Line: line}
	}
}

func (m *Model) processJSONStream(output <-chan string) tea.Cmd {
	if output == nil {
		return nil
	}
	msgChan := make(chan terraform.StreamMessage, 100)
	parser := tfparser.NewStreamParser()
	m.streamParser = parser
	m.streamMsgChan = msgChan

	reader := chanToReader(output)
	go func() {
		_ = parser.Parse(reader, msgChan)
		close(msgChan)
	}()
	return m.streamJSONMessagesCmd()
}

func (m *Model) streamJSONMessagesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.streamMsgChan == nil {
			return nil
		}
		msg, ok := <-m.streamMsgChan
		if !ok {
			return nil
		}
		return StreamMessageMsg{Message: msg}
	}
}

func (m *Model) waitPlanCompleteCmd(result *terraform.ExecutionResult) tea.Cmd {
	return func() tea.Msg {
		if result == nil {
			return PlanCompleteMsg{Error: errors.New("plan execution result missing")}
		}
		<-result.Done()
		if result.Error != nil {
			return PlanCompleteMsg{Error: result.Error}
		}

		output := result.Output
		if output == "" {
			output = result.Stdout
		}
		if m.useJSON && m.streamParser != nil {
			if plan := m.streamParser.GetAccumulatedPlan(); plan != nil {
				return PlanCompleteMsg{Plan: plan, Output: output}
			}
		}

		textParser := tfparser.NewTextParser()
		plan, err := textParser.Parse(strings.NewReader(output))
		if err != nil {
			jsonParser := tfparser.NewJSONParser()
			jsonPlan, jsonErr := jsonParser.Parse(strings.NewReader(output))
			if jsonErr != nil {
				return PlanCompleteMsg{Error: fmt.Errorf("parse plan output: %w", err), Output: output}
			}
			return PlanCompleteMsg{Plan: jsonPlan, Output: output}
		}
		return PlanCompleteMsg{Plan: plan, Output: output}
	}
}

func (m *Model) waitApplyCompleteCmd(result *terraform.ExecutionResult) tea.Cmd {
	return func() tea.Msg {
		if result == nil {
			return ApplyCompleteMsg{Success: false, Error: errors.New("apply execution result missing")}
		}
		<-result.Done()
		if result.Error != nil {
			return ApplyCompleteMsg{Success: false, Error: result.Error, Result: result}
		}
		return ApplyCompleteMsg{Success: true, Result: result}
	}
}

func (m *Model) setPlan(plan *terraform.Plan) {
	m.plan = plan
	if plan == nil {
		m.resourceList.SetResources(nil)
		return
	}
	if err := m.diffEngine.CalculateResourceDiffs(plan); err != nil {
		m.err = err
	}
	m.resourceList.SetResources(plan.Resources)
	if m.operationState != nil {
		m.operationState.InitializeFromPlan(plan)
	}
}

func (m *Model) recordHistoryCmd(status history.Status, summary string, planOutput string, result *terraform.ExecutionResult, err error) tea.Cmd {
	if m.historyStore == nil {
		return nil
	}
	entry := history.Entry{
		StartedAt:  m.applyStartedAt,
		FinishedAt: time.Now(),
		Duration:   time.Since(m.applyStartedAt),
		Status:     status,
		Summary:    summary,
	}
	if m.executor != nil {
		entry.WorkDir = m.executor.WorkDir()
	}
	if err != nil {
		entry.Error = err.Error()
	}
	if planOutput != "" {
		entry.Output = truncateOutput(planOutput, 2*1024*1024)
	} else if result != nil {
		entry.Output = truncateOutput(result.Output, 2*1024*1024)
	}

	return func() tea.Msg {
		if recordErr := m.historyStore.RecordApply(entry); recordErr != nil {
			return HistoryLoadedMsg{Error: recordErr}
		}
		entries, listErr := m.historyStore.ListRecent(5)
		if listErr != nil {
			return HistoryLoadedMsg{Error: listErr}
		}
		return HistoryLoadedMsg{Entries: entries}
	}
}

func (m *Model) renderToast(message string, isError bool) string {
	if m.styles == nil {
		return ""
	}
	style := m.styles.Highlight
	if isError {
		style = m.styles.Delete
	}
	content := style.Render(message)
	box := m.styles.Border.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *Model) clearToastCmd(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return ClearToastMsg{}
	})
}

func (m *Model) flattenSummary(summary string) string {
	parts := strings.Fields(summary)
	return strings.Join(parts, " ")
}

func (m *Model) syncHistorySelection() {
	if m.historyPanel == nil {
		return
	}
	if len(m.historyEntries) == 0 {
		m.historySelected = 0
		m.historyPanel.SetSelection(0, m.historyFocused)
		return
	}
	if m.historySelected >= len(m.historyEntries) {
		m.historySelected = len(m.historyEntries) - 1
	}
	if m.historySelected < 0 {
		m.historySelected = 0
	}
	m.historyPanel.SetSelection(m.historySelected, m.historyFocused)
}

func (m *Model) handleHistoryKeys(key string) (bool, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.historySelected > 0 {
			m.historySelected--
		}
	case "down", "j":
		if m.historySelected < len(m.historyEntries)-1 {
			m.historySelected++
		}
	case "enter":
		if len(m.historyEntries) == 0 || m.historyStore == nil {
			return true, nil
		}
		entry := m.historyEntries[m.historySelected]
		m.execView = viewHistoryDetail
		if m.historyView != nil {
			m.historyView.SetTitle("Apply details")
			m.historyView.SetContent("Loading...")
		}
		return true, m.loadHistoryDetailCmd(entry.ID)
	default:
		return false, nil
	}
	m.syncHistorySelection()
	return true, nil
}

func (m *Model) loadHistoryDetailCmd(id int64) tea.Cmd {
	if m.historyStore == nil {
		return nil
	}
	return func() tea.Msg {
		entry, err := m.historyStore.GetByID(id)
		if err != nil {
			return HistoryDetailMsg{Error: err}
		}
		return HistoryDetailMsg{Entry: entry}
	}
}

func truncateOutput(output string, maxBytes int) string {
	if maxBytes <= 0 || len(output) <= maxBytes {
		return output
	}
	return output[:maxBytes]
}

func chanToReader(input <-chan string) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		for line := range input {
			_, _ = io.WriteString(pw, line+"\n")
		}
		_ = pw.Close()
	}()
	return pr
}

func (m *Model) planSummary() string {
	if m.plan == nil {
		return "No changes"
	}
	create := m.countResourcesByAction(terraform.ActionCreate)
	update := m.countResourcesByAction(terraform.ActionUpdate)
	deleteCount := m.countResourcesByAction(terraform.ActionDelete)
	replace := m.countResourcesByAction(terraform.ActionReplace)

	lines := []string{
		fmt.Sprintf("+ %d to create", create),
		fmt.Sprintf("~ %d to update", update),
		fmt.Sprintf("- %d to destroy", deleteCount),
	}
	if replace > 0 {
		lines = append(lines, fmt.Sprintf("± %d to replace", replace))
	}
	return strings.Join(lines, "\n")
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
	Up        key.Binding
	Down      key.Binding
	Expand    key.Binding
	ToggleAll key.Binding
	Filter    key.Binding
	Quit      key.Binding
	Help      key.Binding
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
			key.WithHelp("enter/space", "toggle group"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle all groups"),
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
