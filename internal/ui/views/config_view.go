package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/config"
	"github.com/ushiradineth/tftui/internal/styles"
)

// ConfigView renders application configuration details.
type ConfigView struct {
	styles *styles.Styles
	width  int
	height int
	config *config.Config
}

// NewConfigView creates a new config view.
func NewConfigView(s *styles.Styles) *ConfigView {
	return &ConfigView{styles: s}
}

// SetSize updates the layout size.
func (v *ConfigView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetConfig updates the configuration to render.
func (v *ConfigView) SetConfig(cfg *config.Config) {
	v.config = cfg
}

// View renders the configuration details.
func (v *ConfigView) View() string {
	if v.styles == nil {
		return ""
	}
	width := minInt(70, v.width-4)
	if width < 34 {
		width = 34
	}

	lines := []string{v.styles.Highlight.Render("Settings")}
	if v.config == nil {
		lines = append(lines, "", "No configuration loaded.", "", "esc: back")
		return v.renderBox(lines, width)
	}

	cfg := v.config
	lines = append(lines, "")
	lines = append(lines, v.styles.Dimmed.Render("Theme:")+" "+cfg.Theme.Name)

	lines = append(lines, "")
	lines = append(lines, v.styles.Highlight.Render("Terraform"))
	lines = append(lines, "binary: "+fallback(cfg.Terraform.Binary, "default"))
	lines = append(lines, "working dir: "+fallback(cfg.Terraform.WorkingDir, "default"))
	lines = append(lines, fmt.Sprintf("timeout: %s", cfg.Terraform.Timeout))
	lines = append(lines, fmt.Sprintf("parallelism: %d", cfg.Terraform.Parallelism))
	lines = append(lines, "default flags: "+strings.Join(cfg.Terraform.DefaultFlags, " "))

	lines = append(lines, "")
	lines = append(lines, v.styles.Highlight.Render("UI"))
	lines = append(lines, fmt.Sprintf("mouse enabled: %t", cfg.UI.MouseEnabled))
	lines = append(lines, fmt.Sprintf("compact mode: %t", cfg.UI.CompactMode))
	lines = append(lines, fmt.Sprintf("animations: %t", cfg.UI.AnimationsEnabled))
	lines = append(lines, fmt.Sprintf("split default: %t", cfg.UI.SplitViewDefault))
	lines = append(lines, fmt.Sprintf("split ratio: %.2f", cfg.UI.SplitRatio))

	lines = append(lines, "")
	lines = append(lines, v.styles.Highlight.Render("History"))
	lines = append(lines, fmt.Sprintf("enabled: %t", cfg.History.Enabled))
	lines = append(lines, "level: "+cfg.History.Level)
	lines = append(lines, "path: "+fallback(cfg.History.Path, "default"))
	lines = append(lines, fmt.Sprintf("retention days: %d", cfg.History.RetentionDays))
	lines = append(lines, fmt.Sprintf("max entries: %d", cfg.History.MaxEntries))

	lines = append(lines, "")
	lines = append(lines, "esc: back")

	return v.renderBox(lines, width)
}

func (v *ConfigView) renderBox(lines []string, width int) string {
	content := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	box := v.styles.Border.Width(width).Render(content)
	if v.width == 0 || v.height == 0 {
		return box
	}
	return lipgloss.Place(v.width, v.height, lipgloss.Center, lipgloss.Center, box)
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
