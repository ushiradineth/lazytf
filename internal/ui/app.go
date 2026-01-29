package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
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

	// Filter state
	filterCreate  bool
	filterUpdate  bool
	filterDelete  bool
	filterReplace bool

	// Execution mode
	executionMode    bool
	executor         *terraform.Executor
	applyView        *views.ApplyView
	planView         *views.PlanView
	autoPlan         bool
	planFlags        []string
	applyFlags       []string
	planRunFlags     []string
	planRunning      bool
	applyRunning     bool
	refreshRunning   bool
	outputChan       <-chan string
	cancelFunc       context.CancelFunc
	execView         executionView
	planStartedAt    time.Time
	applyStartedAt   time.Time
	refreshStartedAt time.Time
	planFilePath     string
	operationState   *terraform.OperationState
	diagnosticsPanel *components.DiagnosticsPanel

	historyStore       *history.Store
	historyPanel       *components.HistoryPanel
	historyEntries     []history.Entry
	showHistory        bool
	historyHeight      int
	historySelected    int
	historyFocused     bool
	diagnosticsFocused bool
	historyDetail      *history.Entry
	historyLogger      *history.Logger
	stateListView      *views.StateListView
	stateShowView      *views.StateShowView
	stateResources     []terraform.StateResource
	stateListContent   *components.StateListContent
	resourcesActiveTab int // 0 = Resources, 1 = State
	lastPlanOutput     string
	config             *config.Config
	configView         *views.ConfigView
	envWorkDir         string
	envCurrent         string
	envStrategy        environment.StrategyType
	envDetection       *environment.DetectionResult
	envOptions         []environment.Environment

	// Overlay components
	toast     *components.Toast
	helpModal *components.Modal

	// Panel system
	panelManager     *PanelManager
	environmentPanel *components.EnvironmentPanel
	mainArea         *MainArea
	commandLogPanel  *components.CommandLogPanel
}

type executionView int

const (
	viewMain executionView = iota
	viewPlanOutput
	viewApplyOutput
	viewPlanConfirm
	viewHistoryDetail
	viewDiagnostics
	viewCommandLog
	viewStateList
	viewStateShow
)

// ModalState represents which modal overlay is active.
type ModalState int

const (
	ModalNone ModalState = iota
	ModalHelp
	ModalSettings
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

	// Initialize panel system
	panelManager := NewPanelManager()
	environmentPanel := components.NewEnvironmentPanel(appStyles)
	mainArea := NewMainArea(appStyles, diffEngine, nil, nil)
	commandLogPanel := components.NewCommandLogPanel(appStyles)

	// Initialize overlay components
	toast := components.NewToast(appStyles)
	toast.SetPosition(components.ToastTopLeft)
	helpModal := components.NewModal(appStyles)

	m := &Model{
		plan:          plan,
		resourceList:  resourceList,
		diffEngine:    diffEngine,
		diffViewer:    diffViewer,
		styles:        appStyles,
		ready:         false,
		showSplit:     true,
		filterCreate:  true,
		filterUpdate:  true,
		filterDelete:  true,
		filterReplace: true,
		execView:      viewMain,
		configView:    views.NewConfigView(appStyles),

		// Overlay components
		toast:     toast,
		helpModal: helpModal,

		// Panel system
		panelManager:     panelManager,
		environmentPanel: environmentPanel,
		mainArea:         mainArea,
		commandLogPanel:  commandLogPanel,
	}

	// Register panels with manager
	panelManager.RegisterPanel(PanelWorkspace, environmentPanel)
	panelManager.RegisterPanel(PanelResources, resourceList)
	panelManager.RegisterPanel(PanelHistory, nil) // Will be set later
	panelManager.RegisterPanel(PanelMain, mainArea)
	panelManager.RegisterPanel(PanelCommandLog, commandLogPanel)

	// Calculate diffs for all resources
	if plan != nil {
		if err := m.diffEngine.CalculateResourceDiffs(plan); err != nil {
			m.err = err
		}
		resourceList.SetResources(plan.Resources)
	}

	// Initialize focus on the default panel (Resources)
	panelManager.SetFocus(PanelResources)

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
	if m.panelManager != nil {
		m.panelManager.SetExecutionMode(true)
	}
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
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.diagnosticsHeight = 8
	m.resourceList.SetOperationState(m.operationState)
	if plan != nil {
		m.operationState.InitializeFromPlan(plan)
	}

	// Update main area with plan/apply views
	m.mainArea = NewMainArea(m.styles, m.diffEngine, m.applyView, m.planView)
	m.panelManager.RegisterPanel(PanelMain, m.mainArea)

	// Initialize state list content for the Resources panel tab
	m.stateListContent = components.NewStateListContent(m.styles)
	m.stateListContent.OnSelect = func(address string) tea.Cmd {
		return m.beginStateShow(address)
	}

	m.historyPanel = components.NewHistoryPanel(m.styles)
	m.historyHeight = 6
	m.showHistory = false
	// Register history panel with panel manager
	m.panelManager.RegisterPanel(PanelHistory, m.historyPanel)
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
	var cmds []tea.Cmd
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
		// Note: historyPanel size is now set via panelManager.UpdatePanelSizes() in updateLayout()
		// Note: historyView is now embedded in mainArea, sized via mainArea.SetSize()
		if m.configView != nil {
			m.configView.SetSize(m.width, m.height)
		}
		// Note: diagnosticsPanel is now wrapped in commandLogPanel, sized via panel manager

	case PlanStartMsg:
		return m.handlePlanStart(msg)

	case PlanOutputMsg:
		if m.applyView != nil {
			m.applyView.AppendLine(msg.Line)
		}
		cmd := m.streamPlanOutputCmd()
		return m, cmd

	case PlanCompleteMsg:
		return m.handlePlanComplete(msg)

	case ApplyStartMsg:
		return m.handleApplyStart(msg)

	case ApplyOutputMsg:
		if m.applyView != nil {
			m.applyView.AppendLine(msg.Line)
		}
		cmd := m.streamApplyOutputCmd()
		return m, cmd

	case ApplyCompleteMsg:
		return m.handleApplyComplete(msg)

	case RefreshStartMsg:
		return m.handleRefreshStart(msg)

	case RefreshOutputMsg:
		if m.applyView != nil {
			m.applyView.AppendLine(msg.Line)
		}
		cmd := m.streamRefreshOutputCmd()
		return m, cmd

	case RefreshCompleteMsg:
		return m.handleRefreshComplete(msg)

	case ValidateCompleteMsg:
		return m.handleValidateComplete(msg)

	case FormatCompleteMsg:
		return m.handleFormatComplete(msg)

	case StateListCompleteMsg:
		return m.handleStateListComplete(msg)

	case StateShowCompleteMsg:
		return m.handleStateShowComplete(msg)
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
			var cmd tea.Cmd
			if m.toast != nil {
				cmd = m.toast.ShowError(fmt.Sprintf("History error: %v", msg.Error))
			}
			return m, cmd
		}
		m.historyDetail = &msg.Entry
		// Update history content in main area
		if m.mainArea != nil {
			title := "Apply details"
			if msg.Entry.WorkDir != "" {
				title = "Apply details - " + msg.Entry.WorkDir
			}
			content := strings.TrimRight(msg.Entry.Output, "\n")
			if content == "" {
				content = "No stored output for this apply."
			} else {
				parsed := utils.FormatLogOutput(content)
				if strings.TrimSpace(parsed) != "" {
					content = parsed
				}
			}
			m.mainArea.SetHistoryContent(title, content)
		}
		return m, nil
	case EnvironmentDetectedMsg:
		if msg.Error != nil {
			var cmd tea.Cmd
			if m.toast != nil {
				cmd = m.toast.ShowError(fmt.Sprintf("Environment detection failed: %v", msg.Error))
			}
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

		// Update environment panel with detection results
		if m.environmentPanel != nil {
			m.environmentPanel.SetEnvironmentInfo(m.envCurrent, m.envWorkDir, m.envStrategy, m.envOptions)
			m.environmentPanel.SetWarnings(msg.Result.Warnings)
		}

		// Load filter preferences for this workspace
		m.loadFilterPreferences()

		if m.envCurrent != "" {
			if option, ok := m.findEnvironmentOption(m.envCurrent); ok {
				_ = m.applyEnvironmentSelection(option)
			}
		}
		if m.executionMode && m.envCurrent == "" && m.shouldPromptEnvironment() {
			if m.panelManager != nil && m.environmentPanel != nil {
				cmds = append(cmds, m.panelManager.SetFocus(PanelWorkspace))
				m.environmentPanel.ActivateSelector()
				m.updateLayout()
				return m, tea.Batch(cmds...)
			}
		}
		return m, nil
	case ClearToastMsg, components.ClearToast:
		if m.toast != nil {
			m.toast.Hide()
		}
		return m, nil

	case components.EnvironmentChangedMsg:
		// Apply environment change from environment panel
		if err := m.applyEnvironmentSelection(msg.Environment); err != nil {
			var cmd tea.Cmd
			if m.toast != nil {
				cmd = m.toast.ShowError(fmt.Sprintf("Failed to switch environment: %v", err))
			}
			return m, cmd
		}
		m.envCurrent = envSelectionValue(msg.Environment)

		// Update environment panel to reflect the change
		if m.environmentPanel != nil {
			m.environmentPanel.SetEnvironmentInfo(m.envCurrent, m.envWorkDir, m.envStrategy, m.envOptions)
		}

		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowSuccess("Environment changed to " + m.envDisplayName())
		}
		return m, cmd

	case tea.KeyMsg:
		if m.diagnosticsFocused && m.diagnosticsPanel != nil && m.execView == viewMain {
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc", "D":
				m.diagnosticsFocused = false
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
			case "j", "down":
				if m.helpModal != nil {
					m.helpModal.ScrollDown()
				}
				return m, nil
			case "k", "up":
				if m.helpModal != nil {
					m.helpModal.ScrollUp()
				}
				return m, nil
			default:
				return m, nil
			}
		}

		if m.panelManager != nil && m.environmentPanel != nil && m.environmentPanel.SelectorActive() {
			if handled, panelCmd := m.environmentPanel.HandleKey(msg); handled {
				return m, panelCmd
			}
			return m, nil
		}

		if m.inputCaptured() {
			return m, cmd
		}

		// Handle panel navigation if panel manager is active
		if m.panelManager != nil && m.execView == viewMain {
			if handled, navCmd := m.panelManager.HandleNavigation(msg); handled {
				cmds = append(cmds, navCmd)
				// Update layout when focus changes (affects panel heights)
				m.updateLayout()
				// Sync historyFocused state with panel manager
				m.historyFocused = m.panelManager.GetFocusedPanel() == PanelHistory
				return m, tea.Batch(cmds...)
			}

			// Special handling for Enter key on command log panel
			focusedPanel := m.panelManager.GetFocusedPanel()
			if focusedPanel == PanelCommandLog && msg.String() == "enter" {
				m.execView = viewCommandLog
				return m, nil
			}

			// Route keys to focused panel
			if panel, ok := m.panelManager.GetPanel(focusedPanel); ok {
				if handled, panelCmd := panel.HandleKey(msg); handled {
					cmds = append(cmds, panelCmd)
					return m, tea.Batch(cmds...)
				}
			}
		}

		if m.executionMode {
			if handled, cmd := m.handleExecutionKey(msg); handled {
				return m, cmd
			}
		}

		if m.execView != viewMain {
			if m.execView == viewCommandLog && m.commandLogPanel != nil {
				// Forward scroll keys to diagnostics panel
				if diagPanel := m.commandLogPanel.GetDiagnosticsPanel(); diagPanel != nil {
					_, cmd = diagPanel.Update(msg)
					return m, cmd
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "esc":
		case "?":
			if m.modalState == ModalHelp {
				m.modalState = ModalNone
			} else {
				m.modalState = ModalHelp
				m.updateHelpModalContent()
			}
			return m, nil

		case ",":
			if m.modalState == ModalSettings {
				m.modalState = ModalNone
			} else {
				m.modalState = ModalSettings
			}
			return m, nil

		case "c":
			m.filterCreate = !m.filterCreate
			m.resourceList.SetFilter(terraform.ActionCreate, m.filterCreate)
			m.saveFilterPreferences()

		case "t":
			m.resourceList.ToggleAllGroups()

		case "u":
			m.filterUpdate = !m.filterUpdate
			m.resourceList.SetFilter(terraform.ActionUpdate, m.filterUpdate)
			m.saveFilterPreferences()

		case "d":
			m.filterDelete = !m.filterDelete
			m.resourceList.SetFilter(terraform.ActionDelete, m.filterDelete)
			m.saveFilterPreferences()

		case "r":
			m.filterReplace = !m.filterReplace
			m.resourceList.SetFilter(terraform.ActionReplace, m.filterReplace)
			m.saveFilterPreferences()

		case "[":
			// Switch to previous tab in resources panel
			if m.executionMode && m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources {
				if m.resourcesActiveTab > 0 {
					m.resourcesActiveTab--
				} else {
					m.resourcesActiveTab = 1 // Wrap to State tab
				}
				// If switching to State tab and no state loaded, load it
				if m.resourcesActiveTab == 1 && m.stateListContent != nil && m.stateListContent.ResourceCount() == 0 {
					m.stateListContent.SetLoading(true)
					return m, m.beginStateList()
				}
				return m, nil
			}

		case "]":
			// Switch to next tab in resources panel
			if m.executionMode && m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources {
				if m.resourcesActiveTab < 1 {
					m.resourcesActiveTab++
				} else {
					m.resourcesActiveTab = 0 // Wrap to Resources tab
				}
				// If switching to State tab and no state loaded, load it
				if m.resourcesActiveTab == 1 && m.stateListContent != nil && m.stateListContent.ResourceCount() == 0 {
					m.stateListContent.SetLoading(true)
					return m, m.beginStateList()
				}
				return m, nil
			}
		}

	case ErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	// Handle keys for State tab when active
	if m.executionMode && m.resourcesActiveTab == 1 && m.stateListContent != nil {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources {
				handled, stateCmd := m.stateListContent.HandleKey(keyMsg)
				if handled {
					cmds = append(cmds, stateCmd)
					return m, tea.Batch(cmds...)
				}
			}
		}
	}

	// Update resource list (for Resources tab)
	updated, cmd := m.resourceList.Update(msg)
	if rl, ok := updated.(*components.ResourceList); ok {
		m.resourceList = rl
	}
	cmds = append(cmds, cmd)

	// Update environment panel for async selector messages
	if m.environmentPanel != nil {
		updatedPanel, panelCmd := m.environmentPanel.Update(msg)
		if panel, ok := updatedPanel.(*components.EnvironmentPanel); ok {
			m.environmentPanel = panel
		}
		cmds = append(cmds, panelCmd)
	}

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

	if m.executionMode {
		switch m.execView {
		case viewPlanConfirm:
			if m.planView != nil {
				return m.planView.View()
			}
		case viewCommandLog:
			return m.renderFullScreenCommandLog()
			// Note: viewDiagnostics removed - diagnostics now show in command log panel
		case viewStateList:
			if m.stateListView != nil {
				return m.stateListView.View()
			}
		case viewStateShow:
			if m.stateShowView != nil {
				return m.stateShowView.View()
			}
		}
	}

	if m.plan == nil && !m.executionMode {
		return "No plan loaded\n"
	}

	var sections []string

	// Main content (panels + command log)
	sections = append(sections, m.renderMainContent())

	// Status bar
	sections = append(sections, m.renderStatusBar())

	view := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Overlay help modal if showing
	if m.modalState == ModalHelp && m.helpModal != nil {
		view = m.helpModal.Overlay(view)
	}

	// Overlay toast if present
	if m.toast != nil && m.toast.IsVisible() {
		view = m.toast.Overlay(view)
	}

	return view
}

// renderStatusBar renders the bottom status bar
func (m *Model) renderStatusBar() string {
	var parts []string

	// Add workspace/environment info if available
	if m.executionMode && m.envCurrent != "" {
		parts = append(parts, m.styles.Highlight.Render(m.envDisplayName()))
	}

	// Add resource summary
	if m.plan != nil && len(m.plan.Resources) > 0 {
		parts = append(parts, m.resourceSummaryText())
	}

	// Add help text
	parts = append(parts, m.statusHelpText())

	statusText := strings.Join(parts, " │ ")

	// Limit status bar to 1 line to prevent scrolling
	return m.styles.StatusBar.
		Width(m.width).
		MaxHeight(1).
		Render(statusText)
}

// resourceSummaryText returns a summary of resource changes like "+5 ~3 -2 ±2"
func (m *Model) resourceSummaryText() string {
	if m.plan == nil {
		return ""
	}
	create := m.countResourcesByAction(terraform.ActionCreate)
	update := m.countResourcesByAction(terraform.ActionUpdate)
	deleteCount := m.countResourcesByAction(terraform.ActionDelete)
	replace := m.countResourcesByAction(terraform.ActionReplace)

	total := create + update + deleteCount + replace
	if total == 0 {
		return "no changes"
	}

	var parts []string
	if create > 0 {
		parts = append(parts, m.styles.Create.Render(fmt.Sprintf("+%d", create)))
	}
	if update > 0 {
		parts = append(parts, m.styles.Update.Render(fmt.Sprintf("~%d", update)))
	}
	if deleteCount > 0 {
		parts = append(parts, m.styles.Delete.Render(fmt.Sprintf("-%d", deleteCount)))
	}
	if replace > 0 {
		parts = append(parts, m.styles.Replace.Render(fmt.Sprintf("±%d", replace)))
	}

	return fmt.Sprintf("%d changes (%s)", total, strings.Join(parts, " "))
}

func (m *Model) statusHelpText() string {
	base := "q: quit | ,: settings | ?: keybinds"
	if m.panelManager == nil {
		return base
	}
	if m.environmentPanel != nil && m.environmentPanel.SelectorActive() {
		return "type: filter | enter: select | esc: back | " + base
	}
	switch m.panelManager.GetFocusedPanel() {
	case PanelWorkspace:
		return "e: select environment | " + base
	case PanelResources:
		parts := []string{}
		if m.executionMode {
			parts = append(parts, "p: plan", "a: apply", "ctrl+c: cancel")
		}
		parts = append(parts, "enter/space: toggle", "t: toggle all", "c/u/d/r: filter")
		parts = append(parts, base)
		return strings.Join(parts, " | ")
	case PanelHistory:
		return "↑↓/jk: select | enter: view | " + base
	default:
		return base
	}
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

// countResourcesByAction counts resources of a specific action type.
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

func (m *Model) renderMainContent() string {
	if m.panelManager == nil {
		return m.renderLegacyMainContent()
	}

	// Calculate layout to get the widths
	layout := m.panelManager.CalculateLayout(m.width, m.height)

	// Update main area with selected resource (layout is already calculated in updateLayout())
	if m.mainArea != nil && m.resourceList != nil {
		m.mainArea.SetSelectedResource(m.resourceList.GetSelectedResource())
	}

	// Calculate total height for the content area (excluding status bar)
	contentHeight := m.height - StatusBarHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Helper to enforce dimensions on panel output
	enforceDimensions := func(view string, width, height int) string {
		if width <= 0 || height <= 0 {
			return ""
		}
		lines := strings.Split(view, "\n")
		// Enforce width on each line
		for i, line := range lines {
			lineWidth := lipgloss.Width(line)
			if lineWidth < width {
				lines[i] = line + strings.Repeat(" ", width-lineWidth)
			} else if lineWidth > width {
				lines[i] = lipgloss.NewStyle().MaxWidth(width).Render(line)
			}
		}
		// Enforce height: truncate or pad
		if len(lines) > height {
			lines = lines[:height]
		}
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		return strings.Join(lines, "\n")
	}

	// Render left column panels with explicit dimensions
	var leftPanels []string
	if m.environmentPanel != nil {
		leftPanels = append(leftPanels, enforceDimensions(m.environmentPanel.View(), layout.LeftColumnWidth, layout.Workspace.Height))
	}
	// Render resources panel with tabs (Resources / State)
	leftPanels = append(leftPanels, enforceDimensions(m.renderResourcesPanelWithTabs(layout.Resources.Width, layout.Resources.Height), layout.LeftColumnWidth, layout.Resources.Height))
	if m.historyPanel != nil && m.executionMode {
		leftPanels = append(leftPanels, enforceDimensions(m.historyPanel.View(), layout.LeftColumnWidth, layout.History.Height))
	}
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)

	// Left column total height should equal sum of panel heights
	leftColumn = lipgloss.NewStyle().
		Width(layout.LeftColumnWidth).
		MaxWidth(layout.LeftColumnWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(leftColumn)

	// Render right column (main area + command log)
	var rightPanels []string
	if m.mainArea != nil {
		rightPanels = append(rightPanels, enforceDimensions(m.mainArea.View(), layout.RightColumnWidth, layout.Main.Height))
	}
	if m.panelManager.IsCommandLogVisible() && m.commandLogPanel != nil {
		commandLogView := m.commandLogPanel.View()
		if commandLogView != "" {
			rightPanels = append(rightPanels, enforceDimensions(commandLogView, layout.RightColumnWidth, layout.CommandLog.Height))
		}
	}
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)

	// Right column total height should equal main + command log heights
	rightColumn = lipgloss.NewStyle().
		Width(layout.RightColumnWidth).
		MaxWidth(layout.RightColumnWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(rightColumn)

	// Join left and right columns horizontally
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	return mainContent
}

// renderResourcesPanelWithTabs renders the resources panel with tabs (Resources / State)
func (m *Model) renderResourcesPanelWithTabs(width, height int) string {
	if m.resourceList == nil {
		return ""
	}

	// In non-execution mode, just show the resource list (no tabs)
	if !m.executionMode {
		return m.resourceList.View()
	}

	// Determine which content to show based on active tab
	var content string
	if m.resourcesActiveTab == 0 {
		// Resources tab - use the resource list's view but we'll modify the title
		// ResourceList already renders with border, so use full dimensions
		m.resourceList.SetSize(width, height)
		content = m.resourceList.View()
		// The resource list already renders with border and title, so return it with modified title
		return m.addTabsToPanel(content, width, []string{"Resources", "State"}, m.resourcesActiveTab)
	}

	// State tab
	if m.stateListContent == nil {
		m.stateListContent = components.NewStateListContent(m.styles)
		m.stateListContent.OnSelect = func(address string) tea.Cmd {
			return m.beginStateShow(address)
		}
	}
	// Content area is panel size minus borders (2 for border)
	m.stateListContent.SetSize(width-2, height-2)
	stateView := m.stateListContent.View()

	// Wrap in panel border
	focused := m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources
	borderStyle := m.styles.Border
	if focused {
		borderStyle = m.styles.FocusedBorder
	}

	panel := borderStyle.
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Width(width - 2).
		Height(height - 2).
		Render(stateView)

	return m.addTabsToPanel(panel, width, []string{"Resources", "State"}, m.resourcesActiveTab)
}

// addTabsToPanel adds tab indicators to the first line of a panel
func (m *Model) addTabsToPanel(panel string, width int, tabs []string, activeTab int) string {
	lines := strings.Split(panel, "\n")
	if len(lines) == 0 {
		return panel
	}

	// Build tab title
	focused := m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources
	titleStyle := m.styles.PanelTitle
	if focused {
		titleStyle = m.styles.FocusedPanelTitle
	}

	var tabParts []string
	tabParts = append(tabParts, "[2]")
	for i, tab := range tabs {
		if i == activeTab {
			tabParts = append(tabParts, "["+tab+"]")
		} else {
			tabParts = append(tabParts, m.styles.Dimmed.Render(tab))
		}
	}
	tabTitle := strings.Join(tabParts, " ")
	titleRendered := titleStyle.Render(" " + tabTitle + " ")

	// Try to replace the title in the first line
	firstLine := lines[0]
	// Find position after first border character
	if len(firstLine) > 2 {
		// Replace title section
		borderStyle := m.styles.Border
		if focused {
			borderStyle = m.styles.FocusedBorder
		}
		if newLine, ok := components.RenderPanelTitleLine(width, borderStyle, titleRendered); ok {
			lines[0] = newLine
		}
	}

	return strings.Join(lines, "\n")
}

func (m *Model) renderLegacyMainContent() string {
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

func (m *Model) renderFullScreenCommandLog() string {
	if m.commandLogPanel == nil {
		return m.styles.Dimmed.Render("No logs available")
	}

	// Get diagnostics panel content
	diagPanel := m.commandLogPanel.GetDiagnosticsPanel()
	if diagPanel == nil {
		return m.styles.Dimmed.Render("No logs available")
	}

	// Calculate dimensions for full screen
	// Reserve space for title (1), border (2), and footer (1)
	contentHeight := m.height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}
	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Update diagnostics panel size to full screen
	diagPanel.SetSize(contentWidth, contentHeight)

	// Get the content
	content := diagPanel.View()

	// Build title
	title := " Command Log (Full Screen) "
	titleRendered := m.styles.FocusedPanelTitle.Render(title)

	// Build footer with help text
	footer := m.styles.Dimmed.Render("Press 'esc' to exit | arrow keys to scroll")

	// Wrap in border
	panel := m.styles.FocusedBorder.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)

	// Add title to top border
	lines := strings.Split(panel, "\n")
	if len(lines) > 0 {
		// Replace first line with title
		if line, ok := components.RenderPanelTitleLine(contentWidth+2, m.styles.FocusedBorder, titleRendered); ok {
			lines[0] = line
		}
	}
	panel = strings.Join(lines, "\n")

	// Join panel and footer
	return lipgloss.JoinVertical(lipgloss.Left, panel, footer)
}

func (m *Model) updateHelpModalContent() {
	if m.helpModal == nil {
		return
	}

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
			title: "Panel Navigation",
			rows: []helpRow{
				{keys: "1", desc: "focus workspace panel"},
				{keys: "2", desc: "focus resource list"},
				{keys: "3", desc: "focus history"},
				{keys: "0", desc: "focus main area"},
				{keys: "4", desc: "focus command log (enter for full screen)"},
				{keys: "tab", desc: "cycle panels"},
				{keys: "L", desc: "toggle command log"},
			},
		},
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
				{keys: "1 then e", desc: "select environment"},
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
				{keys: "f", desc: "refresh state"},
				{keys: "v", desc: "validate configuration"},
				{keys: "F", desc: "format code (fmt)"},
				{keys: "a", desc: "confirm apply"},
				{keys: "h", desc: "toggle history panel"},
				{keys: "tab", desc: "focus history panel"},
				{keys: "ctrl+c", desc: "cancel running command"},
				{keys: "s", desc: "toggle status column"},
				{keys: "C", desc: "toggle compact progress view"},
				{keys: "D", desc: "focus logs panel"},
				{keys: "[/]", desc: "switch tabs in panel"},
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
	lines = append(lines, m.styles.Dimmed.Render("esc: close"))

	m.helpModal.SetTitle("Keybinds")
	m.helpModal.SetContent(strings.TrimRight(strings.Join(lines, "\n"), "\n"))
	m.helpModal.Show()
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
}

func (m *Model) inputCaptured() bool {
	return false
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

func (m *Model) loadFilterPreferences() {
	if !m.executionMode {
		return
	}
	workDir := m.envWorkDir
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return
	}
	workspace := m.envCurrent
	if workspace == "" {
		workspace = "default"
	}
	pref, err := environment.LoadFilterPreference(absWorkDir, workspace)
	if err != nil || pref == nil {
		return
	}
	m.filterCreate = pref.FilterCreate
	m.filterUpdate = pref.FilterUpdate
	m.filterDelete = pref.FilterDelete
	m.filterReplace = pref.FilterReplace
	// Apply to resource list
	m.resourceList.SetFilter(terraform.ActionCreate, m.filterCreate)
	m.resourceList.SetFilter(terraform.ActionUpdate, m.filterUpdate)
	m.resourceList.SetFilter(terraform.ActionDelete, m.filterDelete)
	m.resourceList.SetFilter(terraform.ActionReplace, m.filterReplace)
}

func (m *Model) saveFilterPreferences() {
	if !m.executionMode {
		return
	}
	workDir := m.envWorkDir
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return
	}
	workspace := m.envCurrent
	if workspace == "" {
		workspace = "default"
	}
	pref := environment.FilterPreference{
		FilterCreate:  m.filterCreate,
		FilterUpdate:  m.filterUpdate,
		FilterDelete:  m.filterDelete,
		FilterReplace: m.filterReplace,
	}
	_ = environment.SaveFilterPreference(absWorkDir, workspace, pref)
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

	// Update overlay component sizes
	if m.helpModal != nil {
		m.helpModal.SetSize(m.width, m.height)
	}
	if m.toast != nil {
		m.toast.SetSize(m.width, m.height)
	}

	// Use panel manager if available
	if m.panelManager != nil {
		// Calculate layout using panel manager (it will handle status bar internally)
		layout := m.panelManager.CalculateLayout(m.width, m.height)

		// Update all panel sizes
		m.panelManager.UpdatePanelSizes(layout)
		return
	}

	// Legacy layout code
	m.updateLegacyLayout()
}

func (m *Model) updateLegacyLayout() {
	reserved := lipgloss.Height(m.renderStatusBar())
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
		// Note: These views are deprecated, staying in viewMain now
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
		case "esc":
			if !m.planRunning && !m.applyRunning {
				m.execView = viewMain
				return true, nil
			}
		}
	case viewCommandLog:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "esc":
			m.execView = viewMain
			return true, nil
		}
	case viewStateList:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "esc":
			m.execView = viewMain
			return true, nil
		case "up", "k":
			if m.stateListView != nil {
				m.stateListView.MoveUp()
			}
			return true, nil
		case "down", "j":
			if m.stateListView != nil {
				m.stateListView.MoveDown()
			}
			return true, nil
		case "enter":
			if m.stateListView != nil {
				if res := m.stateListView.GetSelected(); res != nil {
					return true, m.beginStateShow(res.Address)
				}
			}
			return true, nil
		}
	case viewStateShow:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "esc":
			m.execView = viewStateList
			return true, nil
		}
		// Forward scroll keys to the view
		if m.stateShowView != nil {
			m.stateShowView, _ = m.stateShowView.Update(tea.KeyMsg(msg))
		}
		return true, nil
	default:
		switch key {
		case "p":
			return true, m.beginPlan()
		case "f":
			return true, m.beginRefresh()
		case "v":
			return true, m.beginValidate()
		case "F":
			return true, m.beginFormat()
		case "a":
			if m.plan == nil {
				var cmd tea.Cmd
				if m.toast != nil {
					cmd = m.toast.ShowError("No plan loaded; run terraform plan first")
				}
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
		case "D":
			// Focus the command log panel where diagnostics are shown
			if m.panelManager != nil && m.panelManager.IsCommandLogVisible() {
				return true, m.panelManager.SetFocus(PanelCommandLog)
			}
			// If command log not visible, show it and focus it
			if m.panelManager != nil {
				m.panelManager.SetCommandLogVisible(true)
				m.updateLayout()
				return true, m.panelManager.SetFocus(PanelCommandLog)
			}
			return true, nil
		case "tab":
			if m.showHistory && len(m.historyEntries) > 0 {
				m.historyFocused = !m.historyFocused
				m.syncHistorySelection()
				return true, nil
			}
		case "ctrl+c":
			if m.planRunning || m.applyRunning || m.refreshRunning {
				m.cancelExecution()
				return true, nil
			}
		case "esc":
			// Exit history detail mode when pressing esc
			if m.mainArea != nil && m.mainArea.GetMode() == ModeHistoryDetail {
				m.mainArea.ExitHistoryDetail()
				return true, nil
			}
		}
		// Handle history keys when history panel is focused (history is always visible in execution mode)
		if m.historyFocused {
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
	m.planStartedAt = time.Now()

	// Keep in main view, switch MainArea to logs mode during plan
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeLogs)
	}

	// Show command log panel during operations
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	if m.applyView != nil {
		m.applyView.Reset()
		m.applyView.SetTitle("Running terraform plan...")
		m.applyView.SetStatusText("Running...", "Plan complete", "Plan failed - press esc to return")
		m.applyView.SetStatus(views.ApplyRunning)
	}
	m.updateExecutionViewForStreaming()

	return func() tea.Msg {
		result, output, err := m.executor.Plan(ctx, terraform.PlanOptions{
			Flags: planFlags,
			Env:   planEnv,
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
	m.applyStartedAt = time.Now()

	// Keep in main view, switch MainArea to logs mode during apply
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeLogs)
	}

	// Show command log panel during operations
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	if m.applyView != nil {
		m.applyView.Reset()
		m.applyView.SetTitle("Applying changes...")
		m.applyView.SetStatusText("Running...", "Apply complete", "Apply failed - press esc to return")
		m.applyView.SetStatus(views.ApplyRunning)
	}
	// Transition to main view from confirm view
	if m.execView == viewPlanConfirm {
		m.execView = viewMain
	}
	m.updateExecutionViewForStreaming()

	return func() tea.Msg {
		result, output, err := m.executor.Apply(ctx, terraform.ApplyOptions{
			Flags:       m.applyFlags,
			AutoApprove: true,
			Env:         applyEnv,
		})
		return ApplyStartMsg{Result: result, Output: output, Error: err}
	}
}

func (m *Model) beginRefresh() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.planRunning || m.applyRunning || m.refreshRunning {
		return nil
	}
	m.err = nil
	refreshEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	m.refreshRunning = true
	m.refreshStartedAt = time.Now()

	// Keep in main view, switch MainArea to logs mode during refresh
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeLogs)
	}

	// Show command log panel during operations
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	if m.applyView != nil {
		m.applyView.Reset()
		m.applyView.SetTitle("Running terraform refresh...")
		m.applyView.SetStatusText("Running...", "Refresh complete", "Refresh failed - press esc to return")
		m.applyView.SetStatus(views.ApplyRunning)
	}
	m.updateExecutionViewForStreaming()

	return func() tea.Msg {
		result, output, err := m.executor.Refresh(ctx, terraform.RefreshOptions{
			Env: refreshEnv,
		})
		return RefreshStartMsg{Result: result, Output: output, Error: err}
	}
}

func (m *Model) handleRefreshStart(msg RefreshStartMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.refreshRunning = false
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Failed to start terraform refresh: %v", msg.Error))
		}
		m.addErrorDiagnostic("Refresh failed to start", msg.Error, "")
		return m, nil
	}

	m.outputChan = msg.Output
	cmds := []tea.Cmd{
		m.waitRefreshCompleteCmd(msg.Result),
		m.streamRefreshOutputCmd(),
	}
	if m.applyView != nil {
		cmds = append(cmds, m.applyView.Tick())
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleRefreshComplete(msg RefreshCompleteMsg) (tea.Model, tea.Cmd) {
	m.refreshRunning = false
	m.cancelFunc = nil
	m.outputChan = nil

	// Switch MainArea back to diff mode when refresh completes
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeDiff)
	}

	// Log to session history
	output := ""
	if msg.Result != nil {
		output = msg.Result.Output
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("terraform apply -refresh-only", output)
	}

	if msg.Error != nil || !msg.Success {
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			if msg.Error != nil {
				m.applyView.AppendLine(fmt.Sprintf("Refresh failed: %v", msg.Error))
			}
		}
		// Route logs to command log panel
		if msg.Result != nil {
			if m.commandLogPanel != nil {
				m.commandLogPanel.SetLogText(msg.Result.Output)
				m.commandLogPanel.SetParsedText(utils.FormatLogOutput(msg.Result.Output))
			} else if m.diagnosticsPanel != nil {
				m.diagnosticsPanel.SetLogText(msg.Result.Output)
				m.diagnosticsPanel.SetParsedText(utils.FormatLogOutput(msg.Result.Output))
			}
		}
		if msg.Error != nil {
			output := ""
			if msg.Result != nil {
				output = msg.Result.Output
			}
			m.addErrorDiagnostic("Refresh failed", msg.Error, output)
		}
		m.updateExecutionViewForStreaming()
		cmd := m.recordOperationCmd("refresh", nil, true, m.refreshStartedAt, msg.Result, "", msg.Error)
		return m, cmd
	}

	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}
	// Route logs to command log panel
	parsed := ""
	if msg.Result != nil {
		parsed = utils.FormatLogOutput(msg.Result.Output)
	}
	if strings.TrimSpace(parsed) == "" {
		parsed = "Refresh complete"
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.SetParsedText(parsed)
	} else if m.diagnosticsPanel != nil {
		m.diagnosticsPanel.SetParsedText(parsed)
	}
	m.updateExecutionViewForStreaming()
	var toastCmd tea.Cmd
	if m.toast != nil {
		toastCmd = m.toast.ShowSuccess("State refreshed successfully")
	}
	return m, tea.Batch(
		toastCmd,
		m.recordOperationCmd("refresh", nil, true, m.refreshStartedAt, msg.Result, "", nil),
	)
}

func (m *Model) streamRefreshOutputCmd() tea.Cmd {
	return func() tea.Msg {
		if m.outputChan == nil {
			return nil
		}
		line, ok := <-m.outputChan
		if !ok {
			return nil
		}
		return RefreshOutputMsg{Line: line}
	}
}

func (m *Model) waitRefreshCompleteCmd(result *terraform.ExecutionResult) tea.Cmd {
	return func() tea.Msg {
		if result == nil {
			return RefreshCompleteMsg{Success: false, Error: errors.New("refresh execution result missing")}
		}
		<-result.Done()
		if result.Error != nil {
			return RefreshCompleteMsg{Success: false, Error: result.Error, Result: result}
		}
		return RefreshCompleteMsg{Success: true, Result: result}
	}
}

func (m *Model) beginValidate() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.planRunning || m.applyRunning || m.refreshRunning {
		return nil
	}
	if m.toast != nil {
		m.toast.ShowInfo("Running terraform validate...")
	}
	validateEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		result, err := m.executor.Validate(ctx, terraform.ValidateOptions{
			Env: validateEnv,
		})
		if err != nil {
			return ValidateCompleteMsg{Error: err}
		}
		// Parse JSON output
		var validateResult terraform.ValidateResult
		if result != nil && result.Stdout != "" {
			if parseErr := json.Unmarshal([]byte(result.Stdout), &validateResult); parseErr != nil {
				return ValidateCompleteMsg{Error: parseErr, RawOutput: result.Stdout, ExecResult: result}
			}
		}
		return ValidateCompleteMsg{Result: &validateResult, RawOutput: result.Stdout, ExecResult: result}
	}
}

func (m *Model) handleValidateComplete(msg ValidateCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.addErrorDiagnostic("Validate failed", msg.Error, msg.RawOutput)
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("Validate failed: %v", msg.Error))
		}
		return m, cmd
	}

	if msg.Result == nil {
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowInfo("Validate completed (no result)")
		}
		return m, cmd
	}

	// Display diagnostics in command log panel
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	if len(msg.Result.Diagnostics) > 0 {
		if m.commandLogPanel != nil {
			m.commandLogPanel.SetDiagnostics(msg.Result.Diagnostics)
		} else if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetDiagnostics(msg.Result.Diagnostics)
		}
	}

	var cmd tea.Cmd
	if msg.Result.Valid {
		if m.toast != nil {
			cmd = m.toast.ShowSuccess("Configuration is valid")
		}
	} else {
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("Validation failed: %d errors, %d warnings", msg.Result.ErrorCount, msg.Result.WarningCount))
		}
	}
	return m, cmd
}

func (m *Model) beginFormat() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.planRunning || m.applyRunning || m.refreshRunning {
		return nil
	}
	if m.toast != nil {
		m.toast.ShowInfo("Running terraform fmt...")
	}
	formatEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		result, err := m.executor.Format(ctx, terraform.FormatOptions{
			Recursive: true,
			Env:       formatEnv,
		})
		if err != nil {
			return FormatCompleteMsg{Error: err}
		}
		// Parse output - each line is a changed file
		var changedFiles []string
		if result != nil && result.Stdout != "" {
			for _, line := range strings.Split(result.Stdout, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					changedFiles = append(changedFiles, trimmed)
				}
			}
		}
		return FormatCompleteMsg{ChangedFiles: changedFiles, ExecResult: result}
	}
}

func (m *Model) handleFormatComplete(msg FormatCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.addErrorDiagnostic("Format failed", msg.Error, "")
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("Format failed: %v", msg.Error))
		}
		return m, cmd
	}

	var cmd tea.Cmd
	if len(msg.ChangedFiles) == 0 {
		if m.toast != nil {
			cmd = m.toast.ShowInfo("No files changed")
		}
	} else {
		if m.toast != nil {
			cmd = m.toast.ShowSuccess(fmt.Sprintf("Formatted %d file(s)", len(msg.ChangedFiles)))
		}

		// Display changed files in command log panel
		if m.panelManager != nil {
			m.panelManager.SetCommandLogVisible(true)
			m.updateLayout()
		}

		output := "Formatted files:\n" + strings.Join(msg.ChangedFiles, "\n")
		if m.commandLogPanel != nil {
			m.commandLogPanel.SetParsedText(output)
		} else if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetParsedText(output)
		}
	}
	return m, cmd
}

func (m *Model) beginStateList() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.planRunning || m.applyRunning || m.refreshRunning {
		return nil
	}
	if m.toast != nil {
		m.toast.ShowInfo("Loading state list...")
	}
	stateEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		result, err := m.executor.StateList(ctx, terraform.StateListOptions{
			Env: stateEnv,
		})
		if err != nil {
			return StateListCompleteMsg{Error: err}
		}
		if result.Error != nil {
			return StateListCompleteMsg{Error: result.Error}
		}
		// Parse output - each line is a resource address
		var resources []terraform.StateResource
		if result.Stdout != "" {
			for _, line := range strings.Split(result.Stdout, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					resources = append(resources, terraform.StateResource{
						Address: trimmed,
					})
				}
			}
		}
		return StateListCompleteMsg{Resources: resources}
	}
}

func (m *Model) handleStateListComplete(msg StateListCompleteMsg) (tea.Model, tea.Cmd) {
	// Hide loading toast
	if m.toast != nil {
		m.toast.Hide()
	}

	if msg.Error != nil {
		if m.stateListContent != nil {
			m.stateListContent.SetError(msg.Error.Error())
		}
		m.addErrorDiagnostic("State list failed", msg.Error, "")
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("State list failed: %v", msg.Error))
		}
		return m, cmd
	}

	m.stateResources = msg.Resources

	// Update state list content (for tab view)
	if m.stateListContent != nil {
		m.stateListContent.SetResources(msg.Resources)
	}

	// Initialize state list view if needed (for full screen view)
	if m.stateListView == nil {
		m.stateListView = views.NewStateListView(m.styles)
	}
	m.stateListView.SetSize(m.width, m.height)
	m.stateListView.SetResources(msg.Resources)

	// If we're on the State tab, stay there; otherwise only switch if explicitly requested
	// (The 'S' keybinding has been removed, so this only happens when switching tabs)

	return m, nil
}

func (m *Model) beginStateShow(address string) tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.toast != nil {
		m.toast.ShowInfo("Loading state...")
	}
	stateEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		result, err := m.executor.StateShow(ctx, address, terraform.StateShowOptions{
			Env: stateEnv,
		})
		if err != nil {
			return StateShowCompleteMsg{Address: address, Error: err}
		}
		if result.Error != nil {
			return StateShowCompleteMsg{Address: address, Error: result.Error}
		}
		return StateShowCompleteMsg{Address: address, Output: result.Stdout}
	}
}

func (m *Model) handleStateShowComplete(msg StateShowCompleteMsg) (tea.Model, tea.Cmd) {
	// Log to session history
	output := msg.Output
	if msg.Error != nil {
		output = msg.Error.Error()
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog(fmt.Sprintf("terraform state show %s", msg.Address), output)
	}

	if m.toast != nil {
		m.toast.Hide()
	}
	if msg.Error != nil {
		m.addErrorDiagnostic("State show failed", msg.Error, "")
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("State show failed: %v", msg.Error))
		}
		return m, cmd
	}

	// Initialize state show view if needed
	if m.stateShowView == nil {
		m.stateShowView = views.NewStateShowView(m.styles)
	}
	m.stateShowView.SetSize(m.width, m.height)
	m.stateShowView.SetAddress(msg.Address)
	m.stateShowView.SetContent(msg.Output)

	// Switch to state show view
	m.execView = viewStateShow

	return m, nil
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
		m.streamPlanOutputCmd(),
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

	// Switch MainArea back to diff mode when plan completes
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeDiff)
	}

	// Log to session history
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("terraform plan", msg.Output)
	}

	if msg.Error != nil {
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Plan failed: %v", msg.Error))
		}
		m.planFilePath = ""
		m.planRunFlags = nil
		m.addErrorDiagnostic("Plan failed", msg.Error, msg.Output)
		// Route logs to command log panel
		if m.commandLogPanel != nil {
			m.commandLogPanel.SetLogText(msg.Output)
			m.commandLogPanel.SetParsedText(utils.FormatLogOutput(msg.Output))
		} else if m.diagnosticsPanel != nil {
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
		// Route logs to command log panel
		if m.commandLogPanel != nil {
			m.commandLogPanel.SetLogText(msg.Output)
			m.commandLogPanel.SetParsedText(utils.FormatLogOutput(msg.Output))
		} else if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetLogText(msg.Output)
			m.diagnosticsPanel.SetParsedText(utils.FormatLogOutput(msg.Output))
		}
	}
	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}
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
		m.streamApplyOutputCmd(),
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

	// Switch MainArea back to diff mode when apply completes
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeDiff)
	}

	// Log to session history
	output := ""
	if msg.Result != nil {
		output = msg.Result.Output
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("terraform apply", output)
	}

	if msg.Error != nil || !msg.Success {
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			if msg.Error != nil {
				m.applyView.AppendLine(fmt.Sprintf("Apply failed: %v", msg.Error))
			}
		}
		// Route logs to command log panel
		if msg.Result != nil {
			if m.commandLogPanel != nil {
				m.commandLogPanel.SetLogText(msg.Result.Output)
				m.commandLogPanel.SetParsedText(utils.FormatLogOutput(msg.Result.Output))
			} else if m.diagnosticsPanel != nil {
				m.diagnosticsPanel.SetLogText(msg.Result.Output)
				m.diagnosticsPanel.SetParsedText(utils.FormatLogOutput(msg.Result.Output))
			}
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
		opErr := msg.Error
		if opErr == nil && !msg.Success {
			opErr = errors.New("apply failed")
		}
		m.updateExecutionViewForStreaming()
		return m, tea.Batch(
			m.recordHistoryCmd(status, m.flattenSummary(m.planSummary()), m.lastPlanOutput, msg.Result, msg.Error),
			m.recordOperationCmd("apply", m.applyFlags, true, m.applyStartedAt, msg.Result, "", opErr),
		)
	}

	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}
	summary := m.planSummary()
	// Route logs to command log panel
	parsed := ""
	if msg.Result != nil {
		parsed = utils.FormatLogOutput(msg.Result.Output)
	}
	if strings.TrimSpace(parsed) == "" {
		parsed = "Apply complete"
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.SetParsedText(parsed)
	} else if m.diagnosticsPanel != nil {
		m.diagnosticsPanel.SetParsedText(parsed)
	}
	if m.applyView != nil && msg.Result != nil {
		parsed := utils.FormatLogOutput(msg.Result.Output)
		if strings.TrimSpace(parsed) == "" {
			parsed = strings.TrimSpace(msg.Result.Output)
		}
		m.applyView.SetOutput(parsed)
	}
	// Stay in main view with panel layout
	m.setPlan(&terraform.Plan{Resources: nil})
	m.planFilePath = ""
	m.planRunFlags = nil
	m.updateExecutionViewForStreaming()
	return m, tea.Batch(
		m.recordHistoryCmd(history.StatusSuccess, m.flattenSummary(summary), m.lastPlanOutput, msg.Result, nil),
		m.recordOperationCmd("apply", m.applyFlags, true, m.applyStartedAt, msg.Result, "", nil),
	)
}

func (m *Model) updateExecutionViewForStreaming() {
	if m.execView == viewPlanConfirm {
		return
	}
	// Don't interrupt history detail mode when showing in main area
	if m.mainArea != nil && m.mainArea.GetMode() == ModeHistoryDetail {
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

	// Ensure command log is visible when errors occur
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
	}

	if m.operationState != nil {
		m.operationState.AddDiagnostic(diag)
		// Route diagnostics to command log panel
		if m.commandLogPanel != nil {
			m.commandLogPanel.SetDiagnostics(m.operationState.GetDiagnostics())
		} else if m.diagnosticsPanel != nil {
			m.diagnosticsPanel.SetDiagnostics(m.operationState.GetDiagnostics())
		}
		return
	}
	// Route diagnostics to command log panel
	if m.commandLogPanel != nil {
		m.commandLogPanel.SetDiagnostics([]terraform.Diagnostic{diag})
	} else if m.diagnosticsPanel != nil {
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

		parseInput := output
		if m.executor != nil && m.planFilePath != "" {
			planEnv, err := m.prepareTerraformEnv()
			if err != nil {
				planEnv = nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			showResult, showErr := m.executor.Show(ctx, m.planFilePath, terraform.ShowOptions{Env: planEnv})
			cancel()
			if showErr == nil && showResult != nil && strings.TrimSpace(showResult.Output) != "" {
				parseInput = showResult.Output
			}
		}

		textParser := tfparser.NewTextParser()
		plan, err := textParser.Parse(strings.NewReader(parseInput))
		if err != nil {
			return PlanCompleteMsg{Error: fmt.Errorf("parse plan output: %w", err), Result: result, Output: output}
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
		Command:     m.buildCommand(action, flags, autoApprove),
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

func (m *Model) renderToastOverlay(baseView string, message string, isError bool) string {
	if m.styles == nil || m.width < 20 || m.height < 3 {
		return baseView
	}

	// Create toast style - small box on top right
	toastStyle := m.styles.Highlight.
		Padding(0, 1).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Highlight.GetForeground())

	if isError {
		toastStyle = m.styles.Delete.
			Padding(0, 1).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.styles.Delete.GetForeground())
	}

	// Render toast message in a small box
	toast := toastStyle.Render(message)
	toastWidth := lipgloss.Width(toast)
	toastHeight := lipgloss.Height(toast)

	// Position at top right corner
	lines := strings.Split(baseView, "\n")
	if len(lines) < toastHeight {
		return baseView
	}

	// Overlay toast on the first few lines, right-aligned
	toastLines := strings.Split(toast, "\n")
	for i := 0; i < len(toastLines) && i < len(lines); i++ {
		line := lines[i]
		lineWidth := lipgloss.Width(line)

		// Calculate position for right alignment (with some padding from edge)
		padding := 2
		if lineWidth+toastWidth+padding <= m.width {
			// Overlay toast on the right side of this line
			// We need to handle ANSI codes carefully
			visibleLen := lipgloss.Width(line)
			if visibleLen+toastWidth+padding < m.width {
				// Add spaces to push toast to the right
				spaces := m.width - visibleLen - toastWidth - padding
				lines[i] = line + strings.Repeat(" ", spaces) + toastLines[i]
			} else {
				// Truncate line and add toast
				truncateAt := m.width - toastWidth - padding - 3
				if truncateAt > 0 {
					lines[i] = lipgloss.NewStyle().Width(truncateAt).Render(line) + "..." + toastLines[i]
				}
			}
		}
	}

	return strings.Join(lines, "\n")
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

func (m *Model) buildCommand(action string, flags []string, autoApprove bool) string {
	args := []string{action}
	args = append(args, flags...)
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
		// Show history detail in main area [0] instead of full-screen view
		if m.mainArea != nil {
			m.mainArea.EnterHistoryDetail()
			m.mainArea.SetHistoryContent("Apply details", "Loading...")
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
