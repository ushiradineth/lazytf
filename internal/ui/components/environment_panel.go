package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/styles"
)

// EnvironmentChangedMsg is sent when the user selects a new environment
type EnvironmentChangedMsg struct {
	Environment environment.Environment
}

// EnvironmentPanel displays current workspace/folder with inline selector
type EnvironmentPanel struct {
	styles       *styles.Styles
	width        int
	height       int
	focused      bool
	selectorMode bool // true when 'e' is pressed and selector is active
	infoList     *KeyValueList

	// Environment state
	current      string
	workDir      string
	strategy     environment.StrategyType
	environments []environment.Environment
	warnings     []string

	// Selector state
	envSelector *EnvSelector
}

// NewEnvironmentPanel creates a new environment panel
func NewEnvironmentPanel(s *styles.Styles) *EnvironmentPanel {
	if s == nil {
		s = styles.DefaultStyles()
	}
	envSelector := NewEnvSelector(s)
	envSelector.SetShowTitle(false)
	return &EnvironmentPanel{
		styles:       s,
		selectorMode: false,
		infoList:     NewKeyValueList(),
		envSelector:  envSelector,
	}
}

// SetSize updates the panel dimensions
func (e *EnvironmentPanel) SetSize(width, height int) {
	e.width = width
	e.height = height
}

// SetFocused sets the focus state
func (e *EnvironmentPanel) SetFocused(focused bool) {
	e.focused = focused
	if !focused {
		e.selectorMode = false
	}
}

// IsFocused returns whether the panel is focused
func (e *EnvironmentPanel) IsFocused() bool {
	return e.focused
}

// SelectorActive reports whether the environment selector is active.
func (e *EnvironmentPanel) SelectorActive() bool {
	return e.selectorMode
}

// SetEnvironmentInfo updates the environment information
func (e *EnvironmentPanel) SetEnvironmentInfo(current, workDir string, strategy environment.StrategyType, environments []environment.Environment) {
	e.current = current
	e.workDir = workDir
	e.strategy = strategy
	e.environments = environments
	e.refreshEnvSelector()
}

// SetWarnings updates warnings shown in selector mode.
func (e *EnvironmentPanel) SetWarnings(warnings []string) {
	e.warnings = warnings
}

// Filtering reports whether selector filtering is active.
func (e *EnvironmentPanel) Filtering() bool {
	if !e.selectorMode || e.envSelector == nil {
		return false
	}
	return e.envSelector.Filtering()
}

// ActivateSelector enters selector mode.
func (e *EnvironmentPanel) ActivateSelector() {
	e.selectorMode = true
}

// Update handles Bubble Tea messages (implements Panel interface)
func (e *EnvironmentPanel) Update(msg tea.Msg) (any, tea.Cmd) {
	if !e.selectorMode {
		return e, nil
	}
	if _, ok := msg.(tea.KeyMsg); ok {
		return e, nil
	}
	if e.envSelector == nil {
		return e, nil
	}
	updated, cmd := e.envSelector.Update(msg)
	e.envSelector = updated
	return e, cmd
}

// HandleKey handles key events
func (e *EnvironmentPanel) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	if !e.focused {
		return false, nil
	}

	// Toggle selector mode
	if msg.String() == "e" && !e.selectorMode {
		e.ActivateSelector()
		return true, nil
	}

	// Handle selector navigation
	if e.selectorMode {
		if e.envSelector == nil {
			return true, nil
		}
		switch msg.String() {
		case "enter":
			selected := e.envSelector.SelectedEnvironment()
			if selected == nil {
				return true, nil
			}
			e.selectorMode = false
			return true, func() tea.Msg {
				return EnvironmentChangedMsg{Environment: *selected}
			}
		case "esc":
			e.selectorMode = false
			return true, nil
		}
		if cmd, handled := e.envSelector.HandleFilterKey(msg); handled {
			return true, cmd
		}
		updated, cmd := e.envSelector.Update(msg)
		e.envSelector = updated
		return true, cmd
	}

	return false, nil
}

// View renders the panel
func (e *EnvironmentPanel) View() string {
	if e.styles == nil || e.height <= 0 {
		return ""
	}

	// Determine border style based on focus
	borderStyle := e.styles.Border
	titleStyle := e.styles.PanelTitle
	if e.focused {
		borderStyle = e.styles.FocusedBorder
		titleStyle = e.styles.FocusedPanelTitle
	}

	var content string
	if e.selectorMode {
		content = e.renderSelector()
	} else {
		content = e.renderCurrentEnvironment()
	}

	// Title
	titleText := "[1] Workspace"
	if e.selectorMode {
		titleText = "[1] Select Environment"
	}

	// Build panel with border and title
	panel := borderStyle.
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		BorderTopForeground(borderStyle.GetBorderTopForeground()).
		Width(e.width - 2).
		Height(e.height - 2).
		Render(content)

	titleRendered := titleStyle.Render(" " + titleText + " ")

	// Split panel and replace top border with title-inserted version
	lines := strings.Split(panel, "\n")
	if len(lines) > 0 && e.width > 4 {
		if line, ok := RenderPanelTitleLine(e.width, borderStyle, titleRendered); ok {
			lines[0] = line
		}
	}

	return strings.Join(lines, "\n")
}

// renderCurrentEnvironment renders the current environment display
func (e *EnvironmentPanel) renderCurrentEnvironment() string {
	maxLines := e.height - 2
	lines := make([]string, 0, maxLines)
	contentWidth := e.contentWidth()

	// Current environment
	currentLabel := e.current
	if currentLabel == "" {
		currentLabel = "default"
	}

	rows := []KeyValueRow{
		{
			Label:      "Current: ",
			Value:      currentLabel,
			LabelStyle: e.styles.Bold,
		},
	}

	// Strategy
	strategyLabel := string(e.strategy)
	switch e.strategy {
	case environment.StrategyWorkspace:
		strategyLabel = "Workspace"
	case environment.StrategyFolder:
		strategyLabel = "Folder"
	case environment.StrategyMixed:
		strategyLabel = "Mixed"
	}
	rows = append(rows, KeyValueRow{
		Label:      "Strategy: ",
		Value:      strategyLabel,
		LabelStyle: e.styles.Dimmed,
		ValueStyle: e.styles.Dimmed,
	})

	// Environment count
	envCount := fmt.Sprintf("%d available", len(e.environments))
	rows = append(rows, KeyValueRow{
		Value:      envCount,
		LabelStyle: e.styles.Dimmed,
		ValueStyle: e.styles.Dimmed,
	})

	if e.infoList == nil {
		e.infoList = NewKeyValueList()
	}
	e.infoList.SetWidth(contentWidth)
	e.infoList.SetRows(rows)
	lines = append(lines, strings.Split(e.infoList.View(), "\n")...)

	// Instruction
	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// renderSelector renders the environment selector
func (e *EnvironmentPanel) renderSelector() string {
	maxLines := e.height - 2
	if maxLines < 1 {
		maxLines = 1
	}
	lines := make([]string, 0, maxLines)
	contentWidth := e.contentWidth()

	var warnLines []string
	if len(e.warnings) > 0 && maxLines >= 3 {
		warnLines = append(warnLines, e.styles.Dimmed.Render("Warnings:"))
		warnLines = append(warnLines, e.styles.Dimmed.Render("  "+truncateVisible(e.warnings[0], contentWidth-2)))
	}

	helpLine := "type: filter | enter: select | esc: back"

	filterText := ""
	if e.envSelector != nil {
		filterText = e.envSelector.FilterText()
	}
	filterLine := e.styles.Dimmed.Render("Filter: " + filterText)

	listHeight := maxLines - len(warnLines)
	if filterLine != "" {
		listHeight--
	}
	if helpLine != "" {
		listHeight--
	}
	if listHeight < 1 {
		listHeight = 1
	}

	if filterLine != "" {
		lines = append(lines, filterLine)
	}

	if e.envSelector != nil {
		e.envSelector.SetSize(contentWidth, listHeight)
		lines = appendLines(lines, e.envSelector.View())
	}

	lines = append(lines, warnLines...)
	if helpLine != "" {
		lines = append(lines, e.styles.Help.Render(helpLine))
	}

	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (e *EnvironmentPanel) contentWidth() int {
	width := e.width - 4
	if width < 1 {
		return 1
	}
	return width
}

func (e *EnvironmentPanel) refreshEnvSelector() {
	if e.envSelector == nil {
		return
	}
	baseDir := e.workDir
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	e.envSelector.SetBaseDir(baseDir)
	e.envSelector.SetStrategy(e.strategy)
	e.envSelector.SetEnvironments(e.environments, e.current)
}

func truncateVisible(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(text) <= maxWidth {
		return text
	}
	return runewidth.Truncate(text, maxWidth, "...")
}

func appendLines(lines []string, block string) []string {
	if block == "" {
		return append(lines, "")
	}
	return append(lines, strings.Split(block, "\n")...)
}
