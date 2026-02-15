package profile

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		wantCPU  bool
		wantMem  bool
		wantTrc  bool
		wantStat bool
	}{
		{"empty", "", false, false, false, false},
		{"cpu only", "cpu", true, false, false, false},
		{"mem only", "mem", false, true, false, false},
		{"memory alias", "memory", false, true, false, false},
		{"heap alias", "heap", false, true, false, false},
		{"trace only", "trace", false, false, true, false},
		{"stats only", "stats", false, false, false, true},
		{"all", "all", true, true, true, true},
		{"multiple", "cpu,mem,stats", true, true, false, true},
		{"with spaces", "cpu, mem, trace", true, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LAZYTF_PROFILE", tt.env)

			opts := ParseEnv()

			if opts.CPU != tt.wantCPU {
				t.Errorf("CPU = %v, want %v", opts.CPU, tt.wantCPU)
			}
			if opts.Memory != tt.wantMem {
				t.Errorf("Memory = %v, want %v", opts.Memory, tt.wantMem)
			}
			if opts.Trace != tt.wantTrc {
				t.Errorf("Trace = %v, want %v", opts.Trace, tt.wantTrc)
			}
			if opts.Stats != tt.wantStat {
				t.Errorf("Stats = %v, want %v", opts.Stats, tt.wantStat)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	opts := ParseFlags("cpu,mem")
	if !opts.CPU {
		t.Error("expected CPU to be enabled")
	}
	if !opts.Memory {
		t.Error("expected Memory to be enabled")
	}
	if opts.Trace {
		t.Error("expected Trace to be disabled")
	}
}

func TestProfilerStartStop(t *testing.T) {
	dir := t.TempDir()

	opts := Options{
		OutputDir: dir,
		Prefix:    "test",
		CPU:       true,
		Memory:    true,
		Stats:     true,
	}

	p := New(opts)

	if err := p.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Let it collect some stats.
	time.Sleep(100 * time.Millisecond)

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Check files were created.
	files, err := filepath.Glob(filepath.Join(dir, "test-*.prof"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) < 2 {
		t.Errorf("expected at least 2 profile files, got %d", len(files))
	}
}

func TestCurrentStats(t *testing.T) {
	stats := CurrentStats()

	if stats.HeapAlloc == 0 {
		t.Error("HeapAlloc should not be 0")
	}
	if stats.NumGoroutine == 0 {
		t.Error("NumGoroutine should not be 0")
	}
	if stats.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestFormatStats(t *testing.T) {
	s := FormatStats()
	if s == "" {
		t.Error("FormatStats should return non-empty string")
	}
	// Should contain key metrics.
	if !strings.Contains(s, "Heap:") {
		t.Error("should contain Heap info")
	}
	if !strings.Contains(s, "Goroutines:") {
		t.Error("should contain Goroutines info")
	}
}

func TestProfilerEnabledProfiles(t *testing.T) {
	opts := Options{
		CPU:    true,
		Memory: true,
	}
	p := New(opts)

	profiles := p.EnabledProfiles()
	if len(profiles) != 2 {
		t.Errorf("expected 2 enabled profiles, got %d", len(profiles))
	}
}

func TestNewUsesDefaultStatsInterval(t *testing.T) {
	p := New(Options{Stats: true})
	if p.statsInterval != time.Second {
		t.Fatalf("expected default stats interval %v, got %v", time.Second, p.statsInterval)
	}
}

func TestNewUsesCustomStatsInterval(t *testing.T) {
	interval := 250 * time.Millisecond
	p := New(Options{Stats: true, StatsInterval: interval})
	if p.statsInterval != interval {
		t.Fatalf("expected custom stats interval %v, got %v", interval, p.statsInterval)
	}
}
