package components

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestToastOverlayDebug(t *testing.T) {
	s := styles.DefaultStyles()

	// Create a base view with realistic dimensions (80x20)
	width := 80
	height := 20
	baseLines := make([]string, height)
	for i := range height {
		line := fmt.Sprintf("Line %2d: ", i+1)
		padding := strings.Repeat("=", width-len(line))
		baseLines[i] = line + padding
	}
	baseView := strings.Join(baseLines, "\n")

	fmt.Println("=== BASE VIEW ===")
	fmt.Println(baseView)
	fmt.Println()

	// Create toast
	toast := NewToast(s)
	toast.SetSize(width, height)
	toast.SetPosition(ToastTopLeft)
	_ = toast.ShowSuccess("Operation completed successfully!")

	fmt.Println("=== TOAST VIEW (standalone) ===")
	toastBox := toast.renderBox()
	fmt.Println(toastBox)
	fmt.Println()

	fmt.Println("=== TOAST OVERLAY ===")
	result := toast.Overlay(baseView)
	fmt.Println(result)
	fmt.Println()

	// Check dimensions
	fmt.Printf("Base view lines: %d\n", len(strings.Split(baseView, "\n")))
	fmt.Printf("Base view width: %d\n", lipgloss.Width(baseLines[0]))
	fmt.Printf("Toast width setting: %d\n", width)
	fmt.Printf("Toast height setting: %d\n", height)
	fmt.Printf("Toast box width: %d\n", lipgloss.Width(toastBox))
	fmt.Printf("Toast box height: %d\n", lipgloss.Height(toastBox))

	// Check if base content is visible around toast
	if !strings.Contains(result, "Line 20") {
		t.Error("Line 20 not visible in overlay - bottom should be visible")
	}
	if !strings.Contains(result, "Operation completed") {
		t.Error("Toast message not visible in overlay")
	}

	// Verify toast appears at top-left with padding=1
	// This means the toast starts at column 1, so we see 1 char from base then toast
	resultLines := strings.Split(result, "\n")
	if len(resultLines) > 1 {
		line2 := resultLines[1]
		fmt.Printf("\nLine 2 analysis:\n")
		fmt.Printf("Line 2: %q\n", line2)
		// With padding=1, first char is from base ("L" from "Line"), then toast box
		if !strings.Contains(line2, "╭") {
			t.Error("Toast box should be visible in line 2")
		}
	}
}

func TestToastSetDuration(t *testing.T) {
	s := styles.DefaultStyles()
	toast := NewToast(s)

	// Default duration
	toast.SetDuration(3 * time.Second)
	// Just ensure it doesn't panic
}

func TestToastSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	toast := NewToast(s)

	newStyles := styles.DefaultStyles()
	toast.SetStyles(newStyles)

	if toast.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestToastShowInfo(t *testing.T) {
	s := styles.DefaultStyles()
	toast := NewToast(s)
	toast.SetSize(80, 20)

	cmd := toast.ShowInfo("Test info message")
	if cmd == nil {
		t.Error("expected non-nil cmd from ShowInfo")
	}
	if !toast.IsVisible() {
		t.Error("expected toast to be visible after ShowInfo")
	}
}

func TestToastHide(t *testing.T) {
	s := styles.DefaultStyles()
	toast := NewToast(s)
	toast.SetSize(80, 20)

	// Show then hide
	_ = toast.ShowSuccess("Test message")
	toast.Hide()

	if toast.IsVisible() {
		t.Error("expected toast to not be visible after Hide")
	}
}

func TestToastIsVisible(t *testing.T) {
	s := styles.DefaultStyles()
	toast := NewToast(s)
	toast.SetSize(80, 20)

	// Initially not visible
	if toast.IsVisible() {
		t.Error("expected toast to not be visible initially")
	}

	// After showing
	_ = toast.ShowSuccess("Test")
	if !toast.IsVisible() {
		t.Error("expected toast to be visible after Show")
	}

	// After hiding
	toast.Hide()
	if toast.IsVisible() {
		t.Error("expected toast to not be visible after Hide")
	}
}

func TestToastUpdate(t *testing.T) {
	s := styles.DefaultStyles()
	toast := NewToast(s)
	toast.SetSize(80, 20)

	// Update with nil message
	cmd := toast.Update(nil)
	_ = cmd

	// Update with ClearToast
	_ = toast.ShowSuccess("Test")
	cmd = toast.Update(ClearToast{})
	_ = cmd
	if toast.IsVisible() {
		t.Error("expected toast to be hidden after ClearToast")
	}
}
