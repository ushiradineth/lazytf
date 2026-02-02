package views

import (
	"strings"
	"testing"
	"time"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestNewConfigView(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewConfigView(s)
	if view == nil {
		t.Fatal("expected non-nil view")
	}
	if view.styles != s {
		t.Error("styles not set correctly")
	}
}

func TestConfigViewSetSize(t *testing.T) {
	view := NewConfigView(styles.DefaultStyles())
	view.SetSize(80, 24)
	if view.width != 80 {
		t.Errorf("expected width 80, got %d", view.width)
	}
	if view.height != 24 {
		t.Errorf("expected height 24, got %d", view.height)
	}
}

func TestConfigViewSetConfig(t *testing.T) {
	view := NewConfigView(styles.DefaultStyles())
	cfg := &config.Config{
		Theme: config.ThemeConfig{Name: "dark"},
	}
	view.SetConfig(cfg)
	if view.config != cfg {
		t.Error("config not set correctly")
	}
}

func TestConfigViewViewNilStyles(t *testing.T) {
	view := &ConfigView{}
	out := view.View()
	if out != "" {
		t.Error("expected empty output for nil styles")
	}
}

func TestConfigViewViewNoConfig(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewConfigView(s)
	view.SetSize(80, 24)

	out := view.View()
	if !strings.Contains(out, "Settings") {
		t.Error("expected Settings header")
	}
	if !strings.Contains(out, "No configuration loaded") {
		t.Error("expected 'No configuration loaded' message")
	}
	if !strings.Contains(out, "esc: back") {
		t.Error("expected esc hint")
	}
}

func TestConfigViewViewWithConfig(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewConfigView(s)
	view.SetSize(80, 40)

	cfg := &config.Config{
		Theme: config.ThemeConfig{Name: "mocha"},
		Terraform: config.TerraformConfig{
			Binary:       "/usr/local/bin/terraform",
			WorkingDir:   "/projects/infra",
			Timeout:      5 * time.Minute,
			Parallelism:  10,
			DefaultFlags: []string{"-no-color", "-input=false"},
		},
		UI: config.UIConfig{
			MouseEnabled:      true,
			CompactMode:       false,
			AnimationsEnabled: true,
			SplitViewDefault:  true,
			SplitRatio:        0.4,
		},
		History: config.HistoryConfig{
			Enabled:       true,
			Level:         "detailed",
			Path:          "/home/user/.lazytf/history.db",
			RetentionDays: 30,
			MaxEntries:    1000,
		},
	}
	view.SetConfig(cfg)

	out := view.View()

	// Check headers
	if !strings.Contains(out, "Settings") {
		t.Error("expected Settings header")
	}
	if !strings.Contains(out, "Terraform") {
		t.Error("expected Terraform header")
	}
	if !strings.Contains(out, "UI") {
		t.Error("expected UI header")
	}
	if !strings.Contains(out, "History") {
		t.Error("expected History header")
	}

	// Check theme
	if !strings.Contains(out, "mocha") {
		t.Error("expected theme name in output")
	}

	// Check terraform settings
	if !strings.Contains(out, "/usr/local/bin/terraform") {
		t.Error("expected terraform binary path")
	}
	if !strings.Contains(out, "/projects/infra") {
		t.Error("expected working dir")
	}
	if !strings.Contains(out, "5m") {
		t.Error("expected timeout value")
	}
	if !strings.Contains(out, "10") {
		t.Error("expected parallelism value")
	}
	if !strings.Contains(out, "-no-color") {
		t.Error("expected default flags")
	}

	// Check UI settings
	if !strings.Contains(out, "mouse enabled: true") {
		t.Error("expected mouse enabled")
	}
	if !strings.Contains(out, "compact mode: false") {
		t.Error("expected compact mode")
	}
	if !strings.Contains(out, "animations: true") {
		t.Error("expected animations")
	}
	if !strings.Contains(out, "split ratio: 0.40") {
		t.Error("expected split ratio")
	}

	// Check history settings
	if !strings.Contains(out, "enabled: true") {
		t.Error("expected history enabled")
	}
	if !strings.Contains(out, "detailed") {
		t.Error("expected history level")
	}
	if !strings.Contains(out, "retention days: 30") {
		t.Error("expected retention days")
	}
	if !strings.Contains(out, "max entries: 1000") {
		t.Error("expected max entries")
	}
}

func TestConfigViewViewWithDefaults(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewConfigView(s)
	view.SetSize(80, 40)

	cfg := &config.Config{
		Theme:     config.ThemeConfig{Name: "default"},
		Terraform: config.TerraformConfig{},
		History:   config.HistoryConfig{},
	}
	view.SetConfig(cfg)

	out := view.View()

	// Empty values should show "default"
	if !strings.Contains(out, "binary: default") {
		t.Error("expected 'default' for empty binary")
	}
	if !strings.Contains(out, "working dir: default") {
		t.Error("expected 'default' for empty working dir")
	}
	if !strings.Contains(out, "path: default") {
		t.Error("expected 'default' for empty history path")
	}
}

func TestConfigViewViewSmallWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewConfigView(s)
	view.SetSize(20, 40) // Very small width

	cfg := &config.Config{
		Theme: config.ThemeConfig{Name: "dark"},
	}
	view.SetConfig(cfg)

	out := view.View()
	if out == "" {
		t.Error("expected non-empty output even with small width")
	}
}

func TestConfigViewViewZeroSize(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewConfigView(s)
	// Don't set size, width/height remain 0

	cfg := &config.Config{
		Theme: config.ThemeConfig{Name: "dark"},
	}
	view.SetConfig(cfg)

	out := view.View()
	if out == "" {
		t.Error("expected non-empty output")
	}
	// Should render without placement (no centering)
}

func TestFallback(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue string
		expected     string
	}{
		{"non-empty value", "custom", "default", "custom"},
		{"empty value", "", "default", "default"},
		{"whitespace only", "   ", "default", "default"},
		{"empty default", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fallback(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("fallback(%q, %q) = %q; want %q", tt.value, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestConfigViewRenderBox(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewConfigView(s)

	// Test with zero size (no placement)
	lines := []string{"Line 1", "Line 2"}
	out := view.renderBox(lines, 40)
	if out == "" {
		t.Error("expected non-empty output")
	}

	// Test with size set (with placement)
	view.SetSize(80, 20)
	out = view.renderBox(lines, 40)
	if out == "" {
		t.Error("expected non-empty output with size set")
	}
}

func TestConfigViewSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewConfigView(s)

	newStyles := styles.DefaultStyles()
	view.SetStyles(newStyles)

	if view.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}
