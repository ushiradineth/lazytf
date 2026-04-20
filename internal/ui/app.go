package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/notifications"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
	"github.com/ushiradineth/lazytf/internal/ui/views"
	"github.com/ushiradineth/lazytf/internal/utils"
)

// Model is the main application model.
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
	executionMode      bool
	executor           terraform.ExecutorInterface
	applyView          *views.ApplyView
	planView           *views.PlanView
	planFlags          []string
	applyFlags         []string
	planRunFlags       []string
	applyRunFlags      []string
	planRunning        bool
	applyRunning       bool
	refreshRunning     bool
	operationRunning   bool
	outputChan         <-chan string
	cancelFunc         context.CancelFunc
	execView           executionView
	planStartedAt      time.Time
	applyStartedAt     time.Time
	refreshStartedAt   time.Time
	planFilePath       string
	operationState     *terraform.OperationState
	diagnosticsPanel   *components.DiagnosticsPanel
	historyStore       *history.Store
	historyPanel       *components.HistoryPanel
	historyEntries     []history.Entry
	historyEnabled     bool
	showHistory        bool
	historyHeight      int
	historySelected    int
	historyFocused     bool
	diagnosticsFocused bool
	historyDetail      *history.Entry
	historyLogger      *history.Logger
	notifier           notifications.Notifier
	stateListView      *views.StateListView
	stateShowView      *views.StateShowView
	stateListContent   *components.StateListContent
	stateMoveSource    string
	stateMoveInput     string
	stateMoveCursorOn  bool
	pendingConfirmCmd  tea.Cmd
	resourcesActiveTab int // 0 = Resources, 1 = State
	lastPlanOutput     string
	planEnvironment    string
	planWorkDir        string
	targetModeEnabled  bool
	targetPlanPinned   string
	planTargetSnapshot string
	pendingTargetApply bool
	pendingTargetSig   string
	config             *config.Config
	configManager      *config.Manager
	envWorkDir         string
	envCurrent         string
	envStrategy        environment.StrategyType
	envDetection       *environment.DetectionResult
	envOptions         []environment.Environment

	// Overlay components
	toast      *components.Toast
	helpModal  *components.Modal
	themeModal *components.Modal

	// Theme switching
	previewThemeName string
	originalStyles   *styles.Styles

	// Panel system
	panelManager        *PanelManager
	environmentPanel    *components.EnvironmentPanel
	mainArea            *MainArea
	commandLogPanel     *components.CommandLogPanel
	resourcesController *ResourcesPanelController

	// Progress indicator
	progressIndicator *components.ProgressIndicator

	// Keybind registry
	keybindRegistry *keybinds.Registry

	// Mouse handling state
	lastMouseLeftPressX int
	lastMouseLeftPressY int
	lastMouseLeftPress  bool
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
	ModalConfirmApply
	ModalStateMoveDestination
	ModalTheme
)

type mouseIntent int

const (
	mouseIntentNone mouseIntent = iota
	mouseIntentLeftClick
	mouseIntentWheelUp
	mouseIntentWheelDown
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
	Executor               *terraform.Executor
	Flags                  []string
	WorkDir                string
	EnvName                string
	PreloadedPlanPath      string
	PreloadedPlanEnv       string
	PreloadedPlanDir       string
	PreloadedPlanFromStdin bool
	HistoryStore           *history.Store
	HistoryLogger          *history.Logger
	HistoryEnabled         bool
	Notifier               notifications.Notifier
	Config                 *config.Config
	ConfigManager          *config.Manager
}

// NewModel creates a new application model.
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
	toast.SetPosition(components.ToastTopRight)
	helpModal := components.NewModal(appStyles)
	themeModal := components.NewModal(appStyles)

	// Initialize resources panel controller
	resourcesController := NewResourcesPanelController(resourceList)

	// Initialize progress indicator
	progressIndicator := components.NewProgressIndicator(appStyles)

	// Initialize keybind registry (non-execution mode by default)
	kbRegistry := keybinds.NewRegistry()
	keybinds.RegisterDefaults(kbRegistry, false)
	keybinds.RegisterWorkspacePanelBindings(kbRegistry)

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
		// Overlay components
		toast:      toast,
		helpModal:  helpModal,
		themeModal: themeModal,

		// Panel system
		panelManager:        panelManager,
		environmentPanel:    environmentPanel,
		mainArea:            mainArea,
		commandLogPanel:     commandLogPanel,
		resourcesController: resourcesController,

		// Progress indicator
		progressIndicator: progressIndicator,

		// Keybind registry
		keybindRegistry: kbRegistry,
	}

	// Register panels with manager
	panelManager.RegisterPanel(PanelWorkspace, environmentPanel)
	panelManager.RegisterPanel(PanelResources, resourceList)
	panelManager.RegisterPanel(PanelHistory, nil) // Will be set later
	panelManager.RegisterPanel(PanelMain, mainArea)
	panelManager.RegisterPanel(PanelCommandLog, commandLogPanel)

	// Calculate diffs for all resources
	if plan != nil {
		resourceList.SetResources(plan.Resources)
	}

	// Initialize focus on the default panel (Resources)
	panelManager.SetFocus(PanelResources)

	// Register keybind handlers
	m.registerKeybindHandlers()

	return m
}

// NewExecutionModel creates a model configured for terraform execution.
func NewExecutionModel(plan *terraform.Plan, cfg ExecutionConfig) *Model {
	return NewExecutionModelWithStyles(plan, cfg, styles.DefaultStyles())
}

// NewExecutionModelWithStyles creates a model configured for terraform execution with styles.
//
//nolint:funlen // Constructor wires execution dependencies and startup state in one place.
func NewExecutionModelWithStyles(plan *terraform.Plan, cfg ExecutionConfig, appStyles *styles.Styles) *Model {
	m := NewModelWithStyles(plan, appStyles)
	m.executionMode = true
	if m.panelManager != nil {
		m.panelManager.SetExecutionMode(true)
	}
	// Re-initialize keybind registry with execution mode
	m.keybindRegistry = keybinds.NewRegistry()
	keybinds.RegisterDefaults(m.keybindRegistry, true)
	keybinds.RegisterWorkspacePanelBindings(m.keybindRegistry)
	m.registerKeybindHandlers()
	// Only assign executor if non-nil to avoid Go interface nil gotcha
	// (typed nil pointer assigned to interface creates non-nil interface)
	if cfg.Executor != nil {
		m.executor = cfg.Executor
	}
	m.planFlags = append([]string{}, cfg.Flags...)
	m.applyFlags = append([]string{}, cfg.Flags...)
	m.notifier = cfg.Notifier
	if m.notifier == nil {
		m.notifier = notifications.NopNotifier{}
	}
	m.envWorkDir = cfg.WorkDir
	m.envCurrent = cfg.EnvName
	if strings.TrimSpace(cfg.PreloadedPlanPath) != "" {
		m.planFilePath = cfg.PreloadedPlanPath
		m.planEnvironment = cfg.PreloadedPlanEnv
		m.planWorkDir = cfg.PreloadedPlanDir
	}
	m.resourcesActiveTab = 0
	if m.resourcesController != nil {
		m.resourcesController.SetActiveTab(0)
	}
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelResources)
	}
	m.envStrategy = environment.StrategyUnknown
	m.planView = views.NewPlanView("", m.styles)
	m.applyView = views.NewApplyView(m.styles)
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
	if plan == nil {
		m.mainArea.SetMode(ModeAbout)
		if m.panelManager != nil {
			m.panelManager.SetFocus(PanelResources)
		}
	}

	// Initialize state list content for the Resources panel tab
	m.stateListContent = components.NewStateListContent(m.styles)
	m.stateListContent.OnSelect = func(address string) tea.Cmd {
		return m.beginStateShow(address)
	}

	// Connect state list content to resources controller
	if m.resourcesController != nil {
		m.resourcesController.SetStateListContent(m.stateListContent)
	}

	m.historyEnabled = cfg.HistoryEnabled
	m.historyHeight = DefaultHistoryHeight
	m.showHistory = false
	if m.historyEnabled {
		m.historyPanel = components.NewHistoryPanel(m.styles)
		// Register history panel with panel manager
		m.panelManager.RegisterPanel(PanelHistory, m.historyPanel)
	}
	m.config = cfg.Config
	m.configManager = cfg.ConfigManager
	m.initHistory(cfg)
	return m
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.executionMode {
		if cmd := m.detectEnvironmentsCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if cmd := m.checkLatestReleaseCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := m.handlePrimaryUpdate(msg); handled {
		return model, cmd
	}
	if model, cmd, handled := m.handleSecondaryUpdate(msg); handled {
		return model, cmd
	}
	return m.handlePostUpdate(msg)
}

func (m *Model) handlePrimaryUpdate(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model := m.handleWindowSize(msg)
		return model, nil, true
	case PlanStartMsg:
		model, cmd := m.handlePlanStart(msg)
		return model, cmd, true
	case PlanOutputMsg:
		model, cmd := m.handlePlanOutput(msg)
		return model, cmd, true
	case PlanCompleteMsg:
		model, cmd := m.handlePlanComplete(msg)
		return model, cmd, true
	case ApplyStartMsg:
		model, cmd := m.handleApplyStart(msg)
		return model, cmd, true
	case ApplyOutputMsg:
		model, cmd := m.handleApplyOutput(msg)
		return model, cmd, true
	case ApplyCompleteMsg:
		model, cmd := m.handleApplyComplete(msg)
		return model, cmd, true
	default:
		return nil, nil, false
	}
}

func (m *Model) handleSecondaryUpdate(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case RefreshStartMsg:
		model, cmd := m.handleRefreshStart(msg)
		return model, cmd, true
	case RefreshOutputMsg:
		model, cmd := m.handleRefreshOutput(msg)
		return model, cmd, true
	case RefreshCompleteMsg:
		model, cmd := m.handleRefreshComplete(msg)
		return model, cmd, true
	case ValidateCompleteMsg:
		model, cmd := m.handleValidateComplete(msg)
		return model, cmd, true
	case FormatCompleteMsg:
		model, cmd := m.handleFormatComplete(msg)
		return model, cmd, true
	case InitCompleteMsg:
		model, cmd := m.handleInitComplete(msg)
		return model, cmd, true
	case StateListCompleteMsg:
		model, cmd := m.handleStateListComplete(msg)
		return model, cmd, true
	case StateShowCompleteMsg:
		model, cmd := m.handleStateShowComplete(msg)
		return model, cmd, true
	case StateRmCompleteMsg:
		model, cmd := m.handleStateRmComplete(msg)
		return model, cmd, true
	case StateMvCompleteMsg:
		model, cmd := m.handleStateMvComplete(msg)
		return model, cmd, true
	default:
		return m.handleTertiaryUpdate(msg)
	}
}

//nolint:gocyclo,funlen // Message routing requires many cases
func (m *Model) handleTertiaryUpdate(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case HistoryLoadedMsg:
		model := m.handleHistoryLoaded(msg)
		return model, nil, true
	case HistoryDetailMsg:
		model, cmd := m.handleHistoryDetail(msg)
		return model, cmd, true
	case EnvironmentDetectedMsg:
		model, cmd := m.handleEnvironmentDetected(msg)
		return model, cmd, true
	case ClearToastMsg, components.ClearToast:
		model := m.handleClearToast()
		return model, nil, true
	case spinner.TickMsg:
		if m.progressIndicator != nil {
			cmd := m.progressIndicator.Update(msg)
			return m, cmd, true
		}
		return m, nil, true
	case StateMoveCursorBlinkMsg:
		if m.modalState != ModalStateMoveDestination {
			m.stateMoveCursorOn = false
			return m, nil, true
		}
		m.stateMoveCursorOn = !m.stateMoveCursorOn
		m.updateStateMoveDestinationModal()
		cmd := m.stateMoveCursorTickCmd()
		return m, cmd, true
	case components.EnvironmentChangedMsg:
		model, cmd := m.handleEnvironmentChanged(msg)
		return model, cmd, true
	case tea.MouseMsg:
		model, cmd := m.handleMouseMsg(msg)
		return model, cmd, true
	case tea.KeyMsg:
		model, cmd := m.handleKeyMsg(msg)
		return model, cmd, true
	case ErrorMsg:
		model := m.handleErrorMsg(msg)
		return model, nil, true
	case NotificationFailedMsg:
		model := m.handleNotificationFailed(msg)
		return model, nil, true
	case VersionCheckMsg:
		model, cmd := m.handleVersionCheck(msg)
		return model, cmd, true

	// Action request messages from panels
	case RequestPlanMsg:
		cmd := m.beginPlan()
		return m, cmd, true
	case RequestApplyMsg:
		return m.handleRequestApply()
	case RequestRefreshMsg:
		cmd := m.beginRefresh()
		return m, cmd, true
	case RequestValidateMsg:
		cmd := m.beginValidate()
		return m, cmd, true
	case RequestFormatMsg:
		cmd := m.beginFormat()
		return m, cmd, true
	case ToggleFilterMsg:
		m.handleToggleFilter(msg.Action)
		return m, nil, true
	case ToggleStatusMsg:
		m.resourceList.SetShowStatus(!m.resourceList.ShowStatus())
		return m, nil, true
	case ToggleAllGroupsMsg:
		m.resourceList.ToggleAllGroups()
		return m, nil, true
	case ToggleTargetModeMsg:
		m.handleToggleTargetMode()
		return m, nil, true
	case ToggleTargetSelectionMsg:
		return m.handleToggleTargetSelection()
	case ClearTargetSelectionMsg:
		m.handleClearTargetSelection()
		return m, nil, true
	case StateListStartMsg:
		cmd := m.beginStateList()
		return m, cmd, true
	case SwitchResourcesTabMsg:
		return m.handleSwitchResourcesTab(msg.Direction)

	default:
		return nil, nil, false
	}
}

func (m *Model) handleHistoryLoaded(msg HistoryLoadedMsg) tea.Model {
	if msg.Error != nil {
		m.err = msg.Error
		return m
	}
	if m.historyPanel != nil {
		m.historyEntries = msg.Entries
		m.historyPanel.SetEntries(msg.Entries)
		m.syncHistorySelection()
	}
	return m
}

func (m *Model) handleHistoryDetail(msg HistoryDetailMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		cmd := m.toastError(fmt.Sprintf("History error: %v", msg.Error))
		return m, cmd
	}

	m.historyDetail = &msg.Entry
	m.updateHistoryDetailContentWithOperations(msg.Entry, msg.Operations)
	return m, nil
}

func (m *Model) handleEnvironmentDetected(msg EnvironmentDetectedMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		cmd := m.toastError(fmt.Sprintf("Environment detection failed: %v", msg.Error))
		return m, cmd
	}
	m.envDetection = &msg.Result
	strategy, current := applyEnvironmentPreference(msg.Result, msg.Current, msg.Preference)
	m.envStrategy = strategy
	m.envCurrent = current
	m.setEnvironmentOptions(msg.Result, m.envStrategy, m.envCurrent)
	m.updateEnvironmentPanel(msg.Result.Warnings)
	m.loadFilterPreferences()
	m.applyCurrentEnvironment()
	historyReloadCmd := m.reloadHistoryCmd()
	if m.executionMode && m.envCurrent == "" && m.shouldPromptEnvironment() {
		model, focusCmd := m.promptEnvironmentSelection()
		return model, tea.Batch(focusCmd, historyReloadCmd)
	}
	return m, historyReloadCmd
}

func applyEnvironmentPreference(
	result environment.DetectionResult,
	current string,
	pref *environment.Preference,
) (environment.StrategyType, string) {
	strategy := result.Strategy
	if pref == nil {
		return strategy, current
	}
	if pref.Strategy != "" && strategyAvailable(result, pref.Strategy) {
		strategy = pref.Strategy
	}
	if pref.Environment != "" {
		current = pref.Environment
	}
	return strategy, current
}

func (m *Model) updateEnvironmentPanel(_ []string) {
	if m.environmentPanel == nil {
		return
	}
	m.environmentPanel.SetEnvironmentInfo(m.envCurrent, m.envWorkDir, m.envStrategy, m.envOptions)
}

func (m *Model) applyCurrentEnvironment() {
	if m.plan != nil {
		// Preserve preloaded plans during startup environment reconciliation.
		return
	}
	if m.envCurrent == "" {
		return
	}
	if option, ok := m.findEnvironmentOption(m.envCurrent); ok {
		_ = m.applyEnvironmentSelection(option)
	}
}

func (m *Model) promptEnvironmentSelection() (tea.Model, tea.Cmd) {
	if m.panelManager == nil || m.environmentPanel == nil {
		return m, nil
	}
	cmd := m.panelManager.SetFocus(PanelWorkspace)
	m.updateLayout()
	return m, tea.Batch(cmd)
}

func (m *Model) handleClearToast() tea.Model {
	if m.toast != nil {
		m.toast.Hide()
	}
	return m
}

func (m *Model) handleEnvironmentChanged(msg components.EnvironmentChangedMsg) (tea.Model, tea.Cmd) {
	if err := m.applyEnvironmentSelection(msg.Environment); err != nil {
		// Log failed environment switch
		if m.commandLogPanel != nil {
			command := m.buildEnvironmentCommand(msg.Environment)
			m.commandLogPanel.AppendSessionLog("Environment switch failed", command, err.Error())
		}
		cmd := m.toastError(fmt.Sprintf("Failed to switch environment: %v", err))
		return m, cmd
	}
	m.envCurrent = envSelectionValue(msg.Environment)
	if m.environmentPanel != nil {
		m.environmentPanel.SetEnvironmentInfo(m.envCurrent, m.envWorkDir, m.envStrategy, m.envOptions)
	}

	// Log successful environment switch
	if m.commandLogPanel != nil {
		command := m.buildEnvironmentCommand(msg.Environment)
		m.commandLogPanel.AppendSessionLog("Environment switched", command, "Switched to "+m.envDisplayName())
	}

	cmd := m.toastSuccess("Environment changed to " + m.envDisplayName())
	return m, tea.Batch(cmd, m.reloadHistoryCmd())
}

// buildEnvironmentCommand builds a command string for logging environment switches.
func (m *Model) buildEnvironmentCommand(env environment.Environment) string {
	if env.Strategy == environment.StrategyWorkspace {
		return "terraform workspace select " + env.Name
	}
	return "cd " + env.Path
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handled, cmd := m.handlePriorityKey(msg); handled {
		return m, cmd
	}

	if handled, cmd := m.handleNonMainExecutionKey(msg); handled {
		return m, cmd
	}

	if cmd, handled := m.handleMainViewKeybind(msg); handled {
		return m, cmd
	}

	return m, nil
}

func panelSpecForID(layout LayoutSpec, panelID PanelID) (PanelSpec, bool) {
	switch panelID {
	case PanelWorkspace:
		return layout.Workspace, true
	case PanelResources:
		return layout.Resources, true
	case PanelHistory:
		return layout.History, true
	case PanelMain:
		return layout.Main, true
	case PanelCommandLog:
		return layout.CommandLog, true
	default:
		return PanelSpec{}, false
	}
}

func panelContentRow(spec PanelSpec, y int) int {
	return y - spec.Y - 1
}

func panelContentContains(spec PanelSpec, x, y int) bool {
	if spec.Width < 3 || spec.Height < 3 {
		return false
	}
	left := spec.X + 1
	right := spec.X + spec.Width - 2
	top := spec.Y + 1
	bottom := spec.Y + spec.Height - 2
	return x >= left && x <= right && y >= top && y <= bottom
}

func (m *Model) clearMousePressState() {
	m.lastMouseLeftPress = false
	m.lastMouseLeftPressX = 0
	m.lastMouseLeftPressY = 0
}

func mouseIntentFromWheelButton(button tea.MouseButton) mouseIntent {
	if button == tea.MouseButtonWheelUp {
		return mouseIntentWheelUp
	}
	if button == tea.MouseButtonWheelDown {
		return mouseIntentWheelDown
	}
	return mouseIntentNone
}

func (m *Model) resolveLeftClickIntent(event tea.MouseEvent) mouseIntent {
	if event.Action == tea.MouseActionPress {
		m.lastMouseLeftPress = true
		m.lastMouseLeftPressX = event.X
		m.lastMouseLeftPressY = event.Y
		return mouseIntentLeftClick
	}

	if event.Action == tea.MouseActionRelease {
		if m.lastMouseLeftPress && event.X == m.lastMouseLeftPressX && event.Y == m.lastMouseLeftPressY {
			m.clearMousePressState()
			return mouseIntentNone
		}
		m.clearMousePressState()
		return mouseIntentLeftClick
	}

	if event.Action == tea.MouseActionMotion {
		return mouseIntentNone
	}

	return mouseIntentNone
}

func (m *Model) resolveMouseIntent(event tea.MouseEvent) mouseIntent {
	if intent := mouseIntentFromWheelButton(event.Button); intent != mouseIntentNone {
		m.clearMousePressState()
		return intent
	}

	if event.Button != tea.MouseButtonLeft {
		return mouseIntentNone
	}

	return m.resolveLeftClickIntent(event)
}

func (m *Model) mousePanelAt(event tea.MouseEvent) (PanelID, PanelSpec, bool) {
	if m.panelManager == nil {
		return 0, PanelSpec{}, false
	}
	layout := m.panelManager.CalculateLayout(m.width, m.height)
	panelID, ok := m.panelManager.PanelAt(layout, event.X, event.Y)
	if !ok {
		return 0, PanelSpec{}, false
	}
	spec, specOK := panelSpecForID(layout, panelID)
	if !specOK {
		return 0, PanelSpec{}, false
	}
	return panelID, spec, true
}

func (m *Model) focusPanelByMouse(panelID PanelID) tea.Cmd {
	switch panelID {
	case PanelWorkspace:
		return m.handleActionFocusWorkspace(nil)
	case PanelResources:
		return m.handleActionFocusResources(nil)
	case PanelHistory:
		return m.handleActionFocusHistory(nil)
	case PanelMain:
		return m.handleActionFocusMain(nil)
	case PanelCommandLog:
		return m.handleActionFocusCommandLog(nil)
	default:
		return nil
	}
}

func (m *Model) handleMousePanelSelection(panelID PanelID, spec PanelSpec, event tea.MouseEvent) tea.Cmd {
	row := panelContentRow(spec, event.Y)

	switch panelID {
	case PanelResources:
		if m.resourcesActiveTab == 0 {
			if m.resourceList != nil {
				m.resourceList.SelectVisibleRow(row)
			}
			return nil
		}
		if m.stateListContent != nil && m.stateListContent.SelectVisibleRow(row) {
			return m.showSelectedStateDetail()
		}
		return nil
	case PanelHistory:
		if m.historyPanel != nil && m.historyPanel.SelectVisibleRow(row) {
			m.historySelected = m.historyPanel.GetSelectedIndex()
			return m.showSelectedHistoryDetail()
		}
		return nil
	case PanelWorkspace:
		if m.environmentPanel == nil {
			return nil
		}
		selected := m.environmentPanel.SelectVisibleRow(row)
		if selected == nil {
			return nil
		}
		return func() tea.Msg {
			return components.EnvironmentChangedMsg{Environment: *selected}
		}
	case PanelMain:
		if m.mainArea == nil {
			return nil
		}
		m.mainArea.SelectOrToggleDiffTreeAtRow(row)
		return nil
	default:
		return nil
	}
}

func (m *Model) handleMouseWheelWorkspace(wheelUp bool) tea.Cmd {
	if m.environmentPanel == nil {
		return nil
	}
	selected := m.environmentPanel.GetSelectedIndex()
	if wheelUp {
		selected--
	} else {
		selected++
	}
	m.environmentPanel.SetSelectedIndex(selected)
	return nil
}

func (m *Model) handleMouseWheelResources(wheelUp bool) tea.Cmd {
	if m.resourcesActiveTab == 0 {
		if m.resourceList == nil {
			return nil
		}
		if wheelUp {
			m.resourceList.MoveUp()
		} else {
			m.resourceList.MoveDown()
		}
		return nil
	}
	if m.stateListContent == nil {
		return nil
	}
	if wheelUp {
		m.stateListContent.MoveUp()
	} else {
		m.stateListContent.MoveDown()
	}
	return m.showSelectedStateDetail()
}

func (m *Model) handleMouseWheelHistory(wheelUp bool) tea.Cmd {
	if m.historyPanel == nil {
		return nil
	}
	if wheelUp {
		m.historyPanel.MoveUp()
	} else {
		m.historyPanel.MoveDown()
	}
	m.historySelected = m.historyPanel.GetSelectedIndex()
	return m.showSelectedHistoryDetail()
}

func (m *Model) handleMouseWheelMain(wheelUp bool) tea.Cmd {
	if m.mainArea == nil {
		return nil
	}
	keyType := tea.KeyDown
	if wheelUp {
		keyType = tea.KeyUp
	}
	_, cmd := m.mainArea.HandleKey(tea.KeyMsg{Type: keyType})
	return cmd
}

func (m *Model) handleMouseWheelCommandLog(wheelUp bool) tea.Cmd {
	if m.commandLogPanel == nil {
		return nil
	}
	keyType := tea.KeyDown
	if wheelUp {
		keyType = tea.KeyUp
	}
	_, cmd := m.commandLogPanel.HandleKey(tea.KeyMsg{Type: keyType})
	return cmd
}

func (m *Model) handleMouseWheel(panelID PanelID, wheelUp bool) tea.Cmd {
	switch panelID {
	case PanelWorkspace:
		return m.handleMouseWheelWorkspace(wheelUp)
	case PanelResources:
		return m.handleMouseWheelResources(wheelUp)
	case PanelHistory:
		return m.handleMouseWheelHistory(wheelUp)
	case PanelMain:
		return m.handleMouseWheelMain(wheelUp)
	case PanelCommandLog:
		return m.handleMouseWheelCommandLog(wheelUp)
	default:
		return nil
	}
}

func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if !m.ready || m.width <= 0 || m.height <= 0 || m.panelManager == nil {
		return m, nil
	}
	if m.execView != viewMain {
		return m, nil
	}
	if m.modalState != ModalNone {
		return m, nil
	}

	event := tea.MouseEvent(msg)
	intent := m.resolveMouseIntent(event)
	if intent == mouseIntentNone {
		return m, nil
	}
	panelID, spec, hasPanel := m.mousePanelAt(event)

	switch intent {
	case mouseIntentLeftClick:
		if !hasPanel {
			return m, nil
		}
		focusCmd := m.focusPanelByMouse(panelID)
		selectCmd := m.handleMousePanelSelection(panelID, spec, event)
		return m, tea.Batch(focusCmd, selectCmd)
	case mouseIntentWheelUp, mouseIntentWheelDown:
		targetPanel := panelID
		if !hasPanel || !panelContentContains(spec, event.X, event.Y) {
			targetPanel = m.panelManager.GetFocusedPanel()
		}
		wheelCmd := m.handleMouseWheel(targetPanel, intent == mouseIntentWheelUp)
		return m, wheelCmd
	default:
		return m, nil
	}
}

func (m *Model) handlePriorityKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if handled, cmd := m.handleDiagnosticsKey(msg); handled {
		return true, cmd
	}
	if handled, cmd := m.handleEnvironmentPanelKey(msg); handled {
		return true, cmd
	}
	if handled, cmd := m.handleStateMoveDestinationInputKey(msg); handled {
		return true, cmd
	}
	if m.inputCaptured() {
		return true, nil
	}
	return false, nil
}

func (m *Model) handleStateMoveDestinationInputKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.modalState != ModalStateMoveDestination {
		return false, nil
	}

	switch msg.Type {
	case tea.KeyEsc:
		m.modalState = ModalNone
		m.stateMoveCursorOn = false
		if m.helpModal != nil {
			m.helpModal.Hide()
		}
		return true, nil
	case tea.KeyCtrlC:
		m.modalState = ModalNone
		m.stateMoveCursorOn = false
		if m.helpModal != nil {
			m.helpModal.Hide()
		}
		return true, nil
	case tea.KeyEnter:
		destination := strings.TrimSpace(m.stateMoveInput)
		source := strings.TrimSpace(m.stateMoveSource)
		if destination == "" {
			return true, m.toastInfo("Destination address cannot be empty")
		}
		if source == "" {
			m.modalState = ModalNone
			m.stateMoveCursorOn = false
			if m.helpModal != nil {
				m.helpModal.Hide()
			}
			return true, m.toastInfo("Move source cleared. Select source and press m again")
		}
		if destination == source {
			return true, m.toastInfo("Destination must be different from source")
		}
		m.modalState = ModalNone
		m.stateMoveCursorOn = false
		message := "This will move Terraform state to a new address.\n\n" +
			"From:\n  " + source + "\n\nTo:\n  " + destination +
			"\n\nA state backup will be created before move. Continue?"
		m.showConfirmModal("Confirm State Move", message, "Yes, move", m.beginStateMv(source, destination))
		return true, nil
	case tea.KeyBackspace, tea.KeyDelete:
		if m.stateMoveInput != "" {
			runes := []rune(m.stateMoveInput)
			m.stateMoveInput = string(runes[:len(runes)-1])
		}
		m.stateMoveCursorOn = true
		m.updateStateMoveDestinationModal()
		return true, m.stateMoveCursorTickCmd()
	case tea.KeyRunes:
		if len(msg.Runes) > 0 {
			m.stateMoveInput += string(msg.Runes)
			m.stateMoveCursorOn = true
			m.updateStateMoveDestinationModal()
		}
		return true, m.stateMoveCursorTickCmd()
	default:
		return true, nil
	}
}

func (m *Model) handleNonMainExecutionKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.execView == viewMain {
		return false, nil
	}
	if handled, cmd := m.handleExecutionKey(msg); handled {
		return true, cmd
	}
	_, cmd := m.handleNonMainViewKey(msg)
	return true, cmd
}

func (m *Model) handleMainViewKeybind(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.keybindRegistry == nil {
		return nil, false
	}
	ctx := m.buildKeybindContext()
	cmd, handled := m.keybindRegistry.Handle(msg, ctx)
	if !handled {
		return nil, false
	}
	return cmd, true
}

func (m *Model) handleDiagnosticsKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !m.diagnosticsFocused || m.diagnosticsPanel == nil || m.execView != viewMain {
		return false, nil
	}
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return true, tea.Quit
	case keybinds.KeyEsc, "D":
		m.diagnosticsFocused = false
		return true, nil
	default:
		var cmd tea.Cmd
		m.diagnosticsPanel, cmd = m.diagnosticsPanel.Update(msg)
		return true, cmd
	}
}

func (m *Model) handleModalConfirmApplyKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.modalState != ModalConfirmApply {
		return false, nil
	}
	switch msg.String() {
	case "q":
		m.quitting = true
		return true, tea.Quit
	case "y", "Y":
		m.modalState = ModalNone
		return true, m.consumePendingConfirmCmd(m.beginApply)
	case "n", "N", "esc":
		m.modalState = ModalNone
		m.pendingConfirmCmd = nil
		m.clearPendingTargetPlanIntent()
		return true, nil
	case "ctrl+c":
		m.cancelExecution()
		m.modalState = ModalNone
		m.pendingConfirmCmd = nil
		m.clearPendingTargetPlanIntent()
		return true, nil
	default:
		return true, nil
	}
}

func (m *Model) handleEnvironmentPanelKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.panelManager == nil || m.environmentPanel == nil || !m.environmentPanel.SelectorActive() {
		return false, nil
	}
	if m.isOperationRunning() {
		return false, nil
	}
	if handled, panelCmd := m.environmentPanel.HandleKey(msg); handled {
		return true, panelCmd
	}
	return false, nil
}

func (m *Model) handleNonMainViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.execView == viewCommandLog && m.commandLogPanel != nil {
		if diagPanel := m.commandLogPanel.GetDiagnosticsPanel(); diagPanel != nil {
			_, cmd := diagPanel.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) toggleHelpModal() {
	if m.modalState == ModalHelp {
		m.modalState = ModalNone
		return
	}
	m.modalState = ModalHelp
	m.updateHelpModalContent()
}

func (m *Model) showConfirmApplyModal() {
	if m.helpModal == nil {
		return
	}

	// Build the confirmation message with plan summary
	summary := m.planSummaryVerbose()
	message := "Plan summary:\n" + summary
	if m.targetModeEnabled {
		message += "\n\nWarning: -target can produce partial or inconsistent outcomes."
	}
	message += "\n\nDo you want to apply these changes?"
	m.showConfirmModal("Confirm Apply", message, "Yes, apply", m.deferConfirmCommand(m.beginApply))
}

func (m *Model) deferConfirmCommand(factory func() tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		if factory == nil {
			return nil
		}
		cmd := factory()
		if cmd == nil {
			return nil
		}
		return cmd()
	}
}

func (m *Model) showConfirmModal(title, message, yesLabel string, yesCmd tea.Cmd) {
	if m.helpModal == nil {
		return
	}
	actions := []components.ModalAction{
		{Key: "y", Label: yesLabel},
		{Key: "n", Label: "No, cancel"},
	}
	m.helpModal.SetTitle(title)
	m.helpModal.SetConfirm(message, actions)
	m.helpModal.Show()
	m.pendingConfirmCmd = yesCmd
	m.modalState = ModalConfirmApply
}

func (m *Model) showStateMoveDestinationModal(source, initialDestination string) {
	if m.helpModal == nil {
		return
	}
	m.stateMoveSource = source
	m.stateMoveInput = initialDestination
	m.stateMoveCursorOn = true
	m.modalState = ModalStateMoveDestination
	m.updateStateMoveDestinationModal()
	m.helpModal.Show()
}

func (m *Model) updateStateMoveDestinationModal() {
	if m.helpModal == nil {
		return
	}
	inputLine := m.stateMoveInput
	if m.stateMoveCursorOn {
		inputLine += lipgloss.NewStyle().Reverse(true).Render(" ")
	}
	if strings.TrimSpace(inputLine) == "" {
		inputLine = lipgloss.NewStyle().Reverse(true).Render(" ")
	}
	message := "Move source:\n  " + m.stateMoveSource +
		"\n\nDestination address:\n  " + inputLine +
		"\n\nType destination, then press Enter to continue."
	actions := []components.ModalAction{
		{Key: "enter", Label: "Continue"},
		{Key: "esc", Label: "Cancel"},
	}
	m.helpModal.SetTitle("State Move Destination")
	m.helpModal.SetConfirm(message, actions)
}

func (m *Model) stateMoveCursorTickCmd() tea.Cmd {
	return tea.Tick(530*time.Millisecond, func(time.Time) tea.Msg {
		return StateMoveCursorBlinkMsg{}
	})
}

func (m *Model) consumePendingConfirmCmd(fallback func() tea.Cmd) tea.Cmd {
	if m.pendingConfirmCmd == nil {
		if fallback == nil {
			return nil
		}
		return fallback()
	}
	cmd := m.pendingConfirmCmd
	m.pendingConfirmCmd = nil
	return cmd
}

func (m *Model) handleErrorMsg(msg ErrorMsg) tea.Model {
	m.err = msg.Err
	return m
}

func (m *Model) handleNotificationFailed(msg NotificationFailedMsg) tea.Model {
	if msg.Error == nil {
		return m
	}
	summary := "Desktop notification was not sent"
	if msg.Action != "" {
		summary = "Desktop notification for " + msg.Action + " was not sent"
	}
	m.addErrorDiagnostic(summary, msg.Error, "")
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("Desktop notification failed", msg.Action, msg.Error.Error())
	}
	return m
}

func (m *Model) handlePostUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if stateCmd, handled := m.handleStateTabKey(msg); handled {
		cmds = append(cmds, stateCmd)
		return m, tea.Batch(cmds...)
	}

	updated, cmd := m.resourceList.Update(msg)
	if rl, ok := updated.(*components.ResourceList); ok {
		m.resourceList = rl
	}
	cmds = append(cmds, cmd)

	if m.environmentPanel != nil {
		updatedPanel, panelCmd := m.environmentPanel.Update(msg)
		if panel, ok := updatedPanel.(*components.EnvironmentPanel); ok {
			m.environmentPanel = panel
		}
		cmds = append(cmds, panelCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) initHistory(cfg ExecutionConfig) {
	if !cfg.HistoryEnabled {
		return
	}

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
	if store == nil {
		return
	}
	entries, err := m.loadHistoryEntries()
	if err != nil {
		return
	}
	m.historyEntries = entries
	if m.historyPanel != nil {
		m.historyPanel.SetEntries(entries)
	}
	m.syncHistorySelection()
}

func (m *Model) updateHistoryDetailContentWithOperations(entry history.Entry, operations []history.OperationEntry) {
	if m.mainArea == nil {
		return
	}
	title := "Apply details"
	if entry.WorkDir != "" {
		title = "Apply details - " + entry.WorkDir
	}

	// Build metadata for header
	metadata := &utils.LogMetadata{
		Status:      entry.Status,
		StartedAt:   entry.StartedAt,
		FinishedAt:  entry.FinishedAt,
		Duration:    entry.Duration,
		Environment: entry.Environment,
		WorkDir:     entry.WorkDir,
	}

	// Extract plan and apply outputs from operations
	var planOutput, applyOutput string
	for _, op := range operations {
		switch op.Action {
		case "plan":
			if planOutput == "" {
				planOutput = op.Output
			}
		case "apply":
			if applyOutput == "" {
				applyOutput = op.Output
			}
		}
	}

	// Fall back to entry output if no operation outputs
	if applyOutput == "" && planOutput == "" {
		applyOutput = strings.TrimRight(entry.Output, "\n")
	}

	// Format combined output
	var content string
	if planOutput == "" && applyOutput == "" {
		content = "No stored output for this apply."
	} else {
		content = utils.FormatCombinedOutput(metadata, planOutput, applyOutput, 60)
	}

	m.mainArea.SetHistoryContent(title, content)
}

// reloadHistoryCmd returns a command to reload history entries.
func (m *Model) reloadHistoryCmd() tea.Cmd {
	if m.historyStore == nil {
		return nil
	}
	return func() tea.Msg {
		entries, err := m.loadHistoryEntries()
		if err != nil {
			return HistoryLoadedMsg{Error: err}
		}
		return HistoryLoadedMsg{Entries: entries}
	}
}

func (m *Model) handleStateTabKey(msg tea.Msg) (tea.Cmd, bool) {
	if !m.executionMode || m.resourcesActiveTab != 1 || m.stateListContent == nil {
		return nil, false
	}
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, false
	}
	if m.panelManager == nil || m.panelManager.GetFocusedPanel() != PanelResources {
		return nil, false
	}
	handled, stateCmd := m.stateListContent.HandleKey(keyMsg)
	if !handled {
		return nil, false
	}
	return stateCmd, true
}

// View renders the application.
func (m *Model) View() string {
	if immediate := m.viewImmediate(); immediate != "" {
		return immediate
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderMainContent(),
		m.renderStatusBar(),
	)
	return m.applyViewOverlays(view)
}

func (m *Model) viewImmediate() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	if !m.ready {
		return "Loading..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	if view := m.viewExecutionOverride(); view != "" {
		return view
	}
	if m.plan == nil && !m.executionMode {
		return "No plan loaded\n"
	}
	return ""
}

func (m *Model) viewExecutionOverride() string {
	if !m.executionMode {
		return ""
	}
	switch m.execView {
	case viewMain, viewPlanOutput, viewApplyOutput, viewHistoryDetail, viewDiagnostics, viewPlanConfirm,
		viewCommandLog, viewStateList, viewStateShow:
		return ""
	}
	return ""
}

func (m *Model) applyViewOverlays(view string) string {
	if m.modalState == ModalHelp && m.helpModal != nil {
		view = m.helpModal.Overlay(view)
	}
	if m.modalState == ModalConfirmApply && m.helpModal != nil {
		view = m.helpModal.Overlay(view)
	}
	if m.modalState == ModalStateMoveDestination && m.helpModal != nil {
		view = m.helpModal.Overlay(view)
	}
	if m.modalState == ModalTheme && m.themeModal != nil {
		view = m.themeModal.Overlay(view)
	}
	if m.toast != nil && m.toast.IsVisible() {
		view = m.toast.Overlay(view)
	}
	return view
}

// Cleanup performs graceful cleanup of model resources.
func (m *Model) Cleanup() {
	m.cancelExecution()

	if m.historyStore != nil {
		_ = m.historyStore.Close()
		m.historyStore = nil
	}
}

// CleanupTempFiles removes the .lazytf/tmp directory.
func (m *Model) CleanupTempFiles() {
	workDir := m.envWorkDir
	if m.executor != nil {
		workDir = m.executor.WorkDir()
	}
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	tmpDir := filepath.Join(workDir, ".lazytf", "tmp")
	_ = os.RemoveAll(tmpDir)
}
