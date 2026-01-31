package components

import (
	"slices"
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestGetRandomTip(t *testing.T) {
	// Call multiple times to ensure it doesn't panic and returns valid tips
	for range 100 {
		tip := GetRandomTip()
		if tip == "" {
			t.Fatal("GetRandomTip returned empty string")
		}

		// Verify the tip is from our list
		if !slices.Contains(tips, tip) {
			t.Errorf("GetRandomTip returned unknown tip: %q", tip)
		}
	}
}

func TestTipsNotEmpty(t *testing.T) {
	if len(tips) == 0 {
		t.Fatal("tips slice should not be empty")
	}
}

func TestBuildEmptyStateTips(t *testing.T) {
	s := styles.DefaultStyles()
	testTip := "This is a test tip."
	result := buildEmptyStateTips(s, testTip, 80)

	if len(result) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(result))
	}

	// First line should mention 'L' key
	if !strings.Contains(result[0], "L") {
		t.Errorf("expected first line to mention 'L' key, got: %q", result[0])
	}

	// Second line should be empty (spacer)
	if strings.TrimSpace(result[1]) != "" {
		t.Errorf("expected second line to be empty, got: %q", result[1])
	}

	// Third line should contain "Tip:" and the test tip
	if !strings.Contains(result[2], "Tip:") {
		t.Errorf("expected third line to contain 'Tip:', got: %q", result[2])
	}
	if !strings.Contains(result[2], testTip) {
		t.Errorf("expected third line to contain the tip text, got: %q", result[2])
	}
}

func TestEmptyPanelShowsTips(t *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetSize(80, 10)

	out := panel.View()

	// Should show the tip hint
	if !strings.Contains(out, "Tip:") {
		t.Errorf("expected empty panel to show tips, got: %q", out)
	}

	// Should mention the 'L' key for toggling
	if !strings.Contains(out, "L") {
		t.Errorf("expected empty panel to mention 'L' key, got: %q", out)
	}
}

func TestTipStaysConsistentAcrossRenders(t *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetSize(80, 10)

	// Render multiple times and verify the tip stays the same
	firstRender := panel.View()
	for range 10 {
		currentRender := panel.View()
		if currentRender != firstRender {
			t.Errorf("tip changed between renders:\nfirst: %q\ncurrent: %q", firstRender, currentRender)
		}
	}
}
