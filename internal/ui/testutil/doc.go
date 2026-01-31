// Package testutil provides utilities for systematically testing TUI rendering
// in the lazytf application. It includes tools for:
//
//   - Capturing and analyzing rendered output (with ANSI stripping)
//   - Testing across different terminal dimensions
//   - Asserting height, width, content, and visual properties
//   - Golden file comparisons for regression testing
//   - Common test fixtures for resources and history entries
//
// # Basic Usage
//
// Use RenderCapture to capture output from a view function:
//
//	result := testutil.RenderCapture(t, component.View, 80, 24)
//	result.AssertHeight(t, 24).AssertContains(t, "expected text")
//
// Use RenderComponent for Panel-like components:
//
//	result := testutil.RenderComponent(t, myPanel, 80, 24)
//	result.AssertNoLineOverflow(t).AssertHasBorder(t)
//
// # Dimension Testing
//
// Test across multiple terminal sizes using dimension matrices:
//
//	testutil.RunStandardDimensions(t, func(t *testing.T, d testutil.DimensionSet) {
//	    result := testutil.RenderComponent(t, myComponent, d.Width, d.Height)
//	    result.AssertHeight(t, d.Height)
//	})
//
// # Golden Files
//
// Use golden files for regression testing:
//
//	golden := testutil.ComponentGolden("my_component")
//	result := testutil.RenderComponent(t, myComponent, 80, 24)
//	golden.AssertPlain(t, "basic_render", result)
//
// Update golden files with: UPDATE_GOLDEN=1 go test ./...
//
// # Test Fixtures
//
// Use pre-defined fixtures for consistent test data:
//
//	component.SetResources(testutil.SampleResources)  // 5 mixed action types
//	component.SetResources(testutil.ManyResources)    // 50+ for scrollbar testing
//	component.SetResources(testutil.ModuleResources)  // Nested modules
package testutil
