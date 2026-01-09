package parser

import "testing"

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
