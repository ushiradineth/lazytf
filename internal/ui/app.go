package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	tfparser "github.com/ushiradineth/lazytf/internal/terraform/parser"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/views"
	"github.com/ushiradineth/lazytf/internal/utils"
)

// Model is the main application model
type Model struct {
	plan              *terraform.Plan
	resourceList      *components.ResourceList
	diffEngine        *diff.Engine
	diffViewer        *components.DiffViewer
	styles            *styles.Styles
	width             int
	height            int
	diagnosticsHeight int
	ready             bool
	err               error
	quitting          bool
	modalState        ModalState
	showSplit         bool

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
	planRunFlags        []string
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
	streamDone          chan struct{}
	planFilePath        string
	operationState      *terraform.OperationState
	progressCompact     *components.ProgressCompact
	diagnosticsPanel    *components.DiagnosticsPanel
	showDiagnostics     bool
	showCompactProgress bool
	showRawLogs         bool

	historyStore       *history.Store
	historyPanel       *components.HistoryPanel
	historyEntries     []history.Entry
	showHistory        bool
	historyHeight      int
	historySelected    int
	historyFocused     bool
	diagnosticsFocused bool
	historyView        *views.HistoryView
	historyDetail      *history.Entry
	historyLogger      *history.Logger
	lastPlanOutput     string
	config             *config.Config
	configView         *views.ConfigView
	envWorkDir         string
	envCurrent         string
	envStrategy        environment.StrategyType
	envDetection       *environment.DetectionResult
	envOptions         []environment.Environment
	envView            *views.EnvView

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

// ModalState represents which modal overlay is active.
type ModalState int

const (
	ModalNone ModalState = iota
	ModalHelp
	ModalSettings
	ModalEnvSelector
)

type environmentDetector interface {
	Detect(ctx context.Context) (environment.DetectionResult, error)
}

type workspaceManager interface {
	Current(ctx context.Context) (string, error)
	Switch(ctx context.Context, name string) error
}

var newEnvironmentDetector = func(workDir string) (environmentDetector, error) {
	return environment.NewDetector(workDir)
}

var newWorkspaceManager = func(workDir string) (workspaceManager, error) {
	return environment.NewWorkspaceManager(workDir)
}

// ExecutionConfig configures execution mode behavior.
type ExecutionConfig struct {
	Executor       *terraform.Executor
	AutoPlan       bool
	Flags          []string
	UseJSON        bool
	ForceJSON      bool
	WorkDir        string
	EnvName        string
	HistoryStore   *history.Store
	HistoryLogger  *history.Logger
	HistoryEnabled bool
	Config         *config.Config
}

// NewModel creates a new application model
func NewModel(plan *terraform.Plan) *Model {
	return NewModelWithStyles(plan, styles.DefaultStyles())
}

// NewModelWithStyles creates a model with the provided styles.
func NewModelWithStyles(plan *terraform.Plan, appStyles *styles.Styles) *Model {
	if appStyles == nil {
		appStyles = styles.DefaultStyles()
	}
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
		envView:       views.NewEnvView(appStyles),
		configView:    views.NewConfigView(appStyles),
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
	return NewExecutionModelWithStyles(plan, cfg, styles.DefaultStyles())
}

// NewExecutionModelWithStyles creates a model configured for terraform execution with styles.
func NewExecutionModelWithStyles(plan *terraform.Plan, cfg ExecutionConfig, appStyles *styles.Styles) *Model {
	m := NewModelWithStyles(plan, appStyles)
	m.executionMode = true
	m.executor = cfg.Executor
	m.autoPlan = cfg.AutoPlan
	m.planFlags = append([]string{}, cfg.Flags...)
	m.applyFlags = append([]string{}, cfg.Flags...)
	m.envWorkDir = cfg.WorkDir
	m.envCurrent = cfg.EnvName
	m.envStrategy = environment.StrategyUnknown
	m.planView = views.NewPlanView("", m.styles)
	m.applyView = views.NewApplyView(m.styles)
	m.applyView.SetStatusText("Running...", "Apply complete", "Apply failed - press esc to return")
	m.operationState = terraform.NewOperationState()
	m.progressCompact = components.NewProgressCompact(m.operationState, m.styles)
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.diagnosticsHeight = 8
	m.showRawLogs = false
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
	m.envView = views.NewEnvView(m.styles)
	m.config = cfg.Config
	m.configView = views.NewConfigView(m.styles)
	if m.configView != nil {
		m.configView.SetConfig(m.config)
	}
	if cfg.HistoryEnabled {
		store := cfg.HistoryStore
		if store == nil {
			var err error
			store, err = history.OpenDefault()
			if err != nil {
				m.err = err
			}
		}
		m.historyStore = store
		m.historyLogger = cfg.HistoryLogger
		if m.historyLogger == nil && store != nil {
			m.historyLogger = history.NewLogger(store, history.LevelStandard)
		}
		if store != nil {
			if entries, err := m.loadHistoryEntries(); err == nil {
				m.historyEntries = entries
				m.historyPanel.SetEntries(entries)
				m.syncHistorySelection()
			}
		}
	}
	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}
	if m.executionMode {
		if cmd := m.detectEnvironmentsCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
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
		if m.configView != nil {
			m.configView.SetSize(m.width, m.height)
		}
		if m.envView != nil {
			m.envView.SetSize(m.width, m.height)
		}
		if m.progressCompact != nil {
			m.progressCompact.SetSize(m.width, m.height)
		}
		if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetSize(m.width, m.diagnosticsHeight)
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
		cmd := m.streamPlanOutputCmd()
		return m, cmd

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
		cmd := m.streamApplyOutputCmd()
		return m, cmd

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
			cmd := m.clearToastCmd(3 * time.Second)
			return m, cmd
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
			} else {
				parsed := utils.FormatLogOutput(content)
				if strings.TrimSpace(parsed) != "" {
					content = parsed
				}
			}
			m.historyView.SetContent(content)
		}
		return m, nil
	case EnvironmentDetectedMsg:
		if msg.Error != nil {
			m.toastMessage = fmt.Sprintf("Environment detection failed: %v", msg.Error)
			m.toastIsError = true
			cmd := m.clearToastCmd(4 * time.Second)
			return m, cmd
		}
		m.envDetection = &msg.Result
		strategy := msg.Result.Strategy
		current := msg.Current
		if msg.Preference != nil {
			if msg.Preference.Strategy != "" && strategyAvailable(msg.Result, msg.Preference.Strategy) {
				strategy = msg.Preference.Strategy
			}
			if msg.Preference.Environment != "" {
				current = msg.Preference.Environment
			}
		}
		m.envStrategy = strategy
		m.envCurrent = current
		m.setEnvironmentOptions(msg.Result, m.envStrategy, m.envCurrent)
		if m.envCurrent != "" {
			if option, ok := m.findEnvironmentOption(m.envCurrent); ok {
				_ = m.applyEnvironmentSelection(option)
			}
		}
		if m.executionMode && m.envCurrent == "" && m.shouldPromptEnvironment() {
			m.openEnvSelector()
		}
		return m, nil
	case ClearToastMsg:
		m.toastMessage = ""
		m.toastIsError = false
		return m, nil
	case StreamMessageMsg:
		cmd := m.handleStreamMessage(msg.Message)
		return m, cmd
	case OperationStateUpdateMsg:
		if m.diagnosticsPanel != nil && m.operationState != nil {
			m.diagnosticsPanel.SetDiagnostics(m.operationState.GetDiagnostics())
		}
		if m.resourceList != nil && m.resourceList.ShowStatus() {
			m.resourceList.Refresh()
		}
		return m, nil

	case tea.KeyMsg:
		if m.diagnosticsFocused && m.diagnosticsPanel != nil && m.execView == viewMain {
			switch msg.String() {
			case "q", "ctrl+c":
				if m.envView != nil && m.envView.Filtering() {
					return m, cmd
				}
				m.quitting = true
				return m, tea.Quit
			case "esc", "D":
				m.diagnosticsFocused = false
				return m, nil
			case "R":
				m.showRawLogs = !m.showRawLogs
				m.diagnosticsPanel.SetShowRaw(m.showRawLogs)
				return m, nil
			}
			m.diagnosticsPanel, cmd = m.diagnosticsPanel.Update(msg)
			return m, cmd
		}
		if m.modalState == ModalSettings {
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc", ",":
				m.modalState = ModalNone
				return m, nil
			default:
				return m, nil
			}
		}

		if m.modalState == ModalHelp {
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "?", "esc":
				m.modalState = ModalNone
				return m, nil
			default:
				return m, nil
			}
		}

		if m.modalState == ModalEnvSelector {
			if m.envView != nil {
				m.envView, cmd = m.envView.Update(msg)
			}
			if m.envView != nil && m.envView.Filtering() {
				return m, cmd
			}
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc", "e":
				if m.envView != nil && m.envView.Filtering() {
					return m, cmd
				}
				if m.envView != nil && m.envView.Mode() == views.EnvViewEnvironments && m.envDetection != nil && m.envDetection.Strategy == environment.StrategyMixed {
					m.envView.SetMode(views.EnvViewStrategy)
					return m, cmd
				}
				m.modalState = ModalNone
				return m, nil
			case "enter":
				if m.envView == nil {
					return m, nil
				}
				if m.envView.Filtering() {
					return m, cmd
				}
				if m.envView.Mode() == views.EnvViewStrategy {
					selection := m.envView.SelectedStrategy()
					if selection == environment.StrategyUnknown {
						return m, nil
					}
					if m.envDetection == nil {
						return m, nil
					}
					m.envStrategy = selection
					m.setEnvironmentOptions(*m.envDetection, m.envStrategy, m.envCurrent)
					if err := m.saveEnvPreference(m.envStrategy, ""); err != nil {
						m.toastMessage = fmt.Sprintf("Failed to save environment preference: %v", err)
						m.toastIsError = true
						return m, m.clearToastCmd(3 * time.Second)
					}
					m.refreshEnvSelector()
					m.envView.SetMode(views.EnvViewEnvironments)
					return m, nil
				}
				selected := m.envView.SelectedEnvironment()
				if selected == nil {
					return m, nil
				}
				if err := m.applyEnvironmentSelection(*selected); err != nil {
					m.toastMessage = fmt.Sprintf("Failed to switch environment: %v", err)
					m.toastIsError = true
					cmd := m.clearToastCmd(4 * time.Second)
					return m, cmd
				}
				m.envCurrent = envSelectionValue(*selected)
				if entries, err := m.loadHistoryEntries(); err == nil {
					m.historyEntries = entries
					if m.historyPanel != nil {
						m.historyPanel.SetEntries(entries)
						m.syncHistorySelection()
					}
				}
				if err := m.saveEnvPreference(m.envStrategy, m.envCurrent); err != nil {
					m.toastMessage = fmt.Sprintf("Failed to save environment preference: %v", err)
					m.toastIsError = true
					return m, m.clearToastCmd(3 * time.Second)
				}
				m.modalState = ModalNone
				m.toastMessage = "Environment set to " + m.envDisplayName()
				m.toastIsError = false
				cmd := m.clearToastCmd(2 * time.Second)
				return m, cmd
			}
			return m, cmd
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

		if m.inputCaptured() {
			return m, cmd
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
			if m.modalState == ModalHelp {
				m.modalState = ModalNone
			} else {
				m.modalState = ModalHelp
			}
			return m, nil

		case ",":
			if m.modalState == ModalSettings {
				m.modalState = ModalNone
			} else {
				m.modalState = ModalSettings
			}
			return m, nil

		case "e":
			if m.modalState == ModalEnvSelector {
				m.modalState = ModalNone
			} else {
				m.openEnvSelector()
			}
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

	if m.modalState == ModalEnvSelector && m.envView != nil {
		m.envView, cmd = m.envView.Update(msg)
		return m, cmd
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

	if m.modalState == ModalSettings {
		return m.renderSettings()
	}

	if m.modalState == ModalHelp {
		return m.renderHelp()
	}

	if m.modalState == ModalEnvSelector {
		return m.renderEnvSelector()
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

	if m.plan == nil && !m.executionMode {
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

	helpText := "q: quit | ↑↓/jk: navigate | enter/space: toggle group | t: toggle all | c/u/d/r: filter | /: search | e: environments | ,: settings | ?: keybinds"
	if m.executionMode {
		execHelp := "p: plan | a: apply | h: history | tab: focus history | ctrl+c: cancel"
		if m.useJSON {
			execHelp += " | s: status | C: compact | D: focus logs | R: raw logs"
		}
		helpText = execHelp + " | " + helpText
	}

	statusText := fmt.Sprintf("%d resources | %s", totalResources, helpText)
	if m.executionMode {
		statusText = fmt.Sprintf("%d resources | env: %s | %s", totalResources, m.envStatusLabel(), helpText)
	} else {
		statusText = fmt.Sprintf("%d resources | read-only | %s", totalResources, helpText)
	}

	return m.styles.StatusBar.
		Width(m.width).
		Render(statusText)
}

func (m *Model) envStatusLabel() string {
	label := m.envDisplayName()
	if label == "" {
		label = "unknown"
	}
	if m.envStrategy != environment.StrategyUnknown {
		return fmt.Sprintf("%s (%s)", label, m.envStrategy)
	}
	return label
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
		main := lipgloss.JoinHorizontal(lipgloss.Top, leftContent, right)
		return m.appendDiagnostics(main)
	}
	return m.appendDiagnostics(leftContent)
}

func (m *Model) appendDiagnostics(content string) string {
	if !m.executionMode || m.diagnosticsPanel == nil || m.diagnosticsHeight == 0 {
		return content
	}
	panel := m.diagnosticsPanel.View()
	if panel == "" {
		return content
	}
	return lipgloss.JoinVertical(lipgloss.Left, content, panel)
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
				{keys: "e", desc: "select environment"},
				{keys: ",", desc: "open settings"},
				{keys: "?", desc: "toggle keybinds"},
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
				{keys: "C", desc: "toggle compact progress view"},
				{keys: "D", desc: "focus logs panel"},
				{keys: "R", desc: "toggle raw logs"},
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
	lines = append(lines, m.styles.Title.Render("lazytf keybinds"))
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

func (m *Model) renderSettings() string {
	if m.styles == nil {
		return ""
	}
	if m.configView == nil {
		lines := []string{
			m.styles.Highlight.Render("Settings"),
			"",
			"No configuration loaded.",
			"",
			"esc: back",
		}
		content := strings.TrimRight(strings.Join(lines, "\n"), "\n")
		box := m.styles.Border.Width(minInt(50, m.width-4)).Render(content)
		if m.width == 0 || m.height == 0 {
			return box
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	m.configView.SetConfig(m.config)
	return m.configView.View()
}

func (m *Model) detectEnvironmentsCmd() tea.Cmd {
	workDir := m.envWorkDir
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	return func() tea.Msg {
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return EnvironmentDetectedMsg{Error: err}
		}
		pref, err := environment.LoadPreference(absWorkDir)
		if err != nil {
			return EnvironmentDetectedMsg{Error: err}
		}
		detector, err := newEnvironmentDetector(workDir)
		if err != nil {
			return EnvironmentDetectedMsg{Error: err}
		}
		result, err := detector.Detect(context.Background())
		if err != nil {
			return EnvironmentDetectedMsg{Error: err}
		}
		current := m.envCurrent
		if current == "" {
			for _, folder := range result.FolderPaths {
				if folder == absWorkDir {
					current = folder
					break
				}
			}
		}
		if current == "" && len(result.Workspaces) > 0 {
			if manager, err := newWorkspaceManager(workDir); err == nil {
				if name, err := manager.Current(context.Background()); err == nil {
					current = name
				}
			}
		}
		if current == "" && (result.Strategy == environment.StrategyFolder || result.Strategy == environment.StrategyMixed) {
			current = absWorkDir
		}
		return EnvironmentDetectedMsg{Result: result, Current: current, Preference: pref}
	}
}

func (m *Model) setEnvironmentOptions(result environment.DetectionResult, strategy environment.StrategyType, current string) {
	options := make([]environment.Environment, 0, len(result.Environments))
	for _, env := range result.Environments {
		if !strategyMatches(strategy, env.Strategy) {
			continue
		}
		if envMatchesCurrent(env, current) {
			env.IsCurrent = true
		}
		options = append(options, env)
	}
	m.envOptions = options
	m.refreshEnvSelector()
}

func (m *Model) renderEnvSelector() string {
	if m.styles == nil {
		return ""
	}
	if m.envView == nil {
		return ""
	}
	if m.envDetection != nil {
		m.envView.SetWarnings(m.envDetection.Warnings)
	} else {
		m.envView.SetWarnings(nil)
	}
	return m.envView.View()
}

func (m *Model) inputCaptured() bool {
	if m.searching {
		return true
	}
	if m.modalState == ModalEnvSelector && m.envView != nil && m.envView.Filtering() {
		return true
	}
	return false
}

func (m *Model) openEnvSelector() {
	if m.envView == nil || m.envDetection == nil {
		return
	}
	if m.envDetection.Strategy == environment.StrategyMixed {
		m.envView.SetMode(views.EnvViewStrategy)
		m.envView.SetStrategies(buildStrategyOptions(*m.envDetection))
	} else {
		m.envView.SetMode(views.EnvViewEnvironments)
	}
	m.refreshEnvSelector()
	m.modalState = ModalEnvSelector
}

func (m *Model) refreshEnvSelector() {
	if m.envView == nil {
		return
	}
	baseDir := m.envWorkDir
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	m.envView.SetEnvironments(m.envOptions, m.envStrategy, m.envCurrent, baseDir)
}

func (m *Model) shouldPromptEnvironment() bool {
	if m.envDetection == nil {
		return false
	}
	if m.envDetection.Strategy == environment.StrategyMixed {
		return true
	}
	return len(m.envOptions) > 1
}

func (m *Model) findEnvironmentOption(value string) (environment.Environment, bool) {
	for _, option := range m.envOptions {
		if envMatchesCurrent(option, value) {
			return option, true
		}
	}
	return environment.Environment{}, false
}

func (m *Model) saveEnvPreference(strategy environment.StrategyType, current string) error {
	baseDir := m.envWorkDir
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return err
	}
	return environment.SavePreference(absDir, environment.Preference{
		Strategy:    strategy,
		Environment: current,
		UpdatedAt:   time.Now(),
	})
}

func buildStrategyOptions(result environment.DetectionResult) []views.StrategyOption {
	options := []views.StrategyOption{}
	if len(result.Workspaces) > 0 {
		options = append(options, views.StrategyOption{
			Label:    "Terraform workspaces",
			Detail:   fmt.Sprintf("%d workspaces", len(result.Workspaces)),
			Strategy: environment.StrategyWorkspace,
		})
	}
	if len(result.FolderPaths) > 0 {
		options = append(options, views.StrategyOption{
			Label:    "Folder-based environments",
			Detail:   fmt.Sprintf("%d folders", len(result.FolderPaths)),
			Strategy: environment.StrategyFolder,
		})
	}
	if len(result.Workspaces) > 0 && len(result.FolderPaths) > 0 {
	}
	return options
}

func strategyMatches(selected, candidate environment.StrategyType) bool {
	switch selected {
	case environment.StrategyUnknown, environment.StrategyMixed:
		return true
	default:
		return selected == candidate
	}
}

func strategyAvailable(result environment.DetectionResult, strategy environment.StrategyType) bool {
	switch strategy {
	case environment.StrategyWorkspace:
		return len(result.Workspaces) > 0
	case environment.StrategyFolder:
		return len(result.FolderPaths) > 0
	case environment.StrategyMixed:
		return len(result.Workspaces) > 0 && len(result.FolderPaths) > 0
	default:
		return false
	}
}

func envMatchesCurrent(env environment.Environment, current string) bool {
	if current == "" {
		return env.IsCurrent
	}
	if env.Strategy == environment.StrategyWorkspace {
		return env.Name == current
	}
	if env.Path == current {
		return true
	}
	return filepath.Base(env.Path) == current
}

func envSelectionValue(env environment.Environment) string {
	if env.Strategy == environment.StrategyFolder {
		return env.Path
	}
	return env.Name
}

func (m *Model) envDisplayName() string {
	if m.envStrategy == environment.StrategyFolder {
		baseDir := m.envWorkDir
		if strings.TrimSpace(baseDir) == "" {
			baseDir = "."
		}
		if rel, err := filepath.Rel(baseDir, m.envCurrent); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return rel
		}
		if m.envCurrent != "" {
			return filepath.Base(m.envCurrent)
		}
	}
	return m.envCurrent
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
	diagnosticsHeight := 0
	if m.executionMode && m.diagnosticsPanel != nil {
		diagnosticsHeight = m.diagnosticsHeight
		if m.diagnosticsFocused {
			diagnosticsHeight = maxInt(diagnosticsHeight, minInt(m.height/2, 12))
		}
		if diagnosticsHeight >= listHeight {
			diagnosticsHeight = 0
		}
	}
	listHeight -= historyHeight
	listHeight -= diagnosticsHeight
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
		if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetSize(m.width, diagnosticsHeight)
		}
		m.diffViewer.SetSize(diffWidth, listHeight)
	} else {
		m.resourceList.SetSize(m.width, listHeight)
		if m.historyPanel != nil {
			m.historyPanel.SetSize(m.width, historyHeight)
		}
		if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetSize(m.width, diagnosticsHeight)
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
			m.cancelExecution()
			return true, nil
		}
	case viewPlanOutput, viewApplyOutput:
		switch key {
		case "q":
			if !m.planRunning && !m.applyRunning {
				m.execView = viewMain
				return true, nil
			}
			m.quitting = true
			return true, tea.Quit
		case "ctrl+c":
			m.cancelExecution()
			return true, nil
		case "C":
			if m.useJSON {
				m.showCompactProgress = !m.showCompactProgress
				m.showDiagnostics = false
				if !m.showCompactProgress && m.applyView != nil {
					m.applyView.AppendLine("Streaming JSON output enabled. Press 'C' for compact progress.")
				}
				m.updateExecutionViewForStreaming()
				return true, nil
			}
		case "D":
			m.showDiagnostics = !m.showDiagnostics
			m.updateExecutionViewForStreaming()
			return true, nil
		case "esc":
			if !m.planRunning && !m.applyRunning {
				m.execView = viewMain
				return true, nil
			}
		}
	case viewCompactProgress, viewDiagnostics:
		switch key {
		case "q":
			if !m.planRunning && !m.applyRunning {
				m.execView = viewMain
				m.showDiagnostics = false
				return true, nil
			}
			m.quitting = true
			return true, tea.Quit
		case "ctrl+c":
			m.cancelExecution()
			return true, nil
		case "C":
			if m.useJSON {
				m.showCompactProgress = !m.showCompactProgress
				m.showDiagnostics = false
				if !m.showCompactProgress && m.applyView != nil {
					m.applyView.AppendLine("Streaming JSON output enabled. Press 'C' for compact progress.")
				}
				m.updateExecutionViewForStreaming()
				return true, nil
			}
		case "D":
			m.showDiagnostics = !m.showDiagnostics
			m.updateExecutionViewForStreaming()
			return true, nil
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
				m.toastMessage = "No plan loaded; run terraform plan first"
				m.toastIsError = true
				cmd := m.clearToastCmd(3 * time.Second)
				return true, cmd
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
		case "C":
			if m.useJSON {
				m.showCompactProgress = !m.showCompactProgress
				m.showDiagnostics = false
				if !m.showCompactProgress && m.applyView != nil {
					m.applyView.AppendLine("Streaming JSON output enabled. Press 'C' for compact progress.")
				}
				m.updateExecutionViewForStreaming()
				return true, nil
			}
		case "D":
			m.diagnosticsFocused = !m.diagnosticsFocused
			if m.diagnosticsFocused {
				m.showDiagnostics = true
			}
			m.updateExecutionViewForStreaming()
			return true, nil
		case "R":
			m.showRawLogs = !m.showRawLogs
			if m.diagnosticsPanel != nil {
				m.diagnosticsPanel.SetShowRaw(m.showRawLogs)
			}
			return true, nil
		case "tab":
			if m.showHistory && len(m.historyEntries) > 0 {
				m.historyFocused = !m.historyFocused
				m.syncHistorySelection()
				return true, nil
			}
		case "ctrl+c":
			if m.planRunning || m.applyRunning {
				m.cancelExecution()
				return true, nil
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
	planEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}
	planFlags := append([]string{}, m.planFlags...)
	planFilePath := planOutputPath(planFlags)
	if planFilePath == "" {
		workDir := m.envWorkDir
		if m.executor != nil {
			workDir = m.executor.WorkDir()
		}
		if strings.TrimSpace(workDir) == "" {
			workDir = "."
		}
		planFilePath = filepath.Join(workDir, ".lazytf", "tmp", "plan.tfplan")
		planFlags = append(planFlags, "-out="+planFilePath)
	}
	m.planRunFlags = planFlags
	m.planFilePath = planFilePath
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
			m.applyView.AppendLine("Streaming JSON output enabled. Press 'C' for compact progress.")
		}
	}
	m.updateExecutionViewForStreaming()

	return func() tea.Msg {
		result, output, err := m.executor.Plan(ctx, terraform.PlanOptions{
			Flags:   planFlags,
			UseJSON: m.useJSON,
			Env:     planEnv,
		})
		return PlanStartMsg{Result: result, Output: output, Error: err}
	}
}

func (m *Model) applyEnvironmentSelection(option environment.Environment) error {
	if m.planRunning || m.applyRunning {
		return errors.New("cannot change environment while a command is running")
	}
	switch option.Strategy {
	case environment.StrategyWorkspace:
		manager, err := newWorkspaceManager(m.envWorkDir)
		if err != nil {
			return err
		}
		if err := manager.Switch(context.Background(), option.Name); err != nil {
			return err
		}
	case environment.StrategyFolder:
		if m.executor == nil {
			return errors.New("terraform executor not configured")
		}
		exec, err := m.executor.CloneWithWorkDir(option.Path)
		if err != nil {
			return err
		}
		m.executor = exec
	default:
		return fmt.Errorf("unsupported environment strategy: %s", option.Strategy)
	}

	m.setPlan(nil)
	m.planFilePath = ""
	m.planRunFlags = nil
	if m.planView != nil {
		m.planView.SetSummary(m.planSummary())
	}
	if m.operationState != nil {
		m.operationState.InitializeFromPlan(nil)
	}
	return nil
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
	applyEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}
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
			m.applyView.AppendLine("Streaming JSON output enabled. Press 'C' for compact progress.")
		}
	}
	m.updateExecutionViewForStreaming()

	return func() tea.Msg {
		result, output, err := m.executor.Apply(ctx, terraform.ApplyOptions{
			Flags:       m.applyFlags,
			AutoApprove: true,
			UseJSON:     m.useJSON,
			Env:         applyEnv,
		})
		return ApplyStartMsg{Result: result, Output: output, Error: err}
	}
}

func (m *Model) prepareTerraformEnv() ([]string, error) {
	workDir := m.envWorkDir
	if m.executor != nil {
		workDir = m.executor.WorkDir()
	}
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	tmpDir := filepath.Join(workDir, ".lazytf", "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	return []string{"TMPDIR=" + tmpDir}, nil
}

func (m *Model) cancelExecution() {
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}
}

func (m *Model) handlePlanStart(msg PlanStartMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.planRunning = false
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Failed to start terraform plan: %v", msg.Error))
		}
		m.addErrorDiagnostic("Plan failed to start", msg.Error, "")
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
	m.streamDone = nil

	if msg.Error != nil {
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Plan failed: %v", msg.Error))
		}
		m.planFilePath = ""
		m.planRunFlags = nil
		m.addErrorDiagnostic("Plan failed", msg.Error, msg.Output)
		if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetLogText(msg.Output)
			m.diagnosticsPanel.SetParsedText(utils.FormatLogOutput(msg.Output))
		}
		cmd := m.recordOperationCmd("plan", m.planFlagsForRecord(), false, m.planStartedAt, msg.Result, msg.Output, msg.Error)
		return m, cmd
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
		if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetLogText(msg.Output)
			m.diagnosticsPanel.SetParsedText(utils.FormatLogOutput(msg.Output))
		}
	}
	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}
	m.streamParser = nil
	m.showDiagnostics = false
	m.updateExecutionViewForStreaming()
	cmd := m.recordOperationCmd("plan", m.planFlagsForRecord(), false, m.planStartedAt, msg.Result, msg.Output, nil)
	return m, cmd
}

func (m *Model) handleApplyStart(msg ApplyStartMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.applyRunning = false
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Failed to start terraform apply: %v", msg.Error))
		}
		m.addErrorDiagnostic("Apply failed to start", msg.Error, "")
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
	m.streamDone = nil
	if msg.Error != nil || !msg.Success {
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			if msg.Error != nil {
				m.applyView.AppendLine(fmt.Sprintf("Apply failed: %v", msg.Error))
			}
		}
		if msg.Result != nil && m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetLogText(msg.Result.Output)
			m.diagnosticsPanel.SetParsedText(utils.FormatLogOutput(msg.Result.Output))
		}
		if msg.Error != nil {
			output := ""
			if msg.Result != nil {
				output = msg.Result.Output
			}
			m.addErrorDiagnostic("Apply failed", msg.Error, output)
		} else if !msg.Success {
			output := ""
			if msg.Result != nil {
				output = msg.Result.Output
			}
			m.addErrorDiagnostic("Apply failed", errors.New("apply failed"), output)
		}
		status := history.StatusFailed
		if errors.Is(msg.Error, context.Canceled) {
			status = history.StatusCanceled
		}
		m.streamParser = nil
		opErr := msg.Error
		if opErr == nil && !msg.Success {
			opErr = errors.New("apply failed")
		}
		return m, tea.Batch(
			m.recordHistoryCmd(status, m.flattenSummary(m.planSummary()), m.lastPlanOutput, msg.Result, msg.Error),
			m.recordOperationCmd("apply", m.applyFlags, true, m.applyStartedAt, msg.Result, "", opErr),
		)
	}

	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}
	summary := m.planSummary()
	if m.diagnosticsPanel != nil {
		parsed := ""
		if msg.Result != nil {
			parsed = utils.FormatLogOutput(msg.Result.Output)
		}
		if strings.TrimSpace(parsed) == "" {
			parsed = "Apply complete"
		}
		m.diagnosticsPanel.SetParsedText(parsed)
	}
	if m.applyView != nil && msg.Result != nil {
		parsed := utils.FormatLogOutput(msg.Result.Output)
		if strings.TrimSpace(parsed) == "" {
			parsed = strings.TrimSpace(msg.Result.Output)
		}
		m.applyView.SetOutput(parsed)
	}
	m.showDiagnostics = false
	m.execView = viewApplyOutput
	m.setPlan(&terraform.Plan{Resources: nil})
	m.planFilePath = ""
	m.planRunFlags = nil
	m.streamParser = nil
	return m, tea.Batch(
		m.recordHistoryCmd(history.StatusSuccess, m.flattenSummary(summary), m.lastPlanOutput, msg.Result, nil),
		m.recordOperationCmd("apply", m.applyFlags, true, m.applyStartedAt, msg.Result, "", nil),
	)
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

func (m *Model) addErrorDiagnostic(summary string, err error, output string) {
	if err == nil {
		return
	}
	detail := err.Error()
	if strings.TrimSpace(output) != "" {
		detail = detail + "\n\n" + output
	}
	diag := terraform.Diagnostic{
		Severity: "error",
		Summary:  summary,
		Detail:   detail,
	}
	if m.operationState != nil {
		m.operationState.AddDiagnostic(diag)
		if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetDiagnostics(m.operationState.GetDiagnostics())
		}
		return
	}
	if m.diagnosticsPanel != nil {
		m.diagnosticsPanel.SetDiagnostics([]terraform.Diagnostic{diag})
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
	done := make(chan struct{})
	m.streamDone = done

	reader := chanToReader(output)
	go func() {
		if err := parser.Parse(reader, msgChan); err != nil {
			// Best effort parse; stream errors do not block completion.
			_ = err
		}
		close(msgChan)
		close(done)
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
			return PlanCompleteMsg{Error: result.Error, Result: result, Output: result.Output}
		}

		output := result.Output
		if output == "" {
			output = result.Stdout
		}

		if m.executor != nil && m.planFilePath != "" {
			planEnv, err := m.prepareTerraformEnv()
			if err != nil {
				planEnv = nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			showResult, showErr := m.executor.ShowJSON(ctx, m.planFilePath, terraform.ShowOptions{Env: planEnv})
			cancel()
			if showErr == nil && showResult != nil {
				jsonParser := tfparser.NewJSONParser()
				plan, parseErr := jsonParser.Parse(strings.NewReader(showResult.Output))
				if parseErr == nil {
					return PlanCompleteMsg{Plan: plan, Result: result, Output: output}
				}
			}
		}

		if m.useJSON && m.streamParser != nil {
			if m.streamDone != nil {
				select {
				case <-m.streamDone:
				case <-time.After(2 * time.Second):
				}
			}
			if plan := m.streamParser.GetAccumulatedPlan(); plan != nil {
				return PlanCompleteMsg{Plan: plan, Result: result, Output: output}
			}
			if strings.TrimSpace(output) != "" {
				fallback := tfparser.NewStreamParser()
				_ = fallback.Parse(strings.NewReader(output), nil)
				if plan := fallback.GetAccumulatedPlan(); plan != nil {
					return PlanCompleteMsg{Plan: plan, Result: result, Output: output}
				}
			}
		}

		textParser := tfparser.NewTextParser()
		plan, err := textParser.Parse(strings.NewReader(output))
		if err != nil {
			jsonParser := tfparser.NewJSONParser()
			jsonPlan, jsonErr := jsonParser.Parse(strings.NewReader(output))
			if jsonErr != nil {
				return PlanCompleteMsg{Error: fmt.Errorf("parse plan output: %w", err), Result: result, Output: output}
			}
			return PlanCompleteMsg{Plan: jsonPlan, Result: result, Output: output}
		}
		return PlanCompleteMsg{Plan: plan, Result: result, Output: output}
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
		StartedAt:   m.applyStartedAt,
		FinishedAt:  time.Now(),
		Duration:    time.Since(m.applyStartedAt),
		Status:      status,
		Summary:     summary,
		Environment: m.envCurrent,
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
		entries, listErr := m.loadHistoryEntries()
		if listErr != nil {
			return HistoryLoadedMsg{Error: listErr}
		}
		return HistoryLoadedMsg{Entries: entries}
	}
}

func (m *Model) recordOperationCmd(action string, flags []string, autoApprove bool, startedAt time.Time, result *terraform.ExecutionResult, output string, opErr error) tea.Cmd {
	if m.historyLogger == nil {
		return nil
	}
	entry := history.OperationEntry{
		StartedAt:   startedAt,
		Action:      action,
		Command:     m.buildCommand(action, flags, m.useJSON, autoApprove),
		Summary:     m.flattenSummary(m.planSummary()),
		User:        currentUserName(),
		Environment: m.envCurrent,
		Output:      selectOperationOutput(output, result),
	}
	if result != nil {
		entry.ExitCode = result.ExitCode
		entry.Duration = result.Duration
	}
	entry.Status = operationStatus(opErr)
	return func() tea.Msg {
		if err := m.historyLogger.RecordOperation(entry); err != nil {
			return HistoryLoadedMsg{Error: err}
		}
		return nil
	}
}

func (m *Model) loadHistoryEntries() ([]history.Entry, error) {
	if m.historyStore == nil {
		return nil, nil
	}
	if m.envCurrent == "" {
		return m.historyStore.ListRecent(5)
	}
	return m.historyStore.ListRecentForEnvironment(m.envCurrent, 5)
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

func operationStatus(err error) history.Status {
	if err == nil {
		return history.StatusSuccess
	}
	if errors.Is(err, context.Canceled) {
		return history.StatusCanceled
	}
	return history.StatusFailed
}

func selectOperationOutput(output string, result *terraform.ExecutionResult) string {
	if strings.TrimSpace(output) != "" {
		return output
	}
	if result == nil {
		return ""
	}
	if result.Output != "" {
		return result.Output
	}
	if result.Stdout != "" {
		return result.Stdout
	}
	return result.Stderr
}

func (m *Model) buildCommand(action string, flags []string, useJSON bool, autoApprove bool) string {
	args := []string{action}
	args = append(args, flags...)
	if useJSON && !containsFlag(args, "-json") {
		args = append(args, "-json")
	}
	if autoApprove && !containsFlag(args, "-auto-approve") {
		args = append(args, "-auto-approve")
	}
	return "terraform " + strings.Join(args, " ")
}

func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func planOutputPath(flags []string) string {
	for i, flag := range flags {
		if flag == "-out" && i+1 < len(flags) {
			return flags[i+1]
		}
		if strings.HasPrefix(flag, "-out=") {
			value := strings.TrimPrefix(flag, "-out=")
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func (m *Model) planFlagsForRecord() []string {
	if len(m.planRunFlags) > 0 {
		return m.planRunFlags
	}
	return m.planFlags
}

func currentUserName() string {
	if current, err := currentUserFunc(); err == nil && current != nil && current.Username != "" {
		return current.Username
	}
	if value := os.Getenv("USER"); value != "" {
		return value
	}
	return os.Getenv("USERNAME")
}

var currentUserFunc = user.Current

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
			if _, err := io.WriteString(pw, line+"\n"); err != nil {
				if closeErr := pw.CloseWithError(err); closeErr != nil {
					// Best effort close after pipe write failure.
					_ = closeErr
				}
				return
			}
		}
		if err := pw.Close(); err != nil {
			// Best effort close after stream completion.
			_ = err
		}
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
	Keybinds  key.Binding
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
		Keybinds: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle keybinds"),
		),
	}
}
