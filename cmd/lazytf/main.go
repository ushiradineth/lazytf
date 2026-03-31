package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/notifications"
	"github.com/ushiradineth/lazytf/internal/profile"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/terraform/parser"
	"github.com/ushiradineth/lazytf/internal/ui"
)

type workspaceManager interface {
	Current(ctx context.Context) (string, error)
	Switch(ctx context.Context, name string) error
}

type folderManager interface {
	Validate(ctx context.Context, path string) error
}

type teaProgram interface {
	Run() (tea.Model, error)
	Send(msg tea.Msg)
}

var (
	planFile            string
	readOnly            bool
	mouseEnabled        bool
	tfFlags             string
	workDir             string
	envName             string
	presetName          string
	workspaceName       string
	folderPath          string
	configPath          string
	themeName           string
	noHistory           bool
	profileFlags        string
	programRunner       = runProgram
	executionModeRunner = runProgramWithCleanup
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
		Use:   "lazytf",
		Short: "A minimal TUI for reviewing Terraform plans",
		Long: `lazytf is a Terminal User Interface for reviewing Terraform plans.
It displays plan changes in a clean, minimal interface inspired by Terraform Cloud,
showing only changed attributes in a git-style diff format.`,
		Version: consts.Version,
		Args:    cobra.NoArgs,
		RunE:    run,
	}

	rootCmd.Flags().StringVarP(&planFile, "plan", "p", "", "Path to Terraform plan file, or '-' to read plan text from stdin")
	rootCmd.Flags().BoolVar(&readOnly, "readonly", false, "Open plan in read-only mode (requires --plan)")
	mouseEnabled = os.Getenv("TMUX") == ""
	rootCmd.Flags().BoolVar(&mouseEnabled, "mouse", mouseEnabled, "Enable mouse support (disabled by default in tmux)")
	rootCmd.Flags().StringVar(&tfFlags, "tf-flags", "", "Additional flags to pass to terraform")
	rootCmd.Flags().StringVar(&workDir, "workdir", ".", "Working directory for terraform")
	rootCmd.Flags().StringVar(&envName, "env", "", "Environment name to select")
	rootCmd.Flags().StringVar(&presetName, "preset", "", "Environment preset to apply")
	rootCmd.Flags().StringVar(&workspaceName, "workspace", "", "Terraform workspace to select")
	rootCmd.Flags().StringVar(&folderPath, "folder", "", "Terraform environment folder to use")
	rootCmd.Flags().StringVar(&configPath, "config", "", "Path to config file")
	rootCmd.Flags().StringVar(&themeName, "theme", "", "Theme name to use")
	rootCmd.Flags().BoolVar(&noHistory, "no-history", false, "Disable history logging")
	rootCmd.Flags().StringVar(&profileFlags, "profile", "", "Enable profiling (cpu,mem,trace,stats,all)")
	return rootCmd
}

//nolint:gocognit,gocyclo // CLI setup branches by mode and config source.
func run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errors.New("positional arguments are not supported; use --plan <path> or --plan -")
	}
	if readOnly && strings.TrimSpace(planFile) == "" {
		return errors.New("--readonly requires --plan")
	}

	// Initialize profiler from flags or environment.
	profiler := initProfiler()
	if profiler != nil && profiler.IsEnabled() {
		if err := profiler.Start(); err != nil {
			return fmt.Errorf("start profiler: %w", err)
		}
		defer func() {
			if err := profiler.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: profiler stop error: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Profiles written: %v\n", profiler.EnabledProfiles())
			}
		}()
	}

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
	if cmd != nil && !cmd.Flags().Changed("mouse") && cfg.Mouse != nil {
		mouseEnabled = *cfg.Mouse
	}
	if envName == "" {
		envName = strings.TrimSpace(cfg.DefaultEnvironment)
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

	// If a plan input is provided, preload plan into execution mode by default.
	if strings.TrimSpace(planFile) != "" {
		return runPlanInputMode(&cfg, overrideFlags, configManager)
	}

	return runExecutionMode(&cfg, overrideFlags, configManager, nil, "")
}

func runExecutionMode(
	cfg *config.Config,
	overrideFlags []string,
	configManager *config.Manager,
	preloadedPlan *terraform.Plan,
	preloadedPlanPath string,
) error {
	if err := configureWorkDirAndWorkspace(); err != nil {
		return err
	}
	return runExecutionModeConfigured(cfg, overrideFlags, configManager, preloadedPlan, preloadedPlanPath, "", "", false)
}

func runExecutionModeConfigured(
	cfg *config.Config,
	overrideFlags []string,
	configManager *config.Manager,
	preloadedPlan *terraform.Plan,
	preloadedPlanPath string,
	preloadedPlanDir string,
	preloadedPlanEnv string,
	preloadedPlanFromStdin bool,
) error {
	// Clean temp files from previous runs
	cleanupTempFilesOnStartup()

	flags, err := prepareExecutionFlags(cfg, overrideFlags)
	if err != nil {
		return err
	}
	appStyles, err := resolveAppStyles(cfg)
	if err != nil {
		return err
	}
	// Execution model applies flags per operation.
	// Keep executor defaults empty to avoid duplicate arguments.
	exec, err := buildExecutor(cfg, nil)
	if err != nil {
		return err
	}
	selectedEnv := resolveSelectedEnv()
	if strings.TrimSpace(preloadedPlanEnv) != "" {
		selectedEnv = strings.TrimSpace(preloadedPlanEnv)
	}
	resolvedPreloadedPlanDir := preloadedPlanDir
	if strings.TrimSpace(resolvedPreloadedPlanDir) == "" {
		resolvedPreloadedPlanDir = workDir
	}
	if absDir, absErr := filepath.Abs(resolvedPreloadedPlanDir); absErr == nil {
		resolvedPreloadedPlanDir = absDir
	}
	historyStore, historyLogger, err := openHistory(cfg)
	if err != nil {
		return err
	}
	// Ensure history store is closed if we fail before reaching executionModeRunner.
	var runErr error
	defer func() {
		if runErr != nil && historyStore != nil {
			_ = historyStore.Close()
		}
	}()

	notifier, err := openNotifier(cfg)
	if err != nil {
		return err
	}
	execCfg := ui.ExecutionConfig{
		Executor:               exec,
		Flags:                  flags,
		WorkDir:                workDir,
		EnvName:                selectedEnv,
		PreloadedPlanPath:      preloadedPlanPath,
		PreloadedPlanEnv:       strings.TrimSpace(preloadedPlanEnv),
		PreloadedPlanDir:       resolvedPreloadedPlanDir,
		PreloadedPlanFromStdin: preloadedPlanFromStdin,
		HistoryStore:           historyStore,
		HistoryLogger:          historyLogger,
		HistoryEnabled:         cfg.History.Enabled,
		Notifier:               notifier,
		Config:                 cfg,
		ConfigManager:          configManager,
	}
	model := ui.NewExecutionModelWithStyles(preloadedPlan, execCfg, appStyles)
	runErr = executionModeRunner(model, historyStore)
	return runErr
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
	if cfg.Terraform.Parallelism > 0 && !hasFlag(flags, "-parallelism") {
		flags = append(flags, fmt.Sprintf("-parallelism=%d", cfg.Terraform.Parallelism))
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
		if shouldDisableHistoryForError(err) {
			fmt.Fprintf(os.Stderr, "Warning: history disabled: %v\n", err)
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("history store: %w", err)
	}
	historyLogger = history.NewLogger(historyStore, history.Level(cfg.History.Level))
	return historyStore, historyLogger, nil
}

func shouldDisableHistoryForError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "go-sqlite3 requires cgo")
}

func openNotifier(cfg *config.Config) (notifications.Notifier, error) {
	if cfg == nil {
		return notifications.NopNotifier{}, nil
	}
	enabled := cfg.Notification != nil && *cfg.Notification
	return notifications.New(notifications.Config{
		Enabled: enabled,
	})
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

func runPlanInputMode(cfg *config.Config, overrideFlags []string, configManager *config.Manager) error {
	if err := configureWorkDirAndWorkspace(); err != nil {
		return err
	}
	plan, planPath, fromStdin, planWorkDir, planEnv, err := loadPlanInput(cfg, strings.TrimSpace(planFile))
	if err != nil {
		return err
	}
	if strings.TrimSpace(planWorkDir) == "" {
		planWorkDir = workDir
	}

	if !readOnly {
		return runExecutionModeConfigured(cfg, overrideFlags, configManager, plan, planPath, planWorkDir, planEnv, fromStdin)
	}

	appTheme, err := styles.ResolveTheme(cfg.Theme.Name)
	if err != nil {
		return err
	}
	appStyles := styles.NewStyles(appTheme)
	model := ui.NewModelWithStyles(plan, appStyles)
	return programRunner(model)
}

func detectCurrentWorkspaceForPlanInput(targetWorkDir string) string {
	manager, err := newWorkspaceManager(targetWorkDir)
	if err != nil {
		return ""
	}
	workspace, err := manager.Current(context.Background())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(workspace)
}

func loadPlanInput(cfg *config.Config, input string) (*terraform.Plan, string, bool, string, string, error) {
	if input == "-" {
		return loadPlanFromStdin()
	}
	return loadPlanFromBinaryFile(cfg, input)
}

func loadPlanFromStdin() (*terraform.Plan, string, bool, string, string, error) {
	if stdinIsTerminal() {
		return nil, "", false, "", "", errors.New("--plan - requires piped input, for example: terraform plan -no-color | lazytf --plan -")
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, "", false, "", "", fmt.Errorf("read stdin plan input: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, "", false, "", "", errors.New("stdin plan input is empty")
	}
	textParser := parser.NewTextParser()
	plan, err := textParser.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, "", false, "", "", fmt.Errorf("failed to parse stdin plan input: %w", err)
	}
	effectiveWorkDir := workDir
	if absDir, absErr := filepath.Abs(workDir); absErr == nil {
		effectiveWorkDir = absDir
	}
	return plan, "", true, effectiveWorkDir, detectCurrentWorkspaceForPlanInput(effectiveWorkDir), nil
}

func loadPlanFromBinaryFile(cfg *config.Config, planPath string) (*terraform.Plan, string, bool, string, string, error) {
	resolvedPlanPath := planPath
	if !filepath.IsAbs(resolvedPlanPath) {
		absPlanPath, absErr := filepath.Abs(resolvedPlanPath)
		if absErr != nil {
			return nil, "", false, "", "", fmt.Errorf("resolve plan path %q: %w", planPath, absErr)
		}
		resolvedPlanPath = absPlanPath
	}
	resolvedPlanPath = filepath.Clean(resolvedPlanPath)

	effectiveWorkDir := workDir
	if absDir, absErr := filepath.Abs(workDir); absErr == nil {
		effectiveWorkDir = absDir
	}

	exec, err := buildExecutor(cfg, nil)
	if err != nil {
		return nil, "", false, "", "", err
	}
	showResult, err := exec.Show(context.Background(), resolvedPlanPath, terraform.ShowOptions{})
	if err != nil {
		retryExec, retryErr := buildRetryExecutorFromWorkdirHint(exec, resolvedPlanPath)
		if retryErr == nil && retryExec != nil {
			showResult, err = retryExec.Show(context.Background(), resolvedPlanPath, terraform.ShowOptions{})
			effectiveWorkDir = retryExec.WorkDir()
		}
	}
	if err != nil {
		return nil, "", false, "", "", fmt.Errorf("failed to show plan file %q: %w", resolvedPlanPath, err)
	}
	textParser := parser.NewTextParser()
	plan, err := textParser.Parse(strings.NewReader(showResult.Output))
	if err != nil {
		return nil, "", false, "", "", fmt.Errorf("failed to parse terraform show output: %w", err)
	}
	planEnv := detectCurrentWorkspaceForPlanInput(effectiveWorkDir)
	return plan, resolvedPlanPath, false, effectiveWorkDir, planEnv, nil
}

func buildRetryExecutorFromWorkdirHint(base *terraform.Executor, planPath string) (*terraform.Executor, error) {
	if base == nil {
		return nil, errors.New("base executor is nil")
	}
	hintedWorkDir, err := readPlanWorkdirHint(planPath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(hintedWorkDir) == "" {
		return nil, errors.New("no plan workdir hint found")
	}
	return base.CloneWithWorkDir(hintedWorkDir)
}

func readPlanWorkdirHint(planPath string) (string, error) {
	hintPath := planPath + ".workdir"
	data, err := os.ReadFile(hintPath)
	if err != nil {
		return "", err
	}
	hint := strings.TrimSpace(string(data))
	if hint == "" {
		return "", errors.New("empty plan workdir hint")
	}
	if filepath.IsAbs(hint) {
		return "", errors.New("absolute plan workdir hints are not allowed")
	}
	resolved := filepath.Join(filepath.Dir(planPath), hint)
	return filepath.Clean(resolved), nil
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
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

func runProgramWithCleanup(model tea.Model, historyStore *history.Store) error {
	options := []tea.ProgramOption{
		tea.WithAltScreen(),
	}
	if mouseEnabled {
		options = append(options, tea.WithMouseCellMotion())
	}

	p := newProgram(model, options...)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run program in goroutine
	errChan := make(chan error, 1)
	go func() {
		_, err := p.Run()
		errChan <- err
	}()

	// Wait for completion or signal
	var runErr error
	select {
	case runErr = <-errChan:
		// Normal exit
	case sig := <-sigChan:
		// Signal received - quit program
		p.Send(tea.Quit())
		<-errChan // Wait for program to finish
		runErr = fmt.Errorf("interrupted by signal: %v", sig)
	}

	// Cleanup phase - always runs
	signal.Stop(sigChan)
	close(sigChan)

	if uiModel, ok := model.(*ui.Model); ok {
		uiModel.Cleanup()
	}

	// Close history store (backup - Cleanup should have done this)
	if historyStore != nil {
		_ = historyStore.Close()
	}

	if runErr != nil && !strings.Contains(runErr.Error(), "interrupted") {
		return fmt.Errorf("error running TUI: %w", runErr)
	}
	return nil
}

func cleanupTempFilesOnStartup() {
	dir := workDir
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	tmpDir := filepath.Join(dir, ".lazytf", "tmp")
	_ = os.RemoveAll(tmpDir)
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

func hasFlag(flags []string, name string) bool {
	for _, flag := range flags {
		if flag == name || strings.HasPrefix(flag, name+"=") {
			return true
		}
	}
	return false
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

func initProfiler() *profile.Profiler {
	// Check command-line flag first, then environment variable.
	var opts profile.Options
	if profileFlags != "" {
		opts = profile.ParseFlags(profileFlags)
	} else {
		opts = profile.ParseEnv()
	}

	if !opts.CPU && !opts.Memory && !opts.Trace && !opts.Stats {
		return nil
	}

	return profile.New(opts)
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
