package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	if m.executionMode && m.autoPlan {
		if cmd := m.beginPlan(); cmd != nil {
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
	case components.EnvironmentChangedMsg:
		model, cmd := m.handleEnvironmentChanged(msg)
		return model, cmd, true
	case tea.KeyMsg:
		model, cmd := m.handleKeyMsg(msg)
		return model, cmd, true
	case ErrorMsg:
		model := m.handleErrorMsg(msg)
		return model, nil, true
	default:
		return nil, nil, false
	}
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
	if m.applyView != nil {
		m.applyView.AppendLine(msg.Line)
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
	m.updateHistoryDetailContent(msg.Entry)
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
		cmd := m.toastError(fmt.Sprintf("Failed to switch environment: %v", err))
		return m, cmd
	}
	m.envCurrent = envSelectionValue(msg.Environment)
	if m.environmentPanel != nil {
		m.environmentPanel.SetEnvironmentInfo(m.envCurrent, m.envWorkDir, m.envStrategy, m.envOptions)
	}
	cmd := m.toastSuccess("Environment changed to " + m.envDisplayName())
	return m, cmd
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handled, cmd := m.handleDiagnosticsKey(msg); handled {
		return m, cmd
	}
	if handled, cmd := m.handleModalSettingsKey(msg); handled {
		return m, cmd
	}
	if handled, cmd := m.handleModalHelpKey(msg); handled {
		return m, cmd
	}
	if handled, cmd := m.handleEnvironmentPanelKey(msg); handled {
		return m, cmd
	}
	if m.inputCaptured() {
		return m, nil
	}
	if panelCmd, handled := m.handlePanelNavigation(msg); handled {
		return m, panelCmd
	}
	if m.executionMode {
		if handled, cmd := m.handleExecutionKey(msg); handled {
			return m, cmd
		}
	}
	if m.execView != viewMain {
		return m.handleNonMainViewKey(msg)
	}
	return m.handleMainViewKey(msg)
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

func (m *Model) handleModalSettingsKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.modalState != ModalSettings {
		return false, nil
	}
	switch msg.String() {
	case "q", consts.KeyCtrlC:
		m.quitting = true
		return true, tea.Quit
	case consts.KeyEsc, ",":
		m.modalState = ModalNone
		return true, nil
	default:
		return true, nil
	}
}

func (m *Model) handleModalHelpKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.modalState != ModalHelp {
		return false, nil
	}
	switch msg.String() {
	case "q", consts.KeyCtrlC:
		m.quitting = true
		return true, tea.Quit
	case "?", consts.KeyEsc:
		m.modalState = ModalNone
		return true, nil
	case "j", consts.KeyDown:
		if m.helpModal != nil {
			m.helpModal.ScrollDown()
		}
		return true, nil
	case "k", "up":
		if m.helpModal != nil {
			m.helpModal.ScrollUp()
		}
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

func (m *Model) handleMainViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", consts.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case consts.KeyEsc, "?":
		m.toggleHelpModal()
		return m, nil
	case ",":
		m.toggleSettingsModal()
		return m, nil
	case "c":
		m.toggleActionFilter(terraform.ActionCreate, &m.filterCreate)
	case "t":
		m.resourceList.ToggleAllGroups()
	case "u":
		m.toggleActionFilter(terraform.ActionUpdate, &m.filterUpdate)
	case "d":
		m.toggleActionFilter(terraform.ActionDelete, &m.filterDelete)
	case "r":
		m.toggleActionFilter(terraform.ActionReplace, &m.filterReplace)
	case "[":
		if cmd := m.switchResourcesTab(-1); cmd != nil {
			return m, cmd
		}
	case "]":
		if cmd := m.switchResourcesTab(1); cmd != nil {
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
}

func (m *Model) toggleActionFilter(action terraform.ActionType, value *bool) {
	*value = !*value
	m.resourceList.SetFilter(action, *value)
	m.saveFilterPreferences()
}

func (m *Model) switchResourcesTab(direction int) tea.Cmd {
	if !m.canSwitchResourcesTab() {
		return nil
	}
	m.resourcesActiveTab = nextResourcesTab(m.resourcesActiveTab, direction)
	return m.loadStateListIfNeeded()
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
	m.historyPanel.SetEntries(entries)
	m.syncHistorySelection()
}

func (m *Model) updateHistoryDetailContent(entry history.Entry) {
	if m.mainArea == nil {
		return
	}
	title := "Apply details"
	if entry.WorkDir != "" {
		title = "Apply details - " + entry.WorkDir
	}
	content := strings.TrimRight(entry.Output, "\n")
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

func (m *Model) handlePanelNavigation(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.panelManager == nil || m.execView != viewMain {
		return nil, false
	}

	if handled, navCmd := m.panelManager.HandleNavigation(msg); handled {
		m.updateLayout()
		m.historyFocused = m.panelManager.GetFocusedPanel() == PanelHistory
		return tea.Batch(navCmd), true
	}

	focusedPanel := m.panelManager.GetFocusedPanel()
	if focusedPanel == PanelCommandLog && msg.String() == consts.KeyEnter {
		m.execView = viewCommandLog
		return nil, true
	}

	if panel, ok := m.panelManager.GetPanel(focusedPanel); ok {
		if handled, panelCmd := panel.HandleKey(msg); handled {
			return panelCmd, true
		}
	}
	return nil, false
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
	if m.modalState == ModalSettings {
		return m.renderSettings()
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
	case viewPlanConfirm:
		if m.planView != nil {
			return m.planView.View()
		}
	case viewCommandLog:
		return m.renderFullScreenCommandLog()
	case viewStateList:
		if m.stateListView != nil {
			return m.stateListView.View()
		}
	case viewStateShow:
		if m.stateShowView != nil {
			return m.stateShowView.View()
		}
	case viewMain, viewPlanOutput, viewApplyOutput, viewHistoryDetail, viewDiagnostics:
		return ""
	}
	return ""
}

func (m *Model) applyViewOverlays(view string) string {
	if m.modalState == ModalHelp && m.helpModal != nil {
		view = m.helpModal.Overlay(view)
	}
	if m.toast != nil && m.toast.IsVisible() {
		view = m.toast.Overlay(view)
	}
	return view
}
