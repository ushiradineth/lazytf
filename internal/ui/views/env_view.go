package views

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/environment"
	"github.com/ushiradineth/tftui/internal/styles"
	"github.com/ushiradineth/tftui/internal/ui/components"
)

// EnvViewMode selects which list is displayed.
type EnvViewMode int

const (
	EnvViewStrategy EnvViewMode = iota
	EnvViewEnvironments
)

// StrategyOption describes a selectable environment strategy.
type StrategyOption struct {
	Label    string
	Detail   string
	Strategy environment.StrategyType
}

type strategyItem struct {
	option StrategyOption
}

func (s strategyItem) FilterValue() string {
	return s.option.Label
}

type strategyDelegate struct {
	styles *styles.Styles
}

func (d strategyDelegate) Height() int {
	return 1
}

func (d strategyDelegate) Spacing() int {
	return 0
}

func (d strategyDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d strategyDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	strategy, ok := item.(strategyItem)
	if !ok {
		return
	}
	line := strategy.option.Label
	if strategy.option.Detail != "" {
		line = alignLine(line, d.styles.Dimmed.Render(strategy.option.Detail), m.Width())
	}
	if index == m.Index() {
		line = d.styles.Selected.Render(line)
	}
	fmt.Fprint(w, line)
}

// EnvView renders the environment selection modal.
type EnvView struct {
	styles       *styles.Styles
	width        int
	height       int
	mode         EnvViewMode
	strategyList list.Model
	envSelector  *components.EnvSelector
	warnings     []string
}

// NewEnvView creates a new environment selection view.
func NewEnvView(s *styles.Styles) *EnvView {
	if s == nil {
		s = styles.DefaultStyles()
	}
	strategyList := list.New(nil, strategyDelegate{styles: s}, 0, 0)
	strategyList.Title = "Select Environment Strategy"
	strategyList.SetShowStatusBar(false)
	strategyList.SetShowPagination(false)
	strategyList.SetFilteringEnabled(false)
	strategyList.DisableQuitKeybindings()
	strategyList.KeyMap.ShowFullHelp.Unbind()
	return &EnvView{
		styles:       s,
		mode:         EnvViewEnvironments,
		strategyList: strategyList,
		envSelector:  components.NewEnvSelector(s),
	}
}

// SetSize updates the modal layout size.
func (v *EnvView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetMode updates which list is displayed.
func (v *EnvView) SetMode(mode EnvViewMode) {
	v.mode = mode
}

// Mode returns the current view mode.
func (v *EnvView) Mode() EnvViewMode {
	return v.mode
}

// SetWarnings sets warnings to display in the modal.
func (v *EnvView) SetWarnings(warnings []string) {
	v.warnings = warnings
}

// SetStrategies sets selectable strategy options.
func (v *EnvView) SetStrategies(options []StrategyOption) {
	items := make([]list.Item, 0, len(options))
	for _, option := range options {
		items = append(items, strategyItem{option: option})
	}
	v.strategyList.SetItems(items)
}

// SetEnvironments updates the environment list.
func (v *EnvView) SetEnvironments(envs []environment.Environment, strategy environment.EnvironmentStrategy, current, baseDir string) {
	if v.envSelector == nil {
		return
	}
	v.envSelector.SetBaseDir(baseDir)
	v.envSelector.SetStrategy(strategy)
	v.envSelector.SetEnvironments(envs, current)
}

// Update handles list input.
func (v *EnvView) Update(msg tea.Msg) (*EnvView, tea.Cmd) {
	var cmd tea.Cmd
	switch v.mode {
	case EnvViewStrategy:
		v.strategyList, cmd = v.strategyList.Update(msg)
	default:
		v.envSelector, cmd = v.envSelector.Update(msg)
	}
	return v, cmd
}

// Filtering reports whether the current list is filtering.
func (v *EnvView) Filtering() bool {
	if v.mode == EnvViewEnvironments && v.envSelector != nil {
		return v.envSelector.Filtering()
	}
	return false
}

// SelectedEnvironment returns the selected environment.
func (v *EnvView) SelectedEnvironment() *environment.Environment {
	if v.envSelector == nil {
		return nil
	}
	return v.envSelector.SelectedEnvironment()
}

// SelectedStrategy returns the selected strategy.
func (v *EnvView) SelectedStrategy() environment.StrategyType {
	item, ok := v.strategyList.SelectedItem().(strategyItem)
	if !ok {
		return environment.StrategyUnknown
	}
	return item.option.Strategy
}

// View renders the modal.
func (v *EnvView) View() string {
	if v.styles == nil {
		return ""
	}
	width := minInt(74, v.width-4)
	if width < 32 {
		width = 32
	}
	contentWidth := width - 2
	contentHeight := maxInt(8, minInt(v.height-6, 18))

	switch v.mode {
	case EnvViewStrategy:
		v.strategyList.SetSize(contentWidth, contentHeight)
	default:
		if v.envSelector != nil {
			v.envSelector.SetSize(contentWidth, contentHeight)
		}
	}

	var lines []string
	if v.mode == EnvViewStrategy {
		lines = append(lines, v.strategyList.View())
	} else if v.envSelector != nil {
		lines = append(lines, v.envSelector.View())
	}
	if len(v.warnings) > 0 {
		lines = append(lines, "")
		lines = append(lines, v.styles.Dimmed.Render("Warnings:"))
		for _, warning := range v.warnings {
			lines = append(lines, v.styles.Dimmed.Render("  "+warning))
		}
	}
	lines = append(lines, "")
	if v.mode == EnvViewStrategy {
		lines = append(lines, "enter: choose | esc: back")
	} else {
		lines = append(lines, "enter: select | /: filter | esc: back")
	}
	content := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	box := v.styles.Border.Width(width).Render(content)
	if v.width == 0 || v.height == 0 {
		return box
	}
	return lipgloss.Place(v.width, v.height, lipgloss.Center, lipgloss.Center, box)
}

func alignLine(left, right string, width int) string {
	if width <= 0 || right == "" {
		return left
	}
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
