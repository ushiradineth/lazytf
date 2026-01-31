package testutil

import (
	"testing"
)

func TestAssertHeight(t *testing.T) {
	result := &RenderResult{
		Lines:     []string{"a", "b", "c"},
		LineCount: 3,
	}

	// Should pass - no error expected
	result.AssertHeight(t, 3)
}

func TestAssertHeightAtMost(t *testing.T) {
	result := &RenderResult{
		Lines:     []string{"a", "b", "c"},
		LineCount: 3,
	}

	// Should pass when under limit
	result.AssertHeightAtMost(t, 5)

	// Should pass at exactly limit
	result.AssertHeightAtMost(t, 3)
}

func TestAssertHeightAtLeast(t *testing.T) {
	result := &RenderResult{
		Lines:     []string{"a", "b", "c"},
		LineCount: 3,
	}

	// Should pass when at limit
	result.AssertHeightAtLeast(t, 3)

	// Should pass when above limit
	result.AssertHeightAtLeast(t, 2)
}

func TestAssertContains(t *testing.T) {
	result := &RenderResult{
		Plain: "hello world",
	}

	// Should pass for present substring
	result.AssertContains(t, "hello")
	result.AssertContains(t, "world")
	result.AssertContains(t, "llo wor")
}

func TestAssertNotContains(t *testing.T) {
	result := &RenderResult{
		Plain: "hello world",
	}

	// Should pass for missing substring
	result.AssertNotContains(t, "goodbye")
	result.AssertNotContains(t, "xyz")
}

func TestAssertHasBorder(t *testing.T) {
	withBorder := &RenderResult{
		Plain: "╭───╮\n│   │\n╰───╯",
	}

	// Should pass for bordered content
	withBorder.AssertHasBorder(t)

	// Also works with simple ASCII borders
	asciiBorder := &RenderResult{
		Plain: "+---+\n|   |\n+---+",
	}
	asciiBorder.AssertHasBorder(t)
}

func TestAssertHasRoundedBorder(t *testing.T) {
	withRounded := &RenderResult{
		Plain: "╭───╮\n│   │\n╰───╯",
	}

	// Should pass for rounded border
	withRounded.AssertHasRoundedBorder(t)
}

func TestAssertHasScrollbar(t *testing.T) {
	withScrollbar := &RenderResult{
		Plain: "content▐more",
	}

	// Should pass when scrollbar present
	withScrollbar.AssertHasScrollbar(t)
}

func TestAssertNoScrollbar(t *testing.T) {
	noScrollbar := &RenderResult{
		Plain: "just text without scrollbar",
	}

	// Should pass when no scrollbar
	noScrollbar.AssertNoScrollbar(t)
}

func TestAssertNoLineOverflow(t *testing.T) {
	result := &RenderResult{
		Lines:     []string{"short", "medium text"},
		LineCount: 2,
		Width:     20,
	}

	// Should pass for lines within width
	result.AssertNoLineOverflow(t)
}

func TestAssertMaxWidth(t *testing.T) {
	result := &RenderResult{
		Lines:     []string{"short", "longer line"},
		LineCount: 2,
	}

	// Should pass when max width is sufficient
	result.AssertMaxWidth(t, 20)
	result.AssertMaxWidth(t, 11) // "longer line" is 11 chars
}

func TestAssertMinWidth(t *testing.T) {
	result := &RenderResult{
		Lines:        []string{"short", "much longer line here"},
		LineCount:    2,
		MaxLineWidth: 21,
	}

	// Should pass when min width is met
	result.AssertMinWidth(t, 5)
	result.AssertMinWidth(t, 21)
}

func TestAssertContainsAll(t *testing.T) {
	result := &RenderResult{
		Plain: "hello world foo bar",
	}

	// Should pass when all present
	result.AssertContainsAll(t, "hello", "world", "foo")
}

func TestAssertContainsAny(t *testing.T) {
	result := &RenderResult{
		Plain: "hello world",
	}

	// Should pass when at least one present
	result.AssertContainsAny(t, "hello", "missing")
	result.AssertContainsAny(t, "missing", "world")
}

func TestAssertLineContains(t *testing.T) {
	result := &RenderResult{
		Lines:     []string{"first line", "second line"},
		LineCount: 2,
	}

	// Should pass when line contains substring
	result.AssertLineContains(t, 0, "first")
	result.AssertLineContains(t, 1, "second")
}

func TestAssertEmpty(t *testing.T) {
	empty := &RenderResult{
		Plain: "   \n\t  ",
	}

	// Should pass for whitespace-only content
	empty.AssertEmpty(t)

	emptyString := &RenderResult{
		Plain: "",
	}
	emptyString.AssertEmpty(t)
}

func TestAssertNotEmpty(t *testing.T) {
	notEmpty := &RenderResult{
		Plain: "text",
	}

	// Should pass for non-empty content
	notEmpty.AssertNotEmpty(t)
}

func TestAssertHasANSI(t *testing.T) {
	withANSI := &RenderResult{
		Raw: "\x1b[31mred text\x1b[0m",
	}

	// Should pass when ANSI codes present
	withANSI.AssertHasANSI(t)
}

func TestAssertRawContains(t *testing.T) {
	result := &RenderResult{
		Raw: "\x1b[31mred text\x1b[0m",
	}

	// Should pass when raw contains substring
	result.AssertRawContains(t, "red text")
	result.AssertRawContains(t, "\x1b[31m")
}

func TestAssertChaining(t *testing.T) {
	result := &RenderResult{
		Plain:     "hello world\nsecond line",
		Raw:       "\x1b[32mhello world\x1b[0m\nsecond line",
		Lines:     []string{"hello world", "second line"},
		LineCount: 2,
		Width:     80,
	}

	// Test fluent API chaining
	result.
		AssertContains(t, "hello").
		AssertContains(t, "world").
		AssertNotContains(t, "missing").
		AssertHeight(t, 2).
		AssertHeightAtMost(t, 5).
		AssertNotEmpty(t)
}
