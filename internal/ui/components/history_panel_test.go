package components

import (
	"strings"
	"testing"
	"time"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestHistoryPanelRendersEntries(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)
	panel.SetSize(50, 6)
	panel.SetEntries([]history.Entry{
		{
			StartedAt: time.Now(),
			Status:    history.StatusSuccess,
			Summary:   "+ 1 to create ~ 0 to update - 0 to destroy",
		},
	})
	panel.SetSelection(0, true)

	out := panel.View()
	if !strings.Contains(out, "Apply history") {
		t.Fatalf("expected header in output")
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("expected status in output")
	}
	if !strings.Contains(out, "+ 1 to create") {
		t.Fatalf("expected summary in output")
	}
}
