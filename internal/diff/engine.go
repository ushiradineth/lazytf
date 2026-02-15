package diff

import (
	"sync"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

const maxCacheSize = 1000

// Engine orchestrates diff calculation for terraform resources.
type Engine struct {
	mu    sync.RWMutex
	cache map[string][]MinimalDiff // keyed by resource.Address for stability
}

// NewEngine creates a new diff engine.
func NewEngine() *Engine {
	return &Engine{
		cache: make(map[string][]MinimalDiff),
	}
}

// ResetCache clears cached diff results.
func (e *Engine) ResetCache() {
	e.mu.Lock()
	e.cache = make(map[string][]MinimalDiff)
	e.mu.Unlock()
}

// GetResourceDiffs retrieves the calculated diffs for a resource.
func (e *Engine) GetResourceDiffs(resource *terraform.ResourceChange) []MinimalDiff {
	if resource == nil {
		return nil
	}

	cacheKey := resource.Address
	e.mu.RLock()
	if diffs, ok := e.cache[cacheKey]; ok {
		e.mu.RUnlock()
		return diffs
	}
	e.mu.RUnlock()

	if resource.Change == nil {
		return nil
	}

	// Handle nil maps gracefully
	before := resource.Change.Before
	after := resource.Change.After
	afterUnknown := resource.Change.AfterUnknown

	if before == nil {
		before = make(map[string]any)
	}
	if after == nil {
		after = make(map[string]any)
	}
	if afterUnknown == nil {
		afterUnknown = make(map[string]any)
	}

	// Calculate diffs on demand
	diffs := CalculateDiffs(before, after, afterUnknown, resource.Change.BeforeOrder, resource.Change.AfterOrder, resource.Change.AfterUnknownOrder, "")

	e.mu.Lock()
	// Evict cache if it exceeds max size
	if len(e.cache) >= maxCacheSize {
		e.cache = make(map[string][]MinimalDiff)
	}
	e.cache[cacheKey] = diffs
	e.mu.Unlock()

	return diffs
}

// CountChanges returns the number of changes in a resource.
func (e *Engine) CountChanges(resource *terraform.ResourceChange) int {
	diffs := e.GetResourceDiffs(resource)
	return len(diffs)
}
