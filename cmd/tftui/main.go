package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ushiradineth/tftui/internal/terraform/parser"
	"github.com/ushiradineth/tftui/internal/ui"
)

var (
	version      = "0.1.0"
	planFile     string
	mouseEnabled bool
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

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Determine plan file path
	var planPath string
	if len(args) > 0 {
		planPath = args[0]
	} else if planFile != "" {
		planPath = planFile
	} else {
		return fmt.Errorf("no plan file specified. Usage: tftui <plan-file> or tftui --file <plan-file>")
	}

	// Parse the plan
	parser := parser.NewJSONParser()
	plan, err := parser.ParseFile(planPath)
	if err != nil {
		return fmt.Errorf("failed to parse plan file: %w", err)
	}

	// Create and run the TUI
	model := ui.NewModel(plan)
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
