package ui

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestMainAreaViewWithNilStylesReturnsEmpty(t *testing.T) {
	m := &MainArea{}
	if got := m.View(); got != "" {
		t.Fatalf("expected empty view for nil styles, got %q", got)
	}
}

func TestMainAreaViewWithZeroHeightReturnsEmpty(t *testing.T) {
	m := &MainArea{styles: styles.DefaultStyles()}
	if got := m.View(); got != "" {
		t.Fatalf("expected empty view for zero height, got %q", got)
	}
}
