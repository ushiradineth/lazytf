package ui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

const unknownEnvironmentLabel = "unknown"

// Environment-related methods for Model

func (m *Model) envStatusLabel() string {
	label := m.envDisplayName()
	if label == "" {
		label = unknownEnvironmentLabel
	}
	if m.envStrategy != environment.StrategyUnknown {
		return fmt.Sprintf("%s (%s)", label, m.envStrategy)
	}
	return label
}

func (m *Model) detectEnvironmentsCmd() tea.Cmd {
	workDir := defaultWorkDir(m.envWorkDir)
	return func() tea.Msg {
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return EnvironmentDetectedMsg{Error: err}
		}
		pref, err := loadEnvironmentPreference(absWorkDir)
		if err != nil {
			return EnvironmentDetectedMsg{Error: err}
		}
		result, err := detectEnvironments(workDir)
		if err != nil {
			return EnvironmentDetectedMsg{Error: err}
		}
		current := resolveDetectedEnvironment(m, workDir, absWorkDir, result)
		return EnvironmentDetectedMsg{Result: result, Current: current, Preference: pref}
	}
}

func defaultWorkDir(workDir string) string {
	if strings.TrimSpace(workDir) == "" {
		return "."
	}
	return workDir
}

func loadEnvironmentPreference(workDir string) (*environment.Preference, error) {
	pref, err := environment.LoadPreference(workDir)
	if err != nil && !errors.Is(err, environment.ErrNoPreference) {
		return nil, err
	}
	return pref, nil
}

func detectEnvironments(workDir string) (environment.DetectionResult, error) {
	detector, err := newEnvironmentDetector(workDir)
	if err != nil {
		return environment.DetectionResult{}, err
	}
	return detector.Detect(context.Background())
}

func resolveDetectedEnvironment(
	m *Model,
	workDir string,
	absWorkDir string,
	result environment.DetectionResult,
) string {
	current := m.envCurrent
	if current == "" {
		current = matchCurrentFolder(result.FolderPaths, absWorkDir)
	}
	if current == "" && len(result.Workspaces) > 0 {
		if name, err := currentWorkspaceName(workDir); err == nil {
			current = name
		}
	}
	if current == "" && (result.Strategy == environment.StrategyFolder || result.Strategy == environment.StrategyMixed) {
		current = absWorkDir
	}
	return current
}

func matchCurrentFolder(folderPaths []string, absWorkDir string) string {
	for _, folder := range folderPaths {
		if folder == absWorkDir {
			return folder
		}
	}
	return ""
}

func currentWorkspaceName(workDir string) (string, error) {
	manager, err := newWorkspaceManager(workDir)
	if err != nil {
		return "", err
	}
	return manager.Current(context.Background())
}

func (m *Model) setEnvironmentOptions(result environment.DetectionResult, strategy environment.StrategyType, current string) {
	options := make([]environment.Environment, 0, len(result.Environments))
	for _, env := range result.Environments {
		if !strategyMatches(strategy, env.Strategy) {
			continue
		}
		env.IsCurrent = envMatchesCurrent(env, current)
		options = append(options, env)
	}
	m.envOptions = options
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

func (m *Model) applyEnvironmentSelection(option environment.Environment) error {
	if m.planRunning || m.applyRunning || m.refreshRunning {
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

	// Save the environment preference for next startup
	m.saveEnvironmentPreference(option)

	m.setPlan(nil)
	m.planFilePath = ""
	m.planRunFlags = nil
	m.applyRunFlags = nil
	if m.planView != nil {
		m.planView.SetSummary(m.planSummary())
	}
	if m.operationState != nil {
		m.operationState.InitializeFromPlan(nil)
	}
	return nil
}

func (m *Model) saveEnvironmentPreference(option environment.Environment) {
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
	pref := environment.Preference{
		Strategy:    option.Strategy,
		Environment: envSelectionValue(option),
	}
	_ = environment.SavePreference(absWorkDir, pref)
}
