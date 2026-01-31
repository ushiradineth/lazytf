package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/terraform/parser"
	"github.com/ushiradineth/lazytf/internal/ui"
)

type workspaceManager interface {
	Switch(ctx context.Context, name string) error
}

type folderManager interface {
	Validate(ctx context.Context, path string) error
}

type teaProgram interface {
	Run() (tea.Model, error)
}

var (
	version             = "0.1.0"
	planFile            string
	mouseEnabled        bool
	readOnlyMode        bool
	tfFlags             string
	workDir             string
	envName             string
	presetName          string
	workspaceName       string
	folderPath          string
	configPath          string
	themeName           string
	noHistory           bool
	programRunner       = runProgram
	executorFactory     = terraform.NewExecutor
	newWorkspaceManager = func(workDir string) (workspaceManager, error) {
		return environment.NewWorkspaceManager(workDir)
	}
	newFolderManager = func(workDir string) (folderManager, error) {
		return environment.NewFolderManager(workDir)
	}
	newProgram = func(model tea.Model, opts ...tea.ProgramOption) teaProgram {
		return tea.NewProgram(model, opts...)
	}
)

func main() {
	if err := runMain(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMain() error {
	rootCmd := newRootCommand()
	return rootCmd.Execute()
}

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "lazytf [plan-file]",
		Short: "A minimal TUI for reviewing Terraform plans",
		Long: `lazytf is a Terminal User Interface for reviewing Terraform plans.
It displays plan changes in a clean, minimal interface inspired by Terraform Cloud,
showing only changed attributes in a git-style diff format.`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		RunE:    run,
	}

	rootCmd.Flags().StringVarP(&planFile, "file", "f", "", "Path to Terraform plan output file")
	mouseEnabled = os.Getenv("TMUX") == ""
	rootCmd.Flags().BoolVar(&mouseEnabled, "mouse", mouseEnabled, "Enable mouse support (disabled by default in tmux)")
	rootCmd.Flags().BoolVar(&readOnlyMode, "read-only", false, "Run in read-only mode (no terraform execution)")
	rootCmd.Flags().StringVar(&tfFlags, "tf-flags", "", "Additional flags to pass to terraform")
	rootCmd.Flags().StringVar(&workDir, "workdir", ".", "Working directory for terraform")
	rootCmd.Flags().StringVar(&envName, "env", "", "Environment name to select")
	rootCmd.Flags().StringVar(&presetName, "preset", "", "Environment preset to apply")
	rootCmd.Flags().StringVar(&workspaceName, "workspace", "", "Terraform workspace to select")
	rootCmd.Flags().StringVar(&folderPath, "folder", "", "Terraform environment folder to use")
	rootCmd.Flags().StringVar(&configPath, "config", "", "Path to config file")
	rootCmd.Flags().StringVar(&themeName, "theme", "", "Theme name to use")
	rootCmd.Flags().BoolVar(&noHistory, "no-history", false, "Disable history logging")
	return rootCmd
}

func run(cmd *cobra.Command, args []string) error {
	configManager, err := config.NewManager(configPath)
	if err != nil {
		return fmt.Errorf("config manager: %w", err)
	}
	cfg, err := configManager.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if themeName != "" {
		cfg.Theme.Name = themeName
	}
	if noHistory {
		cfg.History.Enabled = false
	}
	if workDir == "." && strings.TrimSpace(cfg.Terraform.WorkingDir) != "" {
		workDir = cfg.Terraform.WorkingDir
	}
	var overrideFlags []string
	if override := cfg.ProjectOverrideFor(workDir); override != nil {
		if override.Theme != "" && themeName == "" {
			cfg.Theme.Name = override.Theme
		}
		if override.PresetName != "" && presetName == "" {
			presetName = override.PresetName
		}
		overrideFlags = append(overrideFlags, override.Flags...)
	}
	readOnlyMode = resolveReadOnlyMode(cmd, args)

	if !readOnlyMode {
		return runExecutionMode(&cfg, overrideFlags, configManager)
	}

	return runReadOnlyMode(&cfg, args)
}

func runExecutionMode(cfg *config.Config, overrideFlags []string, configManager *config.Manager) error {
	flags, err := prepareExecutionFlags(cfg, overrideFlags)
	if err != nil {
		return err
	}
	appStyles, err := resolveAppStyles(cfg)
	if err != nil {
		return err
	}
	if err := configureWorkDirAndWorkspace(); err != nil {
		return err
	}
	exec, err := buildExecutor(cfg, flags)
	if err != nil {
		return err
	}
	selectedEnv := resolveSelectedEnv()
	historyStore, historyLogger, err := openHistory(cfg)
	if err != nil {
		return err
	}

	model := ui.NewExecutionModelWithStyles(nil, ui.ExecutionConfig{
		Executor:       exec,
		Flags:          flags,
		WorkDir:        workDir,
		EnvName:        selectedEnv,
		HistoryStore:   historyStore,
		HistoryLogger:  historyLogger,
		HistoryEnabled: cfg.History.Enabled,
		Config:         cfg,
		ConfigManager:  configManager,
	}, appStyles)
	return programRunner(model)
}

func resolveReadOnlyMode(cmd *cobra.Command, args []string) bool {
	if readOnlyMode {
		return true
	}
	explicitReadOnly := flagExplicit(cmd, "read-only")
	return (len(args) > 0 || planFile != "") && !explicitReadOnly
}

func flagExplicit(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	flag := cmd.Flags().Lookup(name)
	return flag != nil && cmd.Flags().Changed(name)
}

func prepareExecutionFlags(cfg *config.Config, overrideFlags []string) ([]string, error) {
	flags := append([]string{}, cfg.Terraform.DefaultFlags...)
	flags = append(flags, splitFlags(tfFlags)...)
	if len(overrideFlags) > 0 {
		flags = append(flags, overrideFlags...)
	}
	flags, err := applyPreset(cfg, flags)
	if err != nil {
		return nil, err
	}
	return stripFlag(flags, "-json"), nil
}

func resolveAppStyles(cfg *config.Config) (*styles.Styles, error) {
	appTheme, err := styles.ResolveTheme(cfg.Theme.Name)
	if err != nil {
		return nil, err
	}
	return styles.NewStyles(appTheme), nil
}

func configureWorkDirAndWorkspace() error {
	if workspaceName != "" && folderPath != "" {
		return errors.New("cannot use --workspace and --folder together")
	}
	if folderPath != "" {
		resolved, err := resolveFolderSelection(workDir, folderPath)
		if err != nil {
			return err
		}
		workDir = resolved
	}
	if workspaceName == "" {
		return nil
	}
	manager, err := newWorkspaceManager(workDir)
	if err != nil {
		return fmt.Errorf("failed to initialize workspace manager: %w", err)
	}
	if err := manager.Switch(context.Background(), workspaceName); err != nil {
		return fmt.Errorf("failed to select workspace %s: %w", workspaceName, err)
	}
	return nil
}

func buildExecutor(cfg *config.Config, flags []string) (*terraform.Executor, error) {
	var execOpts []terraform.ExecutorOption
	execOpts = append(execOpts, terraform.WithDefaultFlags(flags))
	if strings.TrimSpace(cfg.Terraform.Binary) != "" {
		execOpts = append(execOpts, terraform.WithTerraformPath(cfg.Terraform.Binary))
	}
	if cfg.Terraform.Timeout > 0 {
		execOpts = append(execOpts, terraform.WithTimeout(cfg.Terraform.Timeout))
	}
	exec, err := executorFactory(workDir, execOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize terraform: %w", err)
	}
	return exec, nil
}

func resolveSelectedEnv() string {
	if workspaceName != "" {
		return workspaceName
	}
	if folderPath != "" {
		return filepath.Base(workDir)
	}
	return envName
}

func openHistory(cfg *config.Config) (*history.Store, *history.Logger, error) {
	if !cfg.History.Enabled {
		return nil, nil, nil
	}
	opts := []history.StoreOption{
		history.WithCompressionThreshold(cfg.History.CompressionThreshold),
	}
	var (
		historyStore  *history.Store
		historyLogger *history.Logger
		err           error
	)
	if strings.TrimSpace(cfg.History.Path) != "" {
		historyStore, err = history.Open(cfg.History.Path, opts...)
	} else {
		historyStore, err = history.OpenDefault(opts...)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("history store: %w", err)
	}
	historyLogger = history.NewLogger(historyStore, history.Level(cfg.History.Level))
	return historyStore, historyLogger, nil
}

func applyPreset(cfg *config.Config, flags []string) ([]string, error) {
	if presetName == "" {
		return flags, nil
	}

	preset, ok := cfg.PresetByName(presetName)
	if !ok {
		return nil, fmt.Errorf("preset not found: %s", presetName)
	}
	if preset.WorkDir != "" {
		workDir = preset.WorkDir
	}
	if len(preset.Flags) > 0 {
		flags = append(flags, preset.Flags...)
	}
	if preset.Theme != "" && themeName == "" {
		cfg.Theme.Name = preset.Theme
	}
	if envName == "" && preset.Environment != "" {
		envName = preset.Environment
	}
	return flags, nil
}

func runReadOnlyMode(cfg *config.Config, args []string) error {
	// Determine plan file path.
	var planPath string
	switch {
	case len(args) > 0:
		planPath = args[0]
	case planFile != "":
		planPath = planFile
	default:
		return errors.New("no plan file specified. Usage: lazytf <plan-file> or lazytf --file <plan-file>")
	}

	// Parse the plan output.
	parser := parser.NewTextParser()
	plan, err := parser.ParseFile(planPath)
	if err != nil {
		return fmt.Errorf("failed to parse plan file: %w", err)
	}

	// Create and run the TUI.
	appTheme, err := styles.ResolveTheme(cfg.Theme.Name)
	if err != nil {
		return err
	}
	appStyles := styles.NewStyles(appTheme)
	model := ui.NewModelWithStyles(plan, appStyles)
	return programRunner(model)
}

func runProgram(model tea.Model) error {
	options := []tea.ProgramOption{
		tea.WithAltScreen(),
	}
	if mouseEnabled {
		options = append(options, tea.WithMouseCellMotion())
	}

	p := newProgram(model, options...)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

func splitFlags(flags string) []string {
	if strings.TrimSpace(flags) == "" {
		return nil
	}
	var args []string
	var buf strings.Builder
	inSingle := false
	inDouble := false

	flush := func() {
		if buf.Len() > 0 {
			args = append(args, buf.String())
			buf.Reset()
		}
	}

	for _, r := range flags {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
				continue
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
				continue
			}
		case ' ', '\t', '\n':
			if !inSingle && !inDouble {
				flush()
				continue
			}
		}
		buf.WriteRune(r)
	}
	flush()
	return args
}

//nolint:unparam // target kept as parameter for flexibility
func stripFlag(flags []string, target string) []string {
	if len(flags) == 0 {
		return flags
	}
	filtered := make([]string, 0, len(flags))
	for _, flag := range flags {
		if flag == target {
			continue
		}
		filtered = append(filtered, flag)
	}
	return filtered
}

func resolveFolderSelection(baseDir, folder string) (string, error) {
	if strings.TrimSpace(folder) == "" {
		return baseDir, nil
	}
	// Prevent path traversal attacks
	if strings.Contains(folder, "..") {
		return "", fmt.Errorf("path traversal detected in folder path: %s", folder)
	}
	manager, err := newFolderManager(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to initialize folder manager: %w", err)
	}
	if err := manager.Validate(context.Background(), folder); err != nil {
		return "", fmt.Errorf("invalid folder %s: %w", folder, err)
	}
	var resolved string
	if filepath.IsAbs(folder) {
		resolved, err = filepath.Abs(folder)
	} else {
		resolved, err = filepath.Abs(filepath.Join(baseDir, folder))
	}
	if err != nil {
		return "", err
	}
	// Verify resolved path is within baseDir (unless folder was absolute)
	if !filepath.IsAbs(folder) {
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			return "", err
		}
		relPath, err := filepath.Rel(absBase, resolved)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return "", fmt.Errorf("resolved path %s is outside base directory %s", resolved, absBase)
		}
	}
	return resolved, nil
}
