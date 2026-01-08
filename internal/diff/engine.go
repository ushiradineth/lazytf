package diff

import (
	"github.com/ushiradineth/tftui/internal/terraform"
)

// Engine orchestrates diff calculation for terraform resources
type Engine struct {
	formatter *Formatter
}

// NewEngine creates a new diff engine
func NewEngine() *Engine {
	return &Engine{
		formatter: NewFormatter(),
	}
}

// CalculateResourceDiffs is a no-op kept for compatibility
// Diffs are now calculated on-demand in GetResourceDiffs
func (e *Engine) CalculateResourceDiffs(plan *terraform.Plan) error {
	// Nothing to do - diffs are calculated on demand
	return nil
}

// GetResourceDiffs retrieves the calculated diffs for a resource
func (e *Engine) GetResourceDiffs(resource *terraform.ResourceChange) []MinimalDiff {
	if resource.Change == nil {
		return nil
	}

	// Handle nil maps gracefully
	before := resource.Change.Before
	after := resource.Change.After
	afterUnknown := resource.Change.AfterUnknown

	if before == nil {
		before = make(map[string]interface{})
	}
	if after == nil {
		after = make(map[string]interface{})
	}
	if afterUnknown == nil {
		afterUnknown = make(map[string]interface{})
	}

	// Calculate diffs on demand
	return CalculateMinimalDiff(before, after, afterUnknown, resource.Change.BeforeOrder, resource.Change.AfterOrder, resource.Change.AfterUnknownOrder, "")
}

// CountChanges returns the number of changes in a resource
func (e *Engine) CountChanges(resource *terraform.ResourceChange) int {
	diffs := e.GetResourceDiffs(resource)
	return len(diffs)
}
