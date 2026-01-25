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
	rendered := alignLineList(line, detail, m.Width())
	if index == m.Index() {
		rendered = d.styles.Selected.Render(rendered)
	}
	fmt.Fprint(w, rendered)
}

// EnvSelector renders a list of environments with filtering support.
type EnvSelector struct {
	list    list.Model
	baseDir string
	items   []envItem
	filter  string
	current string
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
	model.SetFilteringEnabled(false)
	model.SetShowHelp(false)
	model.SetShowFilter(false)
	model.SetShowTitle(false)
	model.DisableQuitKeybindings()
	model.KeyMap.ShowFullHelp.Unbind()
	return &EnvSelector{
		list: model,
	}
}

// SetShowTitle toggles the list title display.
func (e *EnvSelector) SetShowTitle(show bool) {
	e.list.SetShowTitle(show)
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
	e.current = current
	e.list.SetItems(items)
	e.applyFilter()
}

// Update handles list updates.
func (e *EnvSelector) Update(msg tea.Msg) (*EnvSelector, tea.Cmd) {
	var cmd tea.Cmd
	e.list, cmd = e.list.Update(msg)
	return e, cmd
}

// View renders the selector list.
func (e *EnvSelector) View() string {
	return e.list.View()
}

// FilterText returns the current filter string.
func (e *EnvSelector) FilterText() string {
	return e.filter
}

// HandleFilterKey applies a keypress to the filter input for live filtering.
func (e *EnvSelector) HandleFilterKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if msg.Alt {
		return nil, false
	}
	changed := false
	switch msg.Type {
	case tea.KeyRunes:
		if len(msg.Runes) == 0 {
			return nil, false
		}
		e.filter += string(msg.Runes)
		changed = true
	case tea.KeyBackspace:
		if e.filter == "" {
			return nil, false
		}
		runes := []rune(e.filter)
		e.filter = string(runes[:len(runes)-1])
		changed = true
	case tea.KeyCtrlU:
		if e.filter == "" {
			return nil, false
		}
		e.filter = ""
		changed = true
	default:
		return nil, false
	}

	if !changed {
		return nil, false
	}
	e.applyFilter()
	return nil, true
}

// Filtering reports whether the list is currently filtering.
func (e *EnvSelector) Filtering() bool {
	return e.filter != ""
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
		parts = append(parts, "updated "+formatAge(meta.LastModified))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " · ")
}

func (e *EnvSelector) applyFilter() {
	filter := strings.ToLower(strings.TrimSpace(e.filter))
	filtered := make([]list.Item, 0, len(e.items))
	currentIndex := -1
	for _, item := range e.items {
		if filter != "" && !envItemMatchesFilter(item, filter) {
			continue
		}
		filtered = append(filtered, item)
		if currentIndex == -1 && isCurrentEnv(item.env, e.current) {
			currentIndex = len(filtered) - 1
		}
	}
	e.list.SetItems(filtered)
	if currentIndex >= 0 {
		e.list.Select(currentIndex)
		return
	}
	if len(filtered) > 0 {
		e.list.Select(0)
	}
}

func envItemMatchesFilter(item envItem, query string) bool {
	label := strings.ToLower(item.label)
	if fuzzyMatchEnv(query, label) {
		return true
	}
	if item.detailText == "" {
		return false
	}
	return fuzzyMatchEnv(query, strings.ToLower(item.detailText))
}

func fuzzyMatchEnv(query, candidate string) bool {
	if query == "" {
		return true
	}
	q := []rune(query)
	c := []rune(candidate)
	if len(q) > len(c) {
		return false
	}
	i := 0
	for _, r := range c {
		if r == q[i] {
			i++
			if i == len(q) {
				return true
			}
		}
	}
	return false
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

func alignLineList(left, right string, width int) string {
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
