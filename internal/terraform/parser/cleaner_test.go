package parser

import (
	"strings"
	"testing"
)

func TestCleanerStripANSI(t *testing.T) {
	cleaner := NewCleaner()
	input := "\x1b[31mred\x1b[0m"
	output := cleaner.StripANSI(input)
	if output != "red" {
		t.Fatalf("expected stripped output, got %q", output)
	}
}

func TestCleanerNormalizeRemovesControl(t *testing.T) {
	cleaner := NewCleaner()
	input := "title\x1b]0;window\x07\r/ok"
	output := cleaner.Normalize(input)
	if output != "titleok" {
		t.Fatalf("expected normalized output, got %q", output)
	}
}

func TestCleanerStripsOSCAndAPC(t *testing.T) {
	cleaner := NewCleaner()
	input := "hello\x1b]0;title\x07world\x1b_ignored\x1b\\done"
	output := cleaner.Normalize(input)
	if output != "helloworlddone" {
		t.Fatalf("expected OSC/APC stripped, got %q", output)
	}
}

func TestCleanerPreservesLineBreaksAndTabs(t *testing.T) {
	cleaner := NewCleaner()
	input := "line1\n\tline2"
	output := cleaner.Normalize(input)
	if output != input {
		t.Fatalf("expected line breaks and tabs preserved, got %q", output)
	}
}

func TestCleanerSpinnerRemoval(t *testing.T) {
	cleaner := NewCleaner()
	input := "start\r/working\r-done"
	output := cleaner.Normalize(input)
	if output != "startworkingdone" {
		t.Fatalf("expected spinner removed, got %q", output)
	}
}

func TestCleanerMalformedEscape(t *testing.T) {
	cleaner := NewCleaner()
	input := "text\x1b["
	output := cleaner.Normalize(input)
	if output == "" {
		t.Fatalf("expected malformed escape to be handled gracefully")
	}
}

func TestCleanerMalformedOscApcTerminators(t *testing.T) {
	cleaner := NewCleaner()
	input := "keep\x1b]0;title\x1b\\more\x1b_foo\x07tail"
	output := cleaner.Normalize(input)
	if output != "keepmore_footail" {
		t.Fatalf("expected malformed terminators stripped safely, got %q", output)
	}
}

func TestCleanerMalformedEscapeKeepsNearbyText(t *testing.T) {
	cleaner := NewCleaner()
	input := "keep\x1b[31mred\x1b[broken tail"
	output := cleaner.Normalize(input)
	if !strings.Contains(output, "keep") || !strings.Contains(output, "red") {
		t.Fatalf("expected nearby text preserved, got %q", output)
	}
}
