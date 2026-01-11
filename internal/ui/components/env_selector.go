package components

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/styles"
)

type envItem struct {
	env        environment.Environment
	label      string
	detailText string
}

func (e envItem) FilterValue() string {
	return e.label
}

type envItemDelegate struct {
	styles *styles.Styles
}

func (d envItemDelegate) Height() int {
	return 1
}

func (d envItemDelegate) Spacing() int {
	return 0
}

func (d envItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d envItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	env, ok := item.(envItem)
	if !ok {
		return
	}
	marker := " "
	if env.env.IsCurrent {
		marker = "*"
	}
	line := fmt.Sprintf("%s %s", marker, env.label)
	detail := env.detailText
	if detail != "" {
		detail = d.styles.Dimmed.Render(detail)
	}
	rendered := alignLine(line, detail, m.Width())
	if index == m.Index() {
		rendered = d.styles.Selected.Render(rendered)
	}
	fmt.Fprint(w, rendered)
}

// EnvSelector renders a list of environments with filtering support.
type EnvSelector struct {
	list      list.Model
	styles    *styles.Styles
	baseDir   string
	strategy  environment.EnvironmentStrategy
	items     []envItem
	filterSeq int
}

// NewEnvSelector creates a new environment selector list.
func NewEnvSelector(s *styles.Styles) *EnvSelector {
	if s == nil {
		s = styles.DefaultStyles()
	}
	delegate := envItemDelegate{styles: s}
	model := list.New(nil, delegate, 0, 0)
	model.Title = "Select Environment"
	model.SetShowStatusBar(false)
	model.SetShowPagination(false)
	model.SetFilteringEnabled(true)
	model.SetShowHelp(false)
	model.SetShowFilter(true)
	model.DisableQuitKeybindings()
	model.KeyMap.ShowFullHelp.Unbind()
	return &EnvSelector{
		list:   model,
		styles: s,
	}
}

// SetBaseDir configures the directory used for relative labels.
func (e *EnvSelector) SetBaseDir(baseDir string) {
	e.baseDir = baseDir
}

// SetSize updates the list layout size.
func (e *EnvSelector) SetSize(width, height int) {
	e.list.SetSize(width, height)
}

// SetStrategy updates the list title based on strategy.
func (e *EnvSelector) SetStrategy(strategy environment.EnvironmentStrategy) {
	e.strategy = strategy
	title := "Select Environment"
	switch strategy {
	case environment.StrategyWorkspace:
		title = "Select Environment (Workspaces)"
	case environment.StrategyFolder:
		title = "Select Environment (Folders)"
	case environment.StrategyMixed:
		title = "Select Environment (Mixed)"
	}
	e.list.Title = title
}

// SetEnvironments updates the list items.
func (e *EnvSelector) SetEnvironments(envs []environment.Environment, current string) {
	items := make([]list.Item, 0, len(envs))
	mapped := make([]envItem, 0, len(envs))
	currentIndex := -1
	for _, env := range envs {
		item := envItem{
			env:        env,
			label:      e.envLabel(env),
			detailText: formatMetadata(env.Metadata),
		}
		if isCurrentEnv(env, current) {
			item.env.IsCurrent = true
			if currentIndex == -1 {
				currentIndex = len(items)
			}
		}
		items = append(items, item)
		mapped = append(mapped, item)
	}
	e.items = mapped
	e.list.SetItems(items)
	if currentIndex >= 0 {
		e.list.Select(currentIndex)
	}
}

// Update handles list updates.
func (e *EnvSelector) Update(msg tea.Msg) (*EnvSelector, tea.Cmd) {
	if debounce, ok := msg.(envFilterDebounceMsg); ok {
		return e.handleDebounce(debounce)
	}
	prevFilter := e.list.FilterInput.Value()
	var cmd tea.Cmd
	e.list, cmd = e.list.Update(msg)
	e.queueDebounce(prevFilter, &cmd)
	return e, cmd
}

// View renders the selector list.
func (e *EnvSelector) View() string {
	return e.list.View()
}

// Filtering reports whether the list is currently filtering.
func (e *EnvSelector) Filtering() bool {
	return e.list.FilterState() == list.Filtering
}

// SelectedEnvironment returns the selected environment.
func (e *EnvSelector) SelectedEnvironment() *environment.Environment {
	item, ok := e.list.SelectedItem().(envItem)
	if !ok {
		return nil
	}
	selected := item.env
	return &selected
}

type envFilterDebounceMsg struct {
	seq   int
	value string
}

func (e *EnvSelector) handleDebounce(msg envFilterDebounceMsg) (*EnvSelector, tea.Cmd) {
	if msg.seq != e.filterSeq {
		return e, nil
	}
	if e.list.FilterState() != list.Filtering {
		return e, nil
	}
	if e.list.FilterInput.Value() != msg.value {
		return e, nil
	}
	return e, nil
}

func (e *EnvSelector) queueDebounce(prevFilter string, cmd *tea.Cmd) {
	if e.list.FilterState() != list.Filtering {
		return
	}
	current := e.list.FilterInput.Value()
	if current == prevFilter {
		return
	}
	e.filterSeq++
	seq := e.filterSeq
	value := current
	debounce := tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return envFilterDebounceMsg{seq: seq, value: value}
	})
	if cmd == nil {
		*cmd = debounce
		return
	}
	*cmd = tea.Batch(*cmd, debounce)
}

func (e *EnvSelector) envLabel(env environment.Environment) string {
	if env.Strategy == environment.StrategyFolder {
		if strings.TrimSpace(e.baseDir) != "" {
			if rel, err := filepath.Rel(e.baseDir, env.Path); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				return rel
			}
		}
		return filepath.Base(env.Path)
	}
	return env.Name
}

func isCurrentEnv(env environment.Environment, current string) bool {
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

func formatMetadata(meta environment.EnvironmentMetadata) string {
	parts := []string{}
	if meta.ResourceCount > 0 {
		parts = append(parts, fmt.Sprintf("%d resources", meta.ResourceCount))
	}
	if meta.HasState {
		parts = append(parts, "state")
	}
	if !meta.LastModified.IsZero() {
		parts = append(parts, fmt.Sprintf("updated %s", formatAge(meta.LastModified)))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " · ")
}

func formatAge(t time.Time) string {
	delta := time.Since(t)
	if delta < time.Minute {
		return "just now"
	}
	if delta < time.Hour {
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	}
	if delta < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	}
	if delta < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(delta.Hours()/24))
	}
	return t.Format("2006-01-02")
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
