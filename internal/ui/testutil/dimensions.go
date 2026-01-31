package testutil

import (
	"fmt"
	"testing"
)

// DimensionSet represents a named set of dimensions for testing.
type DimensionSet struct {
	Name   string
	Width  int
	Height int
}

// String returns a string representation of the dimension set.
func (d DimensionSet) String() string {
	return fmt.Sprintf("%s(%dx%d)", d.Name, d.Width, d.Height)
}

// StandardDimensions returns a set of common terminal dimensions for testing.
func StandardDimensions() []DimensionSet {
	return []DimensionSet{
		{"minimal", 20, 5},
		{"narrow", 40, 24},
		{"standard", 80, 24},
		{"wide", 120, 30},
		{"ultrawide", 200, 50},
	}
}

// EdgeCaseDimensions returns edge case dimensions for testing boundary conditions.
func EdgeCaseDimensions() []DimensionSet {
	return []DimensionSet{
		{"zero", 0, 0},
		{"single_line", 80, 1},
		{"single_col", 1, 24},
		{"tiny", 3, 3},
		{"very_narrow", 10, 10},
		{"very_short", 80, 3},
	}
}

// CompactDimensions returns dimensions typical for compact/minimized views.
func CompactDimensions() []DimensionSet {
	return []DimensionSet{
		{"compact_small", 30, 8},
		{"compact_medium", 50, 12},
		{"compact_large", 70, 16},
	}
}

// WidescreenDimensions returns widescreen terminal dimensions.
func WidescreenDimensions() []DimensionSet {
	return []DimensionSet{
		{"widescreen_720p", 160, 45},
		{"widescreen_1080p", 213, 56},
		{"widescreen_4k", 426, 120},
	}
}

// HeightVariations returns dimensions with varying heights for a fixed width.
func HeightVariations(width int) []DimensionSet {
	return []DimensionSet{
		{fmt.Sprintf("h3_w%d", width), width, 3},
		{fmt.Sprintf("h5_w%d", width), width, 5},
		{fmt.Sprintf("h10_w%d", width), width, 10},
		{fmt.Sprintf("h20_w%d", width), width, 20},
		{fmt.Sprintf("h50_w%d", width), width, 50},
	}
}

// WidthVariations returns dimensions with varying widths for a fixed height.
func WidthVariations(height int) []DimensionSet {
	return []DimensionSet{
		{fmt.Sprintf("w20_h%d", height), 20, height},
		{fmt.Sprintf("w40_h%d", height), 40, height},
		{fmt.Sprintf("w80_h%d", height), 80, height},
		{fmt.Sprintf("w120_h%d", height), 120, height},
		{fmt.Sprintf("w200_h%d", height), 200, height},
	}
}

// RunDimensionMatrix runs a test function for each dimension set.
func RunDimensionMatrix(t *testing.T, sets []DimensionSet, fn func(t *testing.T, d DimensionSet)) {
	t.Helper()
	for _, d := range sets {
		t.Run(d.String(), func(t *testing.T) {
			fn(t, d)
		})
	}
}

// RunHeightMatrix runs a test function for various heights at a fixed width.
func RunHeightMatrix(t *testing.T, width int, fn func(t *testing.T, d DimensionSet)) {
	t.Helper()
	RunDimensionMatrix(t, HeightVariations(width), fn)
}

// RunWidthMatrix runs a test function for various widths at a fixed height.
func RunWidthMatrix(t *testing.T, height int, fn func(t *testing.T, d DimensionSet)) {
	t.Helper()
	RunDimensionMatrix(t, WidthVariations(height), fn)
}

// RunStandardDimensions runs a test function for standard terminal dimensions.
func RunStandardDimensions(t *testing.T, fn func(t *testing.T, d DimensionSet)) {
	t.Helper()
	RunDimensionMatrix(t, StandardDimensions(), fn)
}

// RunAllDimensions runs a test function for all common dimension sets.
func RunAllDimensions(t *testing.T, fn func(t *testing.T, d DimensionSet)) {
	t.Helper()
	all := append(StandardDimensions(), CompactDimensions()...)
	RunDimensionMatrix(t, all, fn)
}
