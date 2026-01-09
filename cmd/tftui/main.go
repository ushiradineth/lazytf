package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ushiradineth/tftui/internal/terraform"
	"github.com/ushiradineth/tftui/internal/terraform/parser"
	"github.com/ushiradineth/tftui/internal/ui"
)

var (
	version         = "0.1.0"
	planFile        string
	mouseEnabled    bool
	executeMode     bool
	autoPlan        bool
	tfFlags         string
	workDir         string
	programRunner   = runProgram
	executorFactory = terraform.NewExecutor
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tftui [plan-file]",
		Short: "A minimal TUI for reviewing Terraform plans",
		Long: `tftui is a Terminal User Interface for reviewing Terraform plans.
It displays plan changes in a clean, minimal interface inspired by Terraform Cloud,
showing only changed attributes in a git-style diff format.`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		RunE:    run,
	}

	rootCmd.Flags().StringVarP(&planFile, "file", "f", "", "Path to Terraform plan JSON file")
	mouseEnabled = os.Getenv("TMUX") == ""
	rootCmd.Flags().BoolVar(&mouseEnabled, "mouse", mouseEnabled, "Enable mouse support (disabled by default in tmux)")
	rootCmd.Flags().BoolVar(&executeMode, "execute", false, "Execute terraform commands directly")
	rootCmd.Flags().BoolVar(&autoPlan, "auto-plan", false, "Automatically run terraform plan on startup")
	rootCmd.Flags().StringVar(&tfFlags, "tf-flags", "", "Additional flags to pass to terraform")
	rootCmd.Flags().StringVar(&workDir, "workdir", ".", "Working directory for terraform")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(_ *cobra.Command, args []string) error {
	if executeMode {
		flags := splitFlags(tfFlags)
		exec, err := executorFactory(workDir, terraform.WithDefaultFlags(flags))
		if err != nil {
			return fmt.Errorf("failed to initialize terraform: %w", err)
		}

		model := ui.NewExecutionModel(nil, ui.ExecutionConfig{
			Executor: exec,
			AutoPlan: autoPlan,
			Flags:    flags,
		})
		return programRunner(model)
	}

	// Determine plan file path
	var planPath string
	switch {
	case len(args) > 0:
		planPath = args[0]
	case planFile != "":
		planPath = planFile
	default:
		return errors.New("no plan file specified. Usage: tftui <plan-file> or tftui --file <plan-file>")
	}

	// Parse the plan
	parser := parser.NewJSONParser()
	plan, err := parser.ParseFile(planPath)
	if err != nil {
		return fmt.Errorf("failed to parse plan file: %w", err)
	}

	// Create and run the TUI
	model := ui.NewModel(plan)
	return programRunner(model)
}

func runProgram(model tea.Model) error {
	options := []tea.ProgramOption{
		tea.WithAltScreen(),
	}
	if mouseEnabled {
		options = append(options, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(model, options...)

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
