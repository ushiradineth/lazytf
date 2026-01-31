package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestNewCommandLogPanel(t *testing.T) {
	t.Run("with styles", func(t *testing.T) {
		s := styles.DefaultStyles()
		panel := NewCommandLogPanel(s)
		if panel == nil {
			t.Fatal("expected non-nil panel")
		}
		if panel.styles != s {
			t.Error("styles not set correctly")
		}
		if !panel.visible {
			t.Error("expected panel to be visible by default")
		}
	})

	t.Run("nil styles uses default", func(t *testing.T) {
		panel := NewCommandLogPanel(nil)
		if panel == nil {
			t.Fatal("expected non-nil panel")
		}
		if panel.styles == nil {
			t.Error("expected default styles to be set")
		}
	})
}

func TestCommandLogPanelSetSize(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	if panel.height != 20 {
		t.Errorf("expected height 20, got %d", panel.height)
	}
}

func TestCommandLogPanelFocus(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	if panel.IsFocused() {
		t.Error("expected not focused by default")
	}

	panel.SetFocused(true)
	if !panel.IsFocused() {
		t.Error("expected focused after SetFocused(true)")
	}

	panel.SetFocused(false)
	if panel.IsFocused() {
		t.Error("expected not focused after SetFocused(false)")
	}
}

func TestCommandLogPanelVisibility(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	if !panel.IsVisible() {
		t.Error("expected visible by default")
	}

	panel.SetVisible(false)
	if panel.IsVisible() {
		t.Error("expected not visible after SetVisible(false)")
	}

	panel.SetVisible(true)
	if !panel.IsVisible() {
		t.Error("expected visible after SetVisible(true)")
	}
}

func TestCommandLogPanelSetDiagnostics(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	diagnostics := []terraform.Diagnostic{
		{Severity: "error", Summary: "Test error"},
		{Severity: "warning", Summary: "Test warning"},
	}

	panel.SetDiagnostics(diagnostics)
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelSetLogText(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	panel.SetLogText("test log output\nline 2")
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelSetParsedText(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	panel.SetParsedText("parsed summary")
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelSetShowRaw(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	panel.SetShowRaw(true)
	panel.SetShowRaw(false)
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelUpdate(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	result, cmd := panel.Update(tea.KeyMsg{Type: tea.KeyDown})
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd // cmd may be nil
}

func TestCommandLogPanelHandleKey(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	t.Run("not focused", func(t *testing.T) {
		panel.SetFocused(false)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
		if handled {
			t.Error("expected not handled when not focused")
		}
	})

	t.Run("focused - down", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - up", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - down arrow", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - up arrow", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - pgup", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyPgUp})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - pgdown", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyPgDown})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - home", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyHome})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - end", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEnd})
		if !handled {
			t.Error("expected handled when focused")
		}
	})

	t.Run("focused - unhandled key", func(t *testing.T) {
		panel.SetFocused(true)
		handled, _ := panel.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		if handled {
			t.Error("expected not handled for unbound key")
		}
	})
}

func TestCommandLogPanelView(t *testing.T) {
	t.Run("nil styles", func(t *testing.T) {
		panel := &CommandLogPanel{}
		view := panel.View()
		if view != "" {
			t.Error("expected empty view for nil styles")
		}
	})

	t.Run("zero height", func(t *testing.T) {
		panel := NewCommandLogPanel(styles.DefaultStyles())
		panel.SetSize(80, 0)
		view := panel.View()
		if view != "" {
			t.Error("expected empty view for zero height")
		}
	})

	t.Run("not visible", func(t *testing.T) {
		panel := NewCommandLogPanel(styles.DefaultStyles())
		panel.SetSize(80, 20)
		panel.SetVisible(false)
		view := panel.View()
		if view != "" {
			t.Error("expected empty view when not visible")
		}
	})

	t.Run("visible with content", func(t *testing.T) {
		panel := NewCommandLogPanel(styles.DefaultStyles())
		panel.SetSize(80, 20)
		panel.SetLogText("test log content")
		view := panel.View()
		if view == "" {
			t.Error("expected non-empty view")
		}
	})

	t.Run("empty content shows placeholder", func(t *testing.T) {
		panel := NewCommandLogPanel(styles.DefaultStyles())
		panel.SetSize(80, 20)
		view := panel.View()
		if view == "" {
			t.Error("expected non-empty view even without content")
		}
	})
}

func TestCommandLogPanelCalculateThumbSize(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	tests := []struct {
		name          string
		visibleHeight int
		totalLines    int
		expected      float64
	}{
		{"content fits", 20, 10, 1.0},
		{"content equals visible", 20, 20, 1.0},
		{"content exceeds visible", 20, 40, 0.5},
		{"zero visible height", 0, 10, 1.0},
		{"negative visible height", -1, 10, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := panel.calculateThumbSize(tt.visibleHeight, tt.totalLines)
			if result != tt.expected {
				t.Errorf("calculateThumbSize(%d, %d) = %f; want %f",
					tt.visibleHeight, tt.totalLines, result, tt.expected)
			}
		})
	}
}

func TestCommandLogPanelPadLine(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())

	tests := []struct {
		name     string
		line     string
		width    int
		expected string
	}{
		{"short line", "hello", 10, "hello     "},
		{"exact width", "hello", 5, "hello"},
		{"truncate long", "hello world", 5, "hello"},
		{"empty line", "", 5, "     "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := panel.padLine(tt.line, tt.width)
			if result != tt.expected {
				t.Errorf("padLine(%q, %d) = %q; want %q", tt.line, tt.width, result, tt.expected)
			}
		})
	}
}

func TestCommandLogPanelGetDiagnosticsPanel(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	diag := panel.GetDiagnosticsPanel()
	if diag == nil {
		t.Error("expected non-nil diagnostics panel")
	}
}

func TestCommandLogPanelAppendSessionLog(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	panel.AppendSessionLog("Planned", "terraform plan", "Plan output here")
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelClearSessionLogs(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	panel.SetSize(80, 20)

	panel.AppendSessionLog("Action", "command", "output")
	panel.ClearSessionLogs()
	if panel.GetDiagnosticsPanel() == nil {
		t.Error("diagnostics panel should not be nil")
	}
}

func TestCommandLogPanelSetStyles(t *testing.T) {
	panel := NewCommandLogPanel(styles.DefaultStyles())
	newStyles := styles.DefaultStyles()
	panel.SetStyles(newStyles)

	if panel.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}
