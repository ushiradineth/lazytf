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
	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
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
	executionMode    bool
	executor         *terraform.Executor
	applyView        *views.ApplyView
	planView         *views.PlanView
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
	historyEnabled     bool
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
	toast         *components.Toast
	helpModal     *components.Modal
	themeModal    *components.Modal
	settingsModal *components.Modal

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
	ModalConfirmApply
	ModalTheme
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
	Flags          []string
	WorkDir        string
	EnvName        string
	HistoryStore   *history.Store
	HistoryLogger  *history.Logger
	HistoryEnabled bool
	Config         *config.Config
	ConfigManager  *config.Manager
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
	settingsModal := components.NewModal(appStyles)

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
		configView:    views.NewConfigView(appStyles),

		// Overlay components
		toast:         toast,
		helpModal:     helpModal,
		themeModal:    themeModal,
		settingsModal: settingsModal,

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
		if err := m.diffEngine.CalculateResourceDiffs(plan); err != nil {
			m.err = err
		}
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
	m.executor = cfg.Executor
	m.planFlags = append([]string{}, cfg.Flags...)
	m.applyFlags = append([]string{}, cfg.Flags...)
	m.envWorkDir = cfg.WorkDir
	m.envCurrent = cfg.EnvName
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
	m.configView = views.NewConfigView(m.styles)
	if m.configView != nil {
		m.configView.SetConfig(m.config)
	}
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
	case StateListCompleteMsg:
		model, cmd := m.handleStateListComplete(msg)
		return model, cmd, true
	case StateShowCompleteMsg:
		model, cmd := m.handleStateShowComplete(msg)
		return model, cmd, true
	default:
		return m.handleTertiaryUpdate(msg)
	}
}

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
	case components.EnvironmentChangedMsg:
		model, cmd := m.handleEnvironmentChanged(msg)
		return model, cmd, true
	case tea.KeyMsg:
		model, cmd := m.handleKeyMsg(msg)
		return model, cmd, true
	case ErrorMsg:
		model := m.handleErrorMsg(msg)
		return model, nil, true

	// Action request messages from panels
	case RequestPlanMsg:
		return m, m.beginPlan(), true
	case RequestApplyMsg:
		return m.handleRequestApply()
	case RequestRefreshMsg:
		return m, m.beginRefresh(), true
	case RequestValidateMsg:
		return m, m.beginValidate(), true
	case RequestFormatMsg:
		return m, m.beginFormat(), true
	case ToggleFilterMsg:
		m.handleToggleFilter(msg.Action)
		return m, nil, true
	case ToggleStatusMsg:
		m.resourceList.SetShowStatus(!m.resourceList.ShowStatus())
		return m, nil, true
	case ToggleAllGroupsMsg:
		m.resourceList.ToggleAllGroups()
		return m, nil, true
	case StateListStartMsg:
		return m, m.beginStateList(), true
	case SwitchResourcesTabMsg:
		return m.handleSwitchResourcesTab(msg.Direction)

	default:
		return nil, nil, false
	}
}

// handleRequestApply handles the RequestApplyMsg by showing confirmation modal.
func (m *Model) handleRequestApply() (tea.Model, tea.Cmd, bool) {
	if m.planRunning || m.applyRunning {
		if m.toast != nil {
			return m, m.toast.ShowInfo("Operation already in progress"), true
		}
		return m, nil, true
	}
	if m.plan == nil {
		if m.toast != nil {
			return m, m.toast.ShowError("No plan loaded; run terraform plan first"), true
		}
		return m, nil, true
	}
	m.showConfirmApplyModal()
	return m, nil, true
}

// handleSwitchResourcesTab handles switching the Resources panel tab.
func (m *Model) handleSwitchResourcesTab(direction int) (tea.Model, tea.Cmd, bool) {
	if !m.canSwitchResourcesTab() {
		return m, nil, true
	}
	m.resourcesActiveTab = nextResourcesTab(m.resourcesActiveTab, direction)

	// Sync the controller's active tab
	if m.resourcesController != nil {
		m.resourcesController.SetActiveTab(m.resourcesActiveTab)
	}

	cmd := m.loadStateListIfNeeded()
	return m, cmd, true
}

// handleToggleFilter handles toggling an action filter.
func (m *Model) handleToggleFilter(action terraform.ActionType) {
	switch action {
	case terraform.ActionCreate:
		m.filterCreate = !m.filterCreate
		m.resourceList.SetFilter(action, m.filterCreate)
	case terraform.ActionUpdate:
		m.filterUpdate = !m.filterUpdate
		m.resourceList.SetFilter(action, m.filterUpdate)
	case terraform.ActionDelete:
		m.filterDelete = !m.filterDelete
		m.resourceList.SetFilter(action, m.filterDelete)
	case terraform.ActionReplace:
		m.filterReplace = !m.filterReplace
		m.resourceList.SetFilter(action, m.filterReplace)
	case terraform.ActionNoOp, terraform.ActionRead:
		// These actions don't have filters
		return
	}
	m.saveFilterPreferences()
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) tea.Model {
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
	if m.configView != nil {
		m.configView.SetSize(m.width, m.height)
	}
	return m
}

func (m *Model) handlePlanOutput(msg PlanOutputMsg) (tea.Model, tea.Cmd) {
	if m.applyView != nil {
		m.applyView.AppendLine(msg.Line)
	}
	cmd := m.streamPlanOutputCmd()
	return m, cmd
}

func (m *Model) handleApplyOutput(msg ApplyOutputMsg) (tea.Model, tea.Cmd) {
	// Don't append if apply has already completed (prevents race condition duplicates)
	if !m.applyRunning {
		return m, nil
	}
	if m.applyView != nil {
		m.applyView.AppendLine(msg.Line)
	}
	// Update resource status from apply output
	if m.operationState != nil {
		m.operationState.ParseApplyLine(msg.Line)
		// Refresh resource list to show updated status
		if m.resourceList != nil {
			m.resourceList.Refresh()
		}
	}
	cmd := m.streamApplyOutputCmd()
	return m, cmd
}

func (m *Model) handleRefreshOutput(msg RefreshOutputMsg) (tea.Model, tea.Cmd) {
	if m.applyView != nil {
		m.applyView.AppendLine(msg.Line)
	}
	cmd := m.streamRefreshOutputCmd()
	return m, cmd
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

	if m.executionMode && m.envCurrent == "" && m.shouldPromptEnvironment() {
		return m.promptEnvironmentSelection()
	}
	return m, nil
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

func (m *Model) updateEnvironmentPanel(warnings []string) {
	if m.environmentPanel == nil {
		return
	}
	m.environmentPanel.SetEnvironmentInfo(m.envCurrent, m.envWorkDir, m.envStrategy, m.envOptions)
	m.environmentPanel.SetWarnings(warnings)
}

func (m *Model) applyCurrentEnvironment() {
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
	m.environmentPanel.ActivateSelector()
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
	return m, cmd
}

// buildEnvironmentCommand builds a command string for logging environment switches.
func (m *Model) buildEnvironmentCommand(env environment.Environment) string {
	if env.Strategy == environment.StrategyWorkspace {
		return "terraform workspace select " + env.Name
	}
	return "cd " + env.Path
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 1. Special state handlers that take priority over the keybind registry
	// These handle text input, selector mode, and focused sub-panels
	if handled, cmd := m.handleDiagnosticsKey(msg); handled {
		return m, cmd
	}
	if handled, cmd := m.handleEnvironmentPanelKey(msg); handled {
		return m, cmd
	}
	if m.inputCaptured() {
		return m, nil
	}

	// 2. Handle non-main view keys (state list, command log fullscreen, etc.)
	if m.execView != viewMain {
		if handled, cmd := m.handleExecutionKey(msg); handled {
			return m, cmd
		}
		return m.handleNonMainViewKey(msg)
	}

	// 3. Try the keybind registry for main view keys
	if m.keybindRegistry != nil {
		ctx := m.buildKeybindContext()
		if cmd, handled := m.keybindRegistry.Handle(msg, ctx); handled {
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) handleDiagnosticsKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !m.diagnosticsFocused || m.diagnosticsPanel == nil || m.execView != viewMain {
		return false, nil
	}
	switch msg.String() {
	case "q", consts.KeyCtrlC:
		m.quitting = true
		return true, tea.Quit
	case consts.KeyEsc, "D":
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
		return true, m.beginApply()
	case "n", "N", consts.KeyEsc:
		m.modalState = ModalNone
		return true, nil
	case consts.KeyCtrlC:
		m.cancelExecution()
		m.modalState = ModalNone
		return true, nil
	default:
		return true, nil
	}
}

func (m *Model) handleEnvironmentPanelKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.panelManager == nil || m.environmentPanel == nil || !m.environmentPanel.SelectorActive() {
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

func (m *Model) toggleSettingsModal() {
	if m.modalState == ModalSettings {
		m.modalState = ModalNone
		return
	}
	m.modalState = ModalSettings
	m.updateSettingsModalContent()
}

func (m *Model) showConfirmApplyModal() {
	if m.helpModal == nil {
		return
	}

	// Build the confirmation message with plan summary
	summary := m.planSummaryVerbose()
	message := "Plan summary:\n" + summary + "\n\nDo you want to apply these changes?"

	actions := []components.ModalAction{
		{Key: "y", Label: "Yes, apply"},
		{Key: "n", Label: "No, cancel"},
	}

	m.helpModal.SetTitle("Confirm Apply")
	m.helpModal.SetConfirm(message, actions)
	m.helpModal.Show()
	m.modalState = ModalConfirmApply
}

func (m *Model) canSwitchResourcesTab() bool {
	return m.executionMode && m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources
}

func nextResourcesTab(current, direction int) int {
	if direction < 0 {
		if current > 0 {
			return current - 1
		}
		return 1
	}
	if current < 1 {
		return current + 1
	}
	return 0
}

func (m *Model) loadStateListIfNeeded() tea.Cmd {
	if m.resourcesActiveTab != 1 || m.stateListContent == nil || m.stateListContent.ResourceCount() != 0 {
		return nil
	}
	// Don't set loading state if an operation is already in progress
	if m.planRunning || m.applyRunning || m.refreshRunning {
		return m.beginStateList() // Will show toast
	}
	m.stateListContent.SetLoading(true)
	return m.beginStateList()
}

func (m *Model) handleErrorMsg(msg ErrorMsg) tea.Model {
	m.err = msg.Err
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
	if m.modalState == ModalTheme && m.themeModal != nil {
		view = m.themeModal.Overlay(view)
	}
	if m.modalState == ModalSettings && m.settingsModal != nil {
		view = m.settingsModal.Overlay(view)
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
