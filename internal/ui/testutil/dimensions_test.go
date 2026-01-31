package testutil

import (
	"testing"
)

func TestDimensionSetString(t *testing.T) {
	d := DimensionSet{Name: "test", Width: 80, Height: 24}
	expected := "test(80x24)"
	if d.String() != expected {
		t.Errorf("expected %q, got %q", expected, d.String())
	}
}

func TestStandardDimensions(t *testing.T) {
	dims := StandardDimensions()
	if len(dims) != 5 {
		t.Errorf("expected 5 standard dimensions, got %d", len(dims))
	}

	// Verify expected dimensions are present
	names := make(map[string]bool)
	for _, d := range dims {
		names[d.Name] = true
	}

	expectedNames := []string{"minimal", "narrow", "standard", "wide", "ultrawide"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("expected dimension set %q not found", name)
		}
	}
}

func TestEdgeCaseDimensions(t *testing.T) {
	dims := EdgeCaseDimensions()
	if len(dims) < 3 {
		t.Errorf("expected at least 3 edge case dimensions, got %d", len(dims))
	}

	// Verify zero dimensions are included
	foundZero := false
	for _, d := range dims {
		if d.Width == 0 && d.Height == 0 {
			foundZero = true
			break
		}
	}
	if !foundZero {
		t.Error("expected zero dimensions in edge cases")
	}
}

func TestHeightVariations(t *testing.T) {
	dims := HeightVariations(80)
	if len(dims) < 3 {
		t.Errorf("expected at least 3 height variations, got %d", len(dims))
	}

	// All should have width 80
	for _, d := range dims {
		if d.Width != 80 {
			t.Errorf("expected width 80, got %d for %s", d.Width, d.Name)
		}
	}

	// Heights should vary
	heights := make(map[int]bool)
	for _, d := range dims {
		heights[d.Height] = true
	}
	if len(heights) < 3 {
		t.Errorf("expected at least 3 different heights, got %d", len(heights))
	}
}

func TestWidthVariations(t *testing.T) {
	dims := WidthVariations(24)
	if len(dims) < 3 {
		t.Errorf("expected at least 3 width variations, got %d", len(dims))
	}

	// All should have height 24
	for _, d := range dims {
		if d.Height != 24 {
			t.Errorf("expected height 24, got %d for %s", d.Height, d.Name)
		}
	}

	// Widths should vary
	widths := make(map[int]bool)
	for _, d := range dims {
		widths[d.Width] = true
	}
	if len(widths) < 3 {
		t.Errorf("expected at least 3 different widths, got %d", len(widths))
	}
}

func TestRunDimensionMatrix(t *testing.T) {
	dims := []DimensionSet{
		{"a", 10, 5},
		{"b", 20, 10},
	}

	count := 0
	RunDimensionMatrix(t, dims, func(t *testing.T, d DimensionSet) {
		count++
		if d.Width < 1 || d.Height < 1 {
			t.Errorf("unexpected zero dimension: %v", d)
		}
	})

	if count != 2 {
		t.Errorf("expected 2 runs, got %d", count)
	}
}

func TestRunHeightMatrix(t *testing.T) {
	count := 0
	RunHeightMatrix(t, 80, func(t *testing.T, d DimensionSet) {
		count++
		if d.Width != 80 {
			t.Errorf("expected width 80, got %d", d.Width)
		}
	})

	if count < 3 {
		t.Errorf("expected at least 3 runs, got %d", count)
	}
}

func TestRunWidthMatrix(t *testing.T) {
	count := 0
	RunWidthMatrix(t, 24, func(t *testing.T, d DimensionSet) {
		count++
		if d.Height != 24 {
			t.Errorf("expected height 24, got %d", d.Height)
		}
	})

	if count < 3 {
		t.Errorf("expected at least 3 runs, got %d", count)
	}
}

func TestRunStandardDimensions(t *testing.T) {
	count := 0
	RunStandardDimensions(t, func(t *testing.T, d DimensionSet) {
		count++
	})

	if count != 5 {
		t.Errorf("expected 5 runs, got %d", count)
	}
}

func TestRunAllDimensions(t *testing.T) {
	count := 0
	RunAllDimensions(t, func(t *testing.T, d DimensionSet) {
		count++
	})

	// Should include standard + compact dimensions
	if count < 8 {
		t.Errorf("expected at least 8 runs, got %d", count)
	}
}
