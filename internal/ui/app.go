package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
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
			// Don't return early - allow navigation keys (2, 3, etc.) to be processed below
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
					cmd := m.beginStateList()
					return m, cmd
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
					cmd := m.beginStateList()
					return m, cmd
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
		case viewMain, viewPlanOutput, viewApplyOutput, viewHistoryDetail, viewDiagnostics:
			// Fall through to main view rendering below
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
