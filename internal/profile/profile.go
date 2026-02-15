// Package profile provides runtime profiling and diagnostics for lazytf.
// Enable via environment variables or command flags:
//
//	LAZYTF_PROFILE=cpu,mem,trace ./lazytf
//	./lazytf --profile cpu,mem
//
// Profiles are written to the current directory with timestamps.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"
)

// Profiler manages runtime profiling.
type Profiler struct {
	mu sync.Mutex

	// Profile outputs
	cpuFile   *os.File
	memFile   *os.File
	traceFile *os.File

	// Options
	outputDir string
	prefix    string

	// Enabled profiles
	cpuEnabled   bool
	memEnabled   bool
	traceEnabled bool

	// Runtime stats collection
	statsEnabled  bool
	statsInterval time.Duration
	statsTicker   *time.Ticker
	statsDone     chan struct{}
	stats         []RuntimeStats
}

// RuntimeStats captures a snapshot of runtime metrics.
type RuntimeStats struct {
	Timestamp    time.Time
	HeapAlloc    uint64 // Bytes allocated and still in use
	HeapSys      uint64 // Bytes obtained from system
	HeapObjects  uint64 // Number of allocated objects
	NumGC        uint32 // Number of completed GC cycles
	NumGoroutine int    // Number of goroutines
	PauseTotalNs uint64 // Total GC pause time
}

// Options configures the profiler.
type Options struct {
	// OutputDir is where profile files are written (default: current dir).
	OutputDir string

	// Prefix for profile filenames (default: "lazytf").
	Prefix string

	// CPU enables CPU profiling.
	CPU bool

	// Memory enables memory profiling (heap snapshot on stop).
	Memory bool

	// Trace enables execution tracing.
	Trace bool

	// Stats enables periodic runtime stats collection.
	Stats bool

	// StatsInterval is how often to collect runtime stats (default: 1s).
	StatsInterval time.Duration
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		OutputDir:     ".",
		Prefix:        "lazytf",
		StatsInterval: time.Second,
	}
}

// ParseEnv parses profiling options from LAZYTF_PROFILE environment variable.
// Format: "cpu,mem,trace,stats" (comma-separated).
func ParseEnv() Options {
	opts := DefaultOptions()

	env := os.Getenv("LAZYTF_PROFILE")
	if env == "" {
		return opts
	}

	for _, p := range strings.Split(strings.ToLower(env), ",") {
		switch strings.TrimSpace(p) {
		case "cpu":
			opts.CPU = true
		case "mem", "memory", "heap":
			opts.Memory = true
		case "trace":
			opts.Trace = true
		case "stats":
			opts.Stats = true
		case "all":
			opts.CPU = true
			opts.Memory = true
			opts.Trace = true
			opts.Stats = true
		}
	}

	if dir := os.Getenv("LAZYTF_PROFILE_DIR"); dir != "" {
		opts.OutputDir = dir
	}

	return opts
}

// ParseFlags parses a comma-separated profile string (e.g., from --profile flag).
func ParseFlags(profiles string) Options {
	opts := DefaultOptions()

	if profiles == "" {
		return opts
	}

	for _, p := range strings.Split(strings.ToLower(profiles), ",") {
		switch strings.TrimSpace(p) {
		case "cpu":
			opts.CPU = true
		case "mem", "memory", "heap":
			opts.Memory = true
		case "trace":
			opts.Trace = true
		case "stats":
			opts.Stats = true
		case "all":
			opts.CPU = true
			opts.Memory = true
			opts.Trace = true
			opts.Stats = true
		}
	}

	return opts
}

// New creates a new profiler with the given options.
func New(opts Options) *Profiler {
	interval := opts.StatsInterval
	if interval <= 0 {
		interval = time.Second
	}

	return &Profiler{
		outputDir:     opts.OutputDir,
		prefix:        opts.Prefix,
		cpuEnabled:    opts.CPU,
		memEnabled:    opts.Memory,
		traceEnabled:  opts.Trace,
		statsEnabled:  opts.Stats,
		statsInterval: interval,
	}
}

// Start begins all enabled profiling.
//
//nolint:gocognit,gocyclo,funlen // Startup needs ordered cleanup paths for partial initialization failures.
func (p *Profiler) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	timestamp := time.Now().Format("20060102-150405")
	startedCPU := false
	startedTrace := false

	cleanupAndWrap := func(prefix string, cause error) error {
		if p.statsDone != nil {
			close(p.statsDone)
			p.statsDone = nil
		}
		if p.statsTicker != nil {
			p.statsTicker.Stop()
			p.statsTicker = nil
		}

		if startedTrace {
			trace.Stop()
			startedTrace = false
		}
		if p.traceFile != nil {
			_ = p.traceFile.Close()
			p.traceFile = nil
		}

		if startedCPU {
			pprof.StopCPUProfile()
			startedCPU = false
		}
		if p.cpuFile != nil {
			_ = p.cpuFile.Close()
			p.cpuFile = nil
		}

		if p.memFile != nil {
			_ = p.memFile.Close()
			p.memFile = nil
		}

		return fmt.Errorf("%s: %w", prefix, cause)
	}

	// Ensure output directory exists.
	if err := os.MkdirAll(p.outputDir, 0o755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	// CPU profiling.
	if p.cpuEnabled {
		path := filepath.Join(p.outputDir, fmt.Sprintf("%s-cpu-%s.prof", p.prefix, timestamp))
		f, err := os.Create(path)
		if err != nil {
			return cleanupAndWrap("create cpu profile", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			return cleanupAndWrap("start cpu profile", err)
		}
		startedCPU = true
		p.cpuFile = f
	}

	// Execution trace.
	if p.traceEnabled {
		path := filepath.Join(p.outputDir, fmt.Sprintf("%s-trace-%s.out", p.prefix, timestamp))
		f, err := os.Create(path)
		if err != nil {
			return cleanupAndWrap("create trace file", err)
		}
		if err := trace.Start(f); err != nil {
			f.Close()
			return cleanupAndWrap("start trace", err)
		}
		startedTrace = true
		p.traceFile = f
	}

	// Memory profile file (written on stop).
	if p.memEnabled {
		path := filepath.Join(p.outputDir, fmt.Sprintf("%s-mem-%s.prof", p.prefix, timestamp))
		f, err := os.Create(path)
		if err != nil {
			return cleanupAndWrap("create mem profile", err)
		}
		p.memFile = f
	}

	// Runtime stats collection.
	if p.statsEnabled {
		p.statsTicker = time.NewTicker(p.statsInterval)
		p.statsDone = make(chan struct{})
		p.stats = make([]RuntimeStats, 0, 300) // Pre-allocate for 5 minutes
		// Pass ticker and done channel directly to avoid race.
		go p.collectStats(p.statsTicker, p.statsDone)
	}

	return nil
}

// Stop ends all profiling and writes output files.
func (p *Profiler) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []string

	// Stop CPU profiling.
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		if err := p.cpuFile.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close cpu profile: %v", err))
		}
		p.cpuFile = nil
	}

	// Stop trace.
	if p.traceFile != nil {
		trace.Stop()
		if err := p.traceFile.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close trace: %v", err))
		}
		p.traceFile = nil
	}

	// Write memory profile.
	if p.memFile != nil {
		runtime.GC() // Get up-to-date statistics
		if err := pprof.WriteHeapProfile(p.memFile); err != nil {
			errs = append(errs, fmt.Sprintf("write heap profile: %v", err))
		}
		if err := p.memFile.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close mem profile: %v", err))
		}
		p.memFile = nil
	}

	// Stop stats collection.
	if p.statsTicker != nil {
		close(p.statsDone)   // Signal goroutine to exit first
		p.statsTicker.Stop() // Then stop the ticker
		p.statsTicker = nil
		p.statsDone = nil

		// Write stats to file.
		if err := p.writeStats(); err != nil {
			errs = append(errs, fmt.Sprintf("write stats: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("profile stop errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// collectStats runs in a goroutine to periodically collect runtime metrics.
// ticker and done are passed directly to avoid race conditions with Stop().
func (p *Profiler) collectStats(ticker *time.Ticker, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			stat := RuntimeStats{
				Timestamp:    time.Now(),
				HeapAlloc:    m.HeapAlloc,
				HeapSys:      m.HeapSys,
				HeapObjects:  m.HeapObjects,
				NumGC:        m.NumGC,
				NumGoroutine: runtime.NumGoroutine(),
				PauseTotalNs: m.PauseTotalNs,
			}

			p.mu.Lock()
			p.stats = append(p.stats, stat)
			p.mu.Unlock()
		}
	}
}

// writeStats writes collected runtime stats to a CSV file.
func (p *Profiler) writeStats() error {
	if len(p.stats) == 0 {
		return nil
	}

	timestamp := time.Now().Format("20060102-150405")
	path := filepath.Join(p.outputDir, fmt.Sprintf("%s-stats-%s.csv", p.prefix, timestamp))

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write header.
	fmt.Fprintln(f, "timestamp,heap_alloc_mb,heap_sys_mb,heap_objects,num_gc,goroutines,gc_pause_ms")

	// Write data.
	for _, s := range p.stats {
		fmt.Fprintf(f, "%s,%.2f,%.2f,%d,%d,%d,%.2f\n",
			s.Timestamp.Format(time.RFC3339),
			float64(s.HeapAlloc)/1024/1024,
			float64(s.HeapSys)/1024/1024,
			s.HeapObjects,
			s.NumGC,
			s.NumGoroutine,
			float64(s.PauseTotalNs)/1e6,
		)
	}

	return nil
}

// GetStats returns a copy of collected runtime stats.
func (p *Profiler) GetStats() []RuntimeStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]RuntimeStats, len(p.stats))
	copy(result, p.stats)
	return result
}

// CurrentStats returns the current runtime statistics.
func CurrentStats() RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return RuntimeStats{
		Timestamp:    time.Now(),
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapObjects:  m.HeapObjects,
		NumGC:        m.NumGC,
		NumGoroutine: runtime.NumGoroutine(),
		PauseTotalNs: m.PauseTotalNs,
	}
}

// FormatStats returns a human-readable string of current runtime stats.
func FormatStats() string {
	s := CurrentStats()
	return fmt.Sprintf(
		"Heap: %.1f MB (%.1f MB sys) | Objects: %d | GC: %d (%.1f ms total) | Goroutines: %d",
		float64(s.HeapAlloc)/1024/1024,
		float64(s.HeapSys)/1024/1024,
		s.HeapObjects,
		s.NumGC,
		float64(s.PauseTotalNs)/1e6,
		s.NumGoroutine,
	)
}

// IsEnabled returns true if any profiling is enabled.
func (p *Profiler) IsEnabled() bool {
	return p.cpuEnabled || p.memEnabled || p.traceEnabled || p.statsEnabled
}

// EnabledProfiles returns a list of enabled profile types.
func (p *Profiler) EnabledProfiles() []string {
	var profiles []string
	if p.cpuEnabled {
		profiles = append(profiles, "cpu")
	}
	if p.memEnabled {
		profiles = append(profiles, "memory")
	}
	if p.traceEnabled {
		profiles = append(profiles, "trace")
	}
	if p.statsEnabled {
		profiles = append(profiles, "stats")
	}
	return profiles
}
