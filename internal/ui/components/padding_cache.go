package components

import (
	"strconv"
	"strings"
	"sync"
)

// PaddingCache provides pre-computed padding strings to avoid repeated allocations.
// Uses a pre-allocated array for common widths (0-300) for O(1) lock-free access,
// with a fallback map for larger widths.
type PaddingCache struct {
	// Fast path: pre-allocated array for widths 0-maxArrayWidth
	array    []string
	maxArray int

	// Slow path: map for larger widths
	mu       sync.RWMutex
	overflow map[int]string
}

const maxArrayWidth = 300

// globalPaddingCache is the shared instance for the application.
var globalPaddingCache = newPaddingCache()

// newPaddingCache creates a new padding cache.
func newPaddingCache() *PaddingCache {
	c := &PaddingCache{
		array:    make([]string, maxArrayWidth+1),
		maxArray: maxArrayWidth,
		overflow: make(map[int]string),
	}
	// Pre-compute all widths 0-300 for lock-free access
	for w := 0; w <= maxArrayWidth; w++ {
		c.array[w] = strings.Repeat(" ", w)
	}
	return c
}

// Get returns a padding string of the specified width.
// Lock-free for widths 0-300, uses mutex for larger widths.
func (c *PaddingCache) Get(width int) string {
	if width <= 0 {
		return ""
	}

	// Fast path: direct array access (no locks)
	if width <= c.maxArray {
		return c.array[width]
	}

	// Slow path: mutex-protected map for large widths
	c.mu.RLock()
	if s, ok := c.overflow[width]; ok {
		c.mu.RUnlock()
		return s
	}
	c.mu.RUnlock()

	// Generate and cache
	padding := strings.Repeat(" ", width)
	c.mu.Lock()
	c.overflow[width] = padding
	c.mu.Unlock()

	return padding
}

// GetPadding returns a padding string of the specified width using the global cache.
// This is the primary function to use throughout the codebase.
func GetPadding(width int) string {
	return globalPaddingCache.Get(width)
}

// BorderCache provides cached repeated border characters.
// Pre-computes common horizontal border patterns for lock-free access.
type BorderCache struct {
	// Pre-computed horizontal borders (most common).
	horizontal []string // "─" repeated

	// Map for other characters, keyed by "char:width".
	mu    sync.RWMutex
	other map[string]string
}

var globalBorderCache = newBorderCache()

func newBorderCache() *BorderCache {
	c := &BorderCache{
		horizontal: make([]string, maxArrayWidth+1),
		other:      make(map[string]string, 32),
	}
	// Pre-compute horizontal borders.
	for w := 0; w <= maxArrayWidth; w++ {
		c.horizontal[w] = strings.Repeat("─", w)
	}
	return c
}

// GetRepeatedChar returns a string with the given character repeated width times.
// Lock-free for "─" (horizontal border) up to width 300.
func GetRepeatedChar(char string, width int) string {
	if width <= 0 {
		return ""
	}

	// Fast path: horizontal border (most common).
	if char == "─" && width <= maxArrayWidth {
		return globalBorderCache.horizontal[width]
	}

	// Slow path: other characters.
	key := char + ":" + strconv.Itoa(width)

	globalBorderCache.mu.RLock()
	if s, ok := globalBorderCache.other[key]; ok {
		globalBorderCache.mu.RUnlock()
		return s
	}
	globalBorderCache.mu.RUnlock()

	// Generate and cache.
	result := strings.Repeat(char, width)

	if width <= maxArrayWidth {
		globalBorderCache.mu.Lock()
		globalBorderCache.other[key] = result
		globalBorderCache.mu.Unlock()
	}

	return result
}
