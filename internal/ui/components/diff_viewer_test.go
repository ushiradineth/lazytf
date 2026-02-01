package components

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestRenderInlineChangeReplaceMarker(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	change := &terraform.Change{
		ReplacePaths: [][]string{{"allocated_storage"}},
	}
	item := diff.MinimalDiff{
		Path:     []string{"allocated_storage"},
		OldValue: 50,
		NewValue: 100,
		Action:   diff.DiffChange,
	}

	out := viewer.renderInlineChange(item, change)
	if strings.Count(out, "forces replacement") != 1 {
		t.Fatalf("expected single replace marker, got %q", out)
	}
	if !strings.Contains(out, "allocated_storage: 50") {
		t.Fatalf("expected path/value in output, got %q", out)
	}
}

func TestBuildContextDiffTrimsTrailingNewline(t *testing.T) {
	before := "a: 1\nb: 2\n"
	after := "a: 1\nb: 3\n"
	lines := buildContextDiff(before, after, 1)
	for _, line := range lines {
		if line == "" {
			t.Fatalf("unexpected empty diff line: %#v", lines)
		}
	}
}

func TestBuildContextDiffAddsGapMarker(t *testing.T) {
	before := "one\ntwo\nthree\nfour\nfive"
	after := "one\nTWO\nthree\nfour\nFIVE"
	lines := buildContextDiff(before, after, 0)
	hasGap := false
	for _, line := range lines {
		if strings.TrimSpace(line) == consts.GapMarker {
			hasGap = true
			break
		}
	}
	if !hasGap {
		t.Fatalf("expected gap marker in diff: %#v", lines)
	}
}

func TestContextDiffGapMarkerOnlyOnceBetweenSpans(t *testing.T) {
	before := "one\ntwo\nthree\nfour\nfive\nsix"
	after := "ONE\ntwo\nthree\nfour\nFIVE\nsix"
	lines := buildContextDiff(before, after, 0)
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == consts.GapMarker {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected single gap marker, got %d: %#v", count, lines)
	}
}

func TestRenderHeaderIncludesActionLabel(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionUpdate,
	}
	out := viewer.renderHeader(resource, nil)
	if !strings.Contains(out, "update:") {
		t.Fatalf("expected 'update:' label in header: %q", out)
	}
}

func TestRenderHeaderActionLabels(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	tests := []struct {
		action terraform.ActionType
		label  string
	}{
		{terraform.ActionCreate, "create:"},
		{terraform.ActionUpdate, "update:"},
		{terraform.ActionDelete, "destroy:"},
		{terraform.ActionReplace, "replace:"},
	}

	for _, tt := range tests {
		resource := &terraform.ResourceChange{Address: "x", Action: tt.action}
		out := viewer.renderHeader(resource, nil)
		if !strings.Contains(out, tt.label) {
			t.Fatalf("expected %s in header for %s: %q", tt.label, tt.action, out)
		}
	}
}

func TestColumnWidthsDefaultsAndSum(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	if got := viewer.columnWidths(); len(got) != 4 || got[0] != 2 || got[1] != 18 {
		t.Fatalf("unexpected default column widths: %#v", got)
	}

	viewer.SetSize(120, 10)
	got := viewer.columnWidths()
	sum := 0
	for _, w := range got {
		sum += w
	}
	if sum != 120 {
		t.Fatalf("expected columns to sum to width, got %d", sum)
	}
	if got[1] < 16 || got[2] < 14 || got[3] < 14 {
		t.Fatalf("expected minimum column widths, got %#v", got)
	}
}

func TestRenderRowTruncatesColumns(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	columns := []int{2, 6, 6, 6}
	out := viewer.renderRow(columns, styles.DefaultStyles().DiffAdd, "+++", "verylongpath", "beforevalue", "aftervalue")
	out = stripANSIDiffViewer(out)
	if !strings.Contains(out, "...") {
		t.Fatalf("expected truncated content in output, got %q", out)
	}
}

func TestRenderDiffRowIncludesValues(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(40, 10)
	columns := viewer.columnWidths()
	item := diff.MinimalDiff{
		Path:     []string{"name"},
		NewValue: "value",
		Action:   diff.DiffAdd,
	}
	out := viewer.renderDiffRow(columns, item)
	out = stripANSIDiffViewer(out)
	if !strings.Contains(out, "name") || !strings.Contains(out, "value") {
		t.Fatalf("expected path and value in output, got %q", out)
	}
}

func TestRenderTableLineCount(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	diffs := []diff.MinimalDiff{
		{Path: []string{"a"}, OldValue: 1, NewValue: 2, Action: diff.DiffChange},
		{Path: []string{"b"}, OldValue: "x", NewValue: "y", Action: diff.DiffChange},
	}
	out := viewer.renderTable(diffs, nil)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(lines))
	}
}

func TestTruncateMiddle(t *testing.T) {
	got := truncateMiddle("abcdefghij", 6)
	if !strings.Contains(got, "...") || !strings.HasPrefix(got, "abc") || !strings.HasSuffix(got, "j") {
		t.Fatalf("unexpected truncated output: %q", got)
	}
}

func TestFormatSingleLineValueEscapesNewlines(t *testing.T) {
	got := formatSingleLineValue("a\nb")
	if !strings.Contains(got, `\n`) {
		t.Fatalf("expected newline escape, got %q", got)
	}
	if !strings.HasPrefix(got, "\"") || !strings.HasSuffix(got, "\"") {
		t.Fatalf("expected quoted string, got %q", got)
	}
}

func stripANSIDiffViewer(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func TestReplaceMarkerOnlyForMatchingPath(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	change := &terraform.Change{
		ReplacePaths: [][]string{{"a", "b"}},
	}
	match := diff.MinimalDiff{Path: []string{"a", "b"}, Action: diff.DiffChange, OldValue: 1, NewValue: 2}
	nonMatch := diff.MinimalDiff{Path: []string{"a", "c"}, Action: diff.DiffChange, OldValue: 1, NewValue: 2}

	matchOut := viewer.renderInlineChange(match, change)
	nonMatchOut := viewer.renderInlineChange(nonMatch, change)
	if strings.Count(matchOut, "forces replacement") != 1 {
		t.Fatalf("expected replace marker for match, got %q", matchOut)
	}
	if strings.Contains(nonMatchOut, "forces replacement") {
		t.Fatalf("did not expect replace marker for non-match, got %q", nonMatchOut)
	}
}

func TestMultilineBlockHeaderIncludesReplaceMarker(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	change := &terraform.Change{
		ReplacePaths: [][]string{{"values[0]"}},
	}
	item := diff.MinimalDiff{
		Path:     []string{"values[0]"},
		Action:   diff.DiffChange,
		OldValue: "a: 1\nb: 2\n",
		NewValue: "a: 1\nb: 3\n",
	}

	lines := viewer.renderMultilineBlock(item, change)
	if len(lines) == 0 || !strings.Contains(lines[0], "forces replacement") {
		t.Fatalf("expected replace marker in header, got %#v", lines)
	}
}

func TestContextDiffPreservesIndentation(t *testing.T) {
	before := "  a:\n    b: 1\n"
	after := "  a:\n    b: 2\n"
	lines := buildContextDiff(before, after, 1)
	foundIndented := false
	for _, line := range lines {
		if strings.HasPrefix(line, "-     b: 1") || strings.HasPrefix(line, "+     b: 2") {
			foundIndented = true
			break
		}
	}
	if !foundIndented {
		t.Fatalf("expected indentation to be preserved: %#v", lines)
	}
	last := lines[len(lines)-1]
	if last == "" {
		t.Fatalf("unexpected blank trailing line: %#v", lines)
	}
}

func TestCompactListFormatForCreateDelete(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	diffs := []diff.MinimalDiff{
		{Path: []string{"name"}, Action: diff.DiffAdd, NewValue: "x"},
		{Path: []string{"id"}, Action: diff.DiffRemove, OldValue: "y"},
	}
	out := viewer.renderCompactList(diffs, nil)
	if !strings.Contains(out, "name:") || !strings.Contains(out, "id:") {
		t.Fatalf("expected inline path formatting, got %q", out)
	}
	if strings.Contains(out, "Path") || strings.Contains(out, "Old value") {
		t.Fatalf("unexpected table headers in compact list: %q", out)
	}
}

func TestMultilineBlockSpacingSingleBlank(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	diffs := []diff.MinimalDiff{
		{Path: []string{"values[0]"}, Action: diff.DiffChange, OldValue: "a\n", NewValue: "b\n"},
		{Path: []string{"set[0]"}, Action: diff.DiffChange, OldValue: "x", NewValue: "y"},
	}
	out := viewer.renderBlocks(diffs, nil)
	if strings.Contains(out, "\n\n\n") {
		t.Fatalf("unexpected double blank lines: %q", out)
	}
}

func TestInlineReplaceMarkerTruncationSingle(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(40, 5)
	change := &terraform.Change{ReplacePaths: [][]string{{"name"}}}
	item := diff.MinimalDiff{
		Path:     []string{"name"},
		Action:   diff.DiffChange,
		OldValue: strings.Repeat("a", 50),
		NewValue: strings.Repeat("b", 50),
	}
	out := viewer.renderInlineChange(item, change)
	if strings.Count(out, "forces replacement") != 1 {
		t.Fatalf("expected single marker with truncation, got %q", out)
	}
}

func TestInlineReplaceMarkerNonASCIIPlacement(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	change := &terraform.Change{ReplacePaths: [][]string{{"name"}}}
	item := diff.MinimalDiff{
		Path:     []string{"name"},
		Action:   diff.DiffChange,
		OldValue: "café",
		NewValue: "café",
	}
	out := viewer.renderInlineChange(item, change)
	if !strings.Contains(out, "café") || !strings.Contains(out, "forces replacement") {
		t.Fatalf("expected value and marker, got %q", out)
	}
	if !strings.Contains(out, "\"café\" → \"café\"   # forces replacement") {
		t.Fatalf("expected marker after value, got %q", out)
	}
}

func TestInlineUnknownValueRendersKnownAfterApply(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	item := diff.MinimalDiff{
		Path:     []string{"id"},
		Action:   diff.DiffChange,
		OldValue: "x",
		NewValue: diff.UnknownValue{},
	}
	out := viewer.renderInlineChange(item, nil)
	if !strings.Contains(out, "(known after apply)") {
		t.Fatalf("expected known after apply, got %q", out)
	}
}

func TestInlineMapAndListValues(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	item := diff.MinimalDiff{
		Path:     []string{"tags"},
		Action:   diff.DiffChange,
		OldValue: map[string]any{"a": "b"},
		NewValue: []any{"x", "y"},
	}
	out := viewer.renderInlineChange(item, nil)
	if !strings.Contains(out, "{...}") || !strings.Contains(out, "[...]") {
		t.Fatalf("expected map/list placeholders, got %q", out)
	}
}

func TestInlineLongStringTruncation(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	long := strings.Repeat("a", 260)
	item := diff.MinimalDiff{
		Path:     []string{"desc"},
		Action:   diff.DiffChange,
		OldValue: long,
		NewValue: long,
	}
	out := viewer.renderInlineChange(item, nil)
	if !strings.Contains(out, "→") || !strings.Contains(out, "...") {
		t.Fatalf("expected arrow and truncation, got %q", out)
	}
}

func TestContextDiffDifferentLineCounts(t *testing.T) {
	before := "a\nb\nc"
	after := "a\nb\nc\nd"
	lines := buildContextDiff(before, after, 1)
	foundAdd := false
	for _, line := range lines {
		if strings.HasPrefix(line, "+ d") {
			foundAdd = true
			break
		}
	}
	if !foundAdd {
		t.Fatalf("expected added line for differing counts: %#v", lines)
	}
}

func TestContextDiffTrailingWhitespacePreserved(t *testing.T) {
	before := "a  \n"
	after := "b  \n"
	lines := buildContextDiff(before, after, 0)
	found := false
	for _, line := range lines {
		if strings.HasSuffix(line, "  ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected trailing whitespace preserved: %#v", lines)
	}
}

func TestContextDiffTabsPreserved(t *testing.T) {
	before := "\tkey: 1\n"
	after := "\tkey: 2\n"
	lines := buildContextDiff(before, after, 0)
	found := false
	for _, line := range lines {
		if strings.Contains(line, "\tkey") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tabs preserved: %#v", lines)
	}
}

func TestContextDiffLastLineNoGap(t *testing.T) {
	before := "a\nb\nc"
	after := "a\nb\nC"
	lines := buildContextDiff(before, after, 1)
	for _, line := range lines {
		if strings.TrimSpace(line) == "..." {
			t.Fatalf("did not expect gap marker: %#v", lines)
		}
	}
}

func TestContextDiffNoChangesPlaceholder(t *testing.T) {
	lines := buildContextDiff("a\nb", "a\nb", 1)
	if len(lines) != 1 || !strings.Contains(lines[0], "no diff") {
		t.Fatalf("expected no diff placeholder, got %#v", lines)
	}
}

func TestReplaceMarkerNestedPathMatch(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	change := &terraform.Change{ReplacePaths: [][]string{{"network", "self_link"}}}
	item := diff.MinimalDiff{
		Path:     []string{"network", "self_link", "value"},
		Action:   diff.DiffChange,
		OldValue: "a",
		NewValue: "b",
	}
	out := viewer.renderInlineChange(item, change)
	if !strings.Contains(out, "forces replacement") {
		t.Fatalf("expected replace marker for nested path, got %q", out)
	}
}

func TestReplaceMarkerListIndexMatch(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	change := &terraform.Change{ReplacePaths: [][]string{{"node_locations[1]"}}}
	item := diff.MinimalDiff{
		Path:     []string{"node_locations[1]"},
		Action:   diff.DiffChange,
		OldValue: "a",
		NewValue: "b",
	}
	out := viewer.renderInlineChange(item, change)
	if !strings.Contains(out, "forces replacement") {
		t.Fatalf("expected replace marker for list index, got %q", out)
	}
}

func TestReplaceMarkerMultiplePaths(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	change := &terraform.Change{ReplacePaths: [][]string{{"a"}, {"b"}}}
	first := diff.MinimalDiff{Path: []string{"a"}, Action: diff.DiffChange, OldValue: 1, NewValue: 2}
	second := diff.MinimalDiff{Path: []string{"b"}, Action: diff.DiffChange, OldValue: 1, NewValue: 2}
	if !strings.Contains(viewer.renderInlineChange(first, change), "forces replacement") {
		t.Fatalf("expected marker for first path")
	}
	if !strings.Contains(viewer.renderInlineChange(second, change), "forces replacement") {
		t.Fatalf("expected marker for second path")
	}
}

func TestMultilineHeaderMarkerWithTruncation(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(20, 5)
	change := &terraform.Change{ReplacePaths: [][]string{{"values[0]"}}}
	item := diff.MinimalDiff{
		Path:     []string{"values[0]"},
		Action:   diff.DiffChange,
		OldValue: "a: 1\n",
		NewValue: "a: 2\n",
	}
	lines := viewer.renderMultilineBlock(item, change)
	if !strings.Contains(strings.Join(lines, "\n"), "forces replacement") {
		t.Fatalf("expected marker in truncated header: %#v", lines)
	}
}

func TestMultilineHeaderMarkerStyled(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	change := &terraform.Change{ReplacePaths: [][]string{{"values[0]"}}}
	item := diff.MinimalDiff{
		Path:     []string{"values[0]"},
		Action:   diff.DiffChange,
		OldValue: "a: 1\n",
		NewValue: "a: 2\n",
	}
	lines := viewer.renderMultilineBlock(item, change)
	if len(lines) == 0 {
		t.Fatalf("expected header line")
	}
	header := lines[0]
	if !strings.Contains(header, "forces replacement") {
		t.Fatalf("expected marker in header: %q", header)
	}
}

func TestRenderBlocksNoTrailingBlankLine(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	diffs := []diff.MinimalDiff{
		{Path: []string{"values[0]"}, Action: diff.DiffChange, OldValue: "a\n", NewValue: "b\n"},
	}
	out := viewer.renderBlocks(diffs, nil)
	if strings.HasSuffix(out, "\n") {
		t.Fatalf("unexpected trailing newline: %q", out)
	}
}

func TestRenderBlocksSingleBlankBetweenRoots(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	diffs := []diff.MinimalDiff{
		{Path: []string{"values[0]"}, Action: diff.DiffChange, OldValue: "a\n", NewValue: "b\n"},
		{Path: []string{"set[0]"}, Action: diff.DiffChange, OldValue: "x", NewValue: "y"},
	}
	out := viewer.renderBlocks(diffs, nil)
	if !strings.Contains(out, "\n\n") {
		t.Fatalf("expected single blank line between blocks: %q", out)
	}
	if strings.Contains(out, "\n\n\n") {
		t.Fatalf("unexpected double blank lines: %q", out)
	}
}

func TestViewerNarrowWidthNoPanic(t *testing.T) {
	t.Helper()
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(10, 5)
	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionUpdate,
		Change: &terraform.Change{
			Actions: []string{"update"},
			Before:  map[string]any{"a": 1},
			After:   map[string]any{"a": 2},
		},
	}
	viewer.View(resource)
}

func TestViewNoResourceSelectedEmpty(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(20, 5)
	if got := viewer.View(nil); strings.TrimSpace(got) != "" {
		t.Fatalf("expected empty view, got %q", got)
	}
}

func TestRenderWithNilAfterMap(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)
	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionDelete,
		Change: &terraform.Change{
			Before: map[string]any{"name": "old"},
			After:  nil,
		},
	}
	out := viewer.View(resource)
	if !strings.Contains(out, "old") {
		t.Fatalf("expected before value to render")
	}
}

func TestReplaceMarkerMultipleIndexPaths(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	change := &terraform.Change{
		ReplacePaths: [][]string{
			{"ingress", "0", "cidr"},
			{"ingress", "1", "cidr"},
		},
	}
	item := diff.MinimalDiff{
		Path:     []string{"ingress", "1", "cidr"},
		OldValue: "0.0.0.0/0",
		NewValue: "10.0.0.0/8",
		Action:   diff.DiffChange,
	}
	line := viewer.renderInlineChange(item, change)
	if !strings.Contains(line, "forces replacement") {
		t.Fatalf("expected replace marker for list index path")
	}
}

func TestHasMultilineDiffTrue(t *testing.T) {
	diffs := []diff.MinimalDiff{
		{Path: []string{"simple"}, OldValue: 1, NewValue: 2, Action: diff.DiffChange},
		{Path: []string{"multi"}, OldValue: "a\nb", NewValue: "a\nc", Action: diff.DiffChange},
	}
	if !hasMultilineDiff(diffs) {
		t.Error("expected hasMultilineDiff to return true")
	}
}

func TestHasMultilineDiffFalse(t *testing.T) {
	diffs := []diff.MinimalDiff{
		{Path: []string{"a"}, OldValue: 1, NewValue: 2, Action: diff.DiffChange},
		{Path: []string{"b"}, OldValue: "x", NewValue: "y", Action: diff.DiffChange},
	}
	if hasMultilineDiff(diffs) {
		t.Error("expected hasMultilineDiff to return false")
	}
}

func TestIsMultilineChangeTrue(t *testing.T) {
	// isMultilineChange only returns true for DiffChange with multiline strings
	item := diff.MinimalDiff{Action: diff.DiffChange, OldValue: "a\nb", NewValue: "a\nc"}
	if !isMultilineChange(item) {
		t.Error("expected isMultilineChange to return true")
	}
}

func TestIsMultilineChangeFalse(t *testing.T) {
	item := diff.MinimalDiff{Action: diff.DiffChange, OldValue: "abc", NewValue: "def"}
	if isMultilineChange(item) {
		t.Error("expected isMultilineChange to return false")
	}
}

func TestIsMultilineChangeNonChangeAction(t *testing.T) {
	// isMultilineChange returns false for non-change actions
	addItem := diff.MinimalDiff{Action: diff.DiffAdd, NewValue: "a\nb"}
	if isMultilineChange(addItem) {
		t.Error("expected isMultilineChange to return false for add action")
	}

	removeItem := diff.MinimalDiff{Action: diff.DiffRemove, OldValue: "a\nb"}
	if isMultilineChange(removeItem) {
		t.Error("expected isMultilineChange to return false for remove action")
	}
}

func TestActionLabelCreate(t *testing.T) {
	result := actionLabel(terraform.ActionCreate)
	if result != "create" {
		t.Errorf("expected 'create', got %q", result)
	}
}

func TestActionLabelDelete(t *testing.T) {
	result := actionLabel(terraform.ActionDelete)
	if result != "destroy" {
		t.Errorf("expected 'destroy', got %q", result)
	}
}

func TestActionLabelUpdate(t *testing.T) {
	result := actionLabel(terraform.ActionUpdate)
	if result != "update" {
		t.Errorf("expected 'update', got %q", result)
	}
}

func TestActionLabelReplace(t *testing.T) {
	result := actionLabel(terraform.ActionReplace)
	if result != "replace" {
		t.Errorf("expected 'replace', got %q", result)
	}
}

func TestActionLabelNoop(t *testing.T) {
	// actionLabel returns empty string for NoOp
	result := actionLabel(terraform.ActionNoOp)
	if result != "" {
		t.Errorf("expected empty string for no-op, got %q", result)
	}
}

func TestActionLabelRead(t *testing.T) {
	// actionLabel returns empty string for Read
	result := actionLabel(terraform.ActionRead)
	if result != "" {
		t.Errorf("expected empty string for read, got %q", result)
	}
}

func TestRenderDiffRowRemove(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(60, 10)
	columns := viewer.columnWidths()
	item := diff.MinimalDiff{
		Path:     []string{"name"},
		OldValue: "oldvalue",
		Action:   diff.DiffRemove,
	}
	out := viewer.renderDiffRow(columns, item)
	out = stripANSIDiffViewer(out)
	if !strings.Contains(out, "name") {
		t.Errorf("expected path in output, got %q", out)
	}
}

func TestViewCreateAction(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)
	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionCreate,
		Change: &terraform.Change{
			Before: nil,
			After:  map[string]any{"name": "new"},
		},
	}
	out := viewer.View(resource)
	if !strings.Contains(out, "new") {
		t.Errorf("expected after value to render, got %q", out)
	}
}

func TestViewReplaceAction(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)
	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionReplace,
		Change: &terraform.Change{
			Before: map[string]any{"name": "old"},
			After:  map[string]any{"name": "new"},
		},
	}
	out := viewer.View(resource)
	if !strings.Contains(out, "replace") {
		t.Errorf("expected replace in header, got %q", out)
	}
}

func TestViewNoOpAction(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)
	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionNoOp,
		Change: &terraform.Change{
			Before: map[string]any{"name": "same"},
			After:  map[string]any{"name": "same"},
		},
	}
	out := viewer.View(resource)
	// No-op resources should still render something
	if out == "" {
		t.Error("expected some output for no-op action")
	}
}

func TestViewReadAction(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)
	resource := &terraform.ResourceChange{
		Address: "data.aws_ami.latest",
		Action:  terraform.ActionRead,
		Change: &terraform.Change{
			Before: nil,
			After:  map[string]any{"id": "ami-123"},
		},
	}
	out := viewer.View(resource)
	if !strings.Contains(out, "ami-123") {
		t.Errorf("expected after value to render, got %q", out)
	}
}

func TestDiffViewerPadWithSize(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(40, 10)
	result := viewer.pad("test content")
	if result == "" {
		t.Error("expected non-empty padded content")
	}
}

func TestDiffViewerPadNoSize(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	// No SetSize called
	result := viewer.pad("test content")
	if result != "test content" {
		t.Errorf("expected unchanged content without size, got %q", result)
	}
}

func TestRenderDiffRowDefaultCase(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(60, 10)
	columns := viewer.columnWidths()

	// Use an unknown action
	item := diff.MinimalDiff{
		Path:   []string{"field"},
		Action: diff.DiffAction("unknown"), // Unknown action
	}
	out := viewer.renderDiffRow(columns, item)
	if out == "" {
		t.Error("expected non-empty output for unknown action")
	}
}

func TestRenderInlineChangeDefaultCase(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(60, 10)

	// Use an unknown action
	item := diff.MinimalDiff{
		Path:   []string{"field"},
		Action: diff.DiffAction("unknown"), // Unknown action
	}
	out := viewer.renderInlineChange(item, nil)
	if !strings.Contains(out, "?") {
		t.Errorf("expected '?' prefix for unknown action, got %q", out)
	}
}

func TestViewNonNoOpActionWithNoDiffs(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)

	// Create a resource with non-no-op action but no Change data
	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionUpdate,
		Change:  nil, // No change data
	}
	out := viewer.View(resource)
	if !strings.Contains(out, "streaming mode") {
		t.Errorf("expected streaming mode message, got %q", out)
	}
}

func TestViewWithCustomActionLabel(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)

	// Use an action type that returns empty string from actionLabel
	resource := &terraform.ResourceChange{
		Address: "custom.resource",
		Action:  terraform.ActionType("custom"), // Custom action
		Change:  nil,
	}
	out := viewer.View(resource)
	// Should still render without panicking
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestLinePrefixAllCases(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"- removed", "-"},
		{"+ added", "+"},
		{"  ...", "."},
		{"  context", ""},
		{"plain text", ""},
	}

	for _, tt := range tests {
		got := linePrefix(tt.line)
		if got != tt.want {
			t.Errorf("linePrefix(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestPathMatchesReplaceEdgeCases(t *testing.T) {
	// Empty replace path
	if pathMatchesReplace([]string{"a"}, []string{}) {
		t.Error("expected false for empty replace path")
	}

	// Path shorter than replace
	if pathMatchesReplace([]string{"a"}, []string{"a", "b"}) {
		t.Error("expected false when path shorter than replace")
	}

	// Non-matching paths
	if pathMatchesReplace([]string{"a", "b"}, []string{"a", "c"}) {
		t.Error("expected false for non-matching paths")
	}
}

func TestDiffSpansEdgeCases(t *testing.T) {
	// Test merging adjacent spans
	indexes := []int{1, 2, 3}
	spans := diffSpans(indexes, 0, 5)
	if len(spans) != 1 {
		t.Errorf("expected 1 merged span, got %d", len(spans))
	}

	// Test extending existing span
	indexes2 := []int{0, 1}
	spans2 := diffSpans(indexes2, 1, 5)
	// With context=1, spans should merge
	if len(spans2) != 1 {
		t.Errorf("expected 1 span with context, got %d", len(spans2))
	}
}

func TestColumnWidthsSmallWidth(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(30, 10) // Small width

	cols := viewer.columnWidths()
	sum := 0
	for _, c := range cols {
		sum += c
	}
	if sum != 30 {
		t.Errorf("expected columns to sum to 30, got %d", sum)
	}
}

func TestRenderBlocksWithEmptyPath(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())

	diffs := []diff.MinimalDiff{
		{Path: []string{}, OldValue: "a", NewValue: "b", Action: diff.DiffChange},
	}
	out := viewer.renderBlocks(diffs, nil)
	if out == "" {
		t.Error("expected non-empty output for empty path")
	}
}

func TestTruncateMiddleEdgeCases(t *testing.T) {
	// maxLen <= 3
	result := truncateMiddle("abcdef", 3)
	if result != "abcdef" {
		t.Errorf("expected no truncation for maxLen=3, got %q", result)
	}

	// Short string
	result2 := truncateMiddle("ab", 10)
	if result2 != "ab" {
		t.Errorf("expected unchanged short string, got %q", result2)
	}
}

func TestRenderDiffRowChange(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(60, 10)
	columns := viewer.columnWidths()

	item := diff.MinimalDiff{
		Path:     []string{"field"},
		OldValue: "old",
		NewValue: "new",
		Action:   diff.DiffChange,
	}
	out := viewer.renderDiffRow(columns, item)
	out = stripANSIDiffViewer(out)
	if !strings.Contains(out, "old") || !strings.Contains(out, "new") {
		t.Errorf("expected old and new values in output, got %q", out)
	}
}

func TestViewWithMultilineDiff(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)

	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionUpdate,
		Change: &terraform.Change{
			Before: map[string]any{"script": "line1\nline2\n"},
			After:  map[string]any{"script": "line1\nline3\n"},
		},
	}
	out := viewer.View(resource)
	if out == "" {
		t.Error("expected non-empty output for multiline diff")
	}
}

func TestViewDeleteAction(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)

	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionDelete,
		Change: &terraform.Change{
			Before: map[string]any{"name": "deleted"},
			After:  nil,
		},
	}
	out := viewer.View(resource)
	if !strings.Contains(out, "destroy") {
		t.Errorf("expected 'destroy' label in header, got %q", out)
	}
}

func TestDiffViewerSetStyles(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	newStyles := styles.DefaultStyles()
	viewer.SetStyles(newStyles)

	if viewer.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestDiffViewerScrollUp(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(80, 10)
	viewer.scrollOffset = 5

	viewer.ScrollUp(2)
	if viewer.scrollOffset != 3 {
		t.Errorf("expected scrollOffset=3, got %d", viewer.scrollOffset)
	}

	// Scroll past top
	viewer.ScrollUp(10)
	if viewer.scrollOffset != 0 {
		t.Errorf("expected scrollOffset=0 after scrolling past top, got %d", viewer.scrollOffset)
	}
}

func TestDiffViewerScrollDown(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(80, 10)
	viewer.totalLines = 30
	viewer.scrollOffset = 0

	viewer.ScrollDown(5)
	if viewer.scrollOffset != 5 {
		t.Errorf("expected scrollOffset=5, got %d", viewer.scrollOffset)
	}

	// Scroll past maximum (totalLines - height = 20)
	viewer.ScrollDown(30)
	if viewer.scrollOffset != 20 {
		t.Errorf("expected scrollOffset=20 (max), got %d", viewer.scrollOffset)
	}
}

func TestDiffViewerScrollDownSmallContent(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(80, 20)
	viewer.totalLines = 5 // Less than height

	viewer.ScrollDown(10)
	// maxOffset should be 0 since content is smaller than viewport
	if viewer.scrollOffset != 0 {
		t.Errorf("expected scrollOffset=0 for small content, got %d", viewer.scrollOffset)
	}
}

func TestDiffViewerScrollToTop(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.scrollOffset = 15

	viewer.ScrollToTop()
	if viewer.scrollOffset != 0 {
		t.Errorf("expected scrollOffset=0, got %d", viewer.scrollOffset)
	}
}

func TestDiffViewerScrollToBottom(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(80, 10)
	viewer.totalLines = 30

	viewer.ScrollToBottom()
	// maxOffset = 30 - 10 = 20
	if viewer.scrollOffset != 20 {
		t.Errorf("expected scrollOffset=20, got %d", viewer.scrollOffset)
	}
}

func TestDiffViewerScrollToBottomSmallContent(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(80, 20)
	viewer.totalLines = 5 // Less than height

	viewer.ScrollToBottom()
	// maxOffset should be 0 since content is smaller than viewport
	if viewer.scrollOffset != 0 {
		t.Errorf("expected scrollOffset=0 for small content, got %d", viewer.scrollOffset)
	}
}

func TestDiffViewerResetScroll(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.scrollOffset = 25

	viewer.ResetScroll()
	if viewer.scrollOffset != 0 {
		t.Errorf("expected scrollOffset=0, got %d", viewer.scrollOffset)
	}
}

func TestDiffViewerGetScrollInfo(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(80, 15)
	viewer.scrollOffset = 5
	viewer.totalLines = 50

	offset, total, visible := viewer.GetScrollInfo()
	if offset != 5 {
		t.Errorf("expected offset=5, got %d", offset)
	}
	if total != 50 {
		t.Errorf("expected total=50, got %d", total)
	}
	if visible != 15 {
		t.Errorf("expected visible=15, got %d", visible)
	}
}
