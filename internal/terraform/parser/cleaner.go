package parser

import (
	"regexp"
	"strings"
)

// Cleaner strips ANSI escape sequences from terraform output.
type Cleaner struct {
	ansiRegex    *regexp.Regexp
	oscRegex     *regexp.Regexp
	apcRegex     *regexp.Regexp
	spinnerRegex *regexp.Regexp
}

// NewCleaner creates a new ANSI cleaner.
func NewCleaner() *Cleaner {
	return &Cleaner{
		ansiRegex:    regexp.MustCompile(`\x1b\[[0-9;]*[mGKHF]`),
		oscRegex:     regexp.MustCompile(`\x1b\][^\x07]*?(?:\x07|\x1b\\)`),
		apcRegex:     regexp.MustCompile(`\x1b_[^\x1b]*\x1b\\`),
		spinnerRegex: regexp.MustCompile(`\r[\|/\\-]`),
	}
}

// StripANSI removes ANSI escape sequences from the input string.
func (c *Cleaner) StripANSI(input string) string {
	if c == nil || input == "" {
		return input
	}

	out := c.oscRegex.ReplaceAllString(input, "")
	out = c.apcRegex.ReplaceAllString(out, "")
	out = c.ansiRegex.ReplaceAllString(out, "")
	return out
}

// Normalize strips ANSI and removes control/spinner artifacts.
func (c *Cleaner) Normalize(input string) string {
	out := c.StripANSI(input)
	out = c.spinnerRegex.ReplaceAllString(out, "")
	out = strings.Map(func(r rune) rune {
		switch r {
		case '\n':
			return r
		case 9:
			return r
		default:
			if r < 32 {
				return -1
			}
			return r
		}
	}, out)
	return out
}
