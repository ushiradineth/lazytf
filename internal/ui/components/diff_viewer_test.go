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
	if !strings.Contains(got, "⏎") {
		t.Fatalf("expected newline marker, got %q", got)
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
	out := viewer.renderBlocks(diffs)
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

func TestNormalizeMultilineForDisplayDedentsSharedIndent(t *testing.T) {
	before := "        search:\n          shards: 5\n"
	after := "        search:\n          shards: 10\n"

	gotBefore, gotAfter := normalizeMultilineForDisplay(before, after)
	if strings.HasPrefix(gotBefore, " ") || strings.HasPrefix(gotAfter, " ") {
		t.Fatalf("expected first lines to be dedented, got before=%q after=%q", gotBefore, gotAfter)
	}
	if !strings.Contains(gotBefore, "  shards: 5") {
		t.Fatalf("expected relative indentation preserved in before, got %q", gotBefore)
	}
	if !strings.Contains(gotAfter, "  shards: 10") {
		t.Fatalf("expected relative indentation preserved in after, got %q", gotAfter)
	}
}

func TestNormalizeMultilineForDisplayHandlesEmpty(t *testing.T) {
	gotBefore, gotAfter := normalizeMultilineForDisplay("", "")
	if gotBefore != "" || gotAfter != "" {
		t.Fatalf("expected empty outputs, got before=%q after=%q", gotBefore, gotAfter)
	}
}

func TestShouldRenderRawFallback(t *testing.T) {
	resource := &terraform.ResourceChange{
		Address:     "aws_instance.web",
		Action:      terraform.ActionUpdate,
		ParseStatus: terraform.ParseStatusPartial,
		RawBlock:    "# aws_instance.web will be updated in-place",
	}
	if !shouldRenderRawFallback(resource, 3) {
		t.Fatal("expected raw fallback to be enabled for partial resource with raw block")
	}

	resource.ParseStatus = terraform.ParseStatusComplete
	if !shouldRenderRawFallback(resource, 0) {
		t.Fatal("expected raw fallback when diff count is zero and raw block exists")
	}

	if shouldRenderRawFallback(resource, 2) {
		t.Fatal("did not expect raw fallback for complete resource with material diffs")
	}
}

func TestRenderDestroyReasonUsesTerraformStyleComments(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	resource := &terraform.ResourceChange{
		Address: "module.edge_domain.module.users_service.kubernetes_service.service",
		Action:  terraform.ActionDelete,
		RawBlock: strings.Join([]string{
			"# module.edge_domain.module.users_service.kubernetes_service.service will be destroyed",
			"  # (because kubernetes_service.service is not in configuration)",
			"- resource \"kubernetes_service\" \"service\" {",
		}, "\n"),
	}

	out := stripANSIDiffViewer(viewer.renderDestroyReason(resource))
	if !strings.Contains(out, "will be destroyed") {
		t.Fatalf("expected destroy comment line, got %q", out)
	}
	if !strings.Contains(out, "because kubernetes_service.service is not in configuration") {
		t.Fatalf("expected because reason line, got %q", out)
	}
	if strings.Contains(out, "Sections affected") {
		t.Fatalf("did not expect compact summary sections, got %q", out)
	}
}

func TestRenderDestroyReasonFallbackWithoutRawBlock(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	resource := &terraform.ResourceChange{
		Address:      "module.edge_domain.module.users_service.kubernetes_service.service",
		Action:       terraform.ActionDelete,
		ResourceType: "kubernetes_service",
		ResourceName: "service",
	}

	out := stripANSIDiffViewer(viewer.renderDestroyReason(resource))
	if !strings.Contains(out, "# module.edge_domain.module.users_service.kubernetes_service.service will be destroyed") {
		t.Fatalf("expected fallback destroy line, got %q", out)
	}
	if !strings.Contains(out, "# (because kubernetes_service.service is not in configuration)") {
		t.Fatalf("expected fallback because line, got %q", out)
	}
}

func TestFormatSingleLineValueCompactsEscapedNewlines(t *testing.T) {
	got := formatSingleLineValue(`line1\nline2\tvalue`)
	if !strings.Contains(got, "⏎") {
		t.Fatalf("expected escaped newline marker, got %q", got)
	}
}

func TestFormatSingleLineValueCompactsDoubleEscapedNewlines(t *testing.T) {
	got := formatSingleLineValue(`line1\\nline2\\tvalue`)
	if strings.Contains(got, `\\n`) || strings.Contains(got, `\\t`) {
		t.Fatalf("expected escaped markers normalized, got %q", got)
	}
	if !strings.Contains(got, "⏎") {
		t.Fatalf("expected newline marker, got %q", got)
	}
}

func TestFormatSingleLineValueUnescapesQuotedBlob(t *testing.T) {
	got := formatSingleLineValue(`\"controller\":\\n  \"config\": true`)
	if strings.Contains(got, `\"`) {
		t.Fatalf("expected escaped quotes removed, got %q", got)
	}
	if strings.Contains(got, `\\n`) {
		t.Fatalf("expected escaped newline removed, got %q", got)
	}
	if !strings.Contains(got, `"controller":`) {
		t.Fatalf("expected readable quoted key, got %q", got)
	}
}

func TestIsMultilineChangeForAddEscapedBlob(t *testing.T) {
	item := diff.MinimalDiff{
		Path:     []string{"values", "__item_1"},
		Action:   diff.DiffAdd,
		NewValue: `\"controller\":\n  \"config\": true`,
	}
	if !isMultilineChange(item) {
		t.Fatalf("expected add blob to be treated as multiline")
	}
}

func TestViewRendersRawFallbackForPartialResource(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(120, 20)

	resource := &terraform.ResourceChange{
		Address:     "module.ops.kubernetes_config_map.config_map",
		Action:      terraform.ActionUpdate,
		ParseStatus: terraform.ParseStatusPartial,
		RawBlock: strings.Join([]string{
			"# module.ops.kubernetes_config_map.config_map will be updated in-place",
			"~ resource \"kubernetes_config_map\" \"config_map\" {",
			"    ~ metadata {",
			"    }",
			"}",
		}, "\n"),
	}

	out := stripANSIDiffViewer(viewer.View(resource))
	if !strings.Contains(out, "raw terraform block") {
		t.Fatalf("expected raw fallback header, got %q", out)
	}
	if !strings.Contains(out, "~ metadata {") {
		t.Fatalf("expected raw block content, got %q", out)
	}
}

func TestViewDeleteShowsTerraformStyleDestroyReason(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(120, 20)

	resource := &terraform.ResourceChange{
		Address: "module.edge_domain.module.users_service.kubernetes_service.service",
		Action:  terraform.ActionDelete,
		RawBlock: strings.Join([]string{
			"# module.edge_domain.module.users_service.kubernetes_service.service will be destroyed",
			"  # (because kubernetes_service.service is not in configuration)",
			"- resource \"kubernetes_service\" \"service\" {",
		}, "\n"),
		Change: &terraform.Change{
			Before: map[string]any{"id": "demo-platform/admin-ui-service"},
			After:  nil,
		},
	}

	out := stripANSIDiffViewer(viewer.View(resource))
	if !strings.Contains(out, "(because kubernetes_service.service is not in configuration)") {
		t.Fatalf("expected terraform-style destroy reason, got %q", out)
	}
	if strings.Contains(out, "Sections affected") {
		t.Fatalf("did not expect compact destroy summary, got %q", out)
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
	out := viewer.renderBlocks(diffs)
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
	out := viewer.renderBlocks(diffs)
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

func TestRenderWithNilAfterMapDeleteShowsReason(t *testing.T) {
	engine := diff.NewEngine()
	viewer := NewDiffViewer(styles.DefaultStyles(), engine)
	viewer.SetSize(80, 20)
	resource := &terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionDelete,
		RawBlock: "# aws_instance.web will be destroyed\n" +
			"  # (because aws_instance.web is not in configuration)",
		Change: &terraform.Change{
			Before: map[string]any{"name": "old"},
			After:  nil,
		},
	}
	out := viewer.View(resource)
	if !strings.Contains(out, "will be destroyed") {
		t.Fatalf("expected destroy reason to render")
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
	// add/remove actions with multiline content should render as multiline blocks
	addItem := diff.MinimalDiff{Action: diff.DiffAdd, NewValue: "a\nb"}
	if !isMultilineChange(addItem) {
		t.Error("expected isMultilineChange to return true for add action with multiline content")
	}

	removeItem := diff.MinimalDiff{Action: diff.DiffRemove, OldValue: "a\nb"}
	if !isMultilineChange(removeItem) {
		t.Error("expected isMultilineChange to return true for remove action with multiline content")
	}

	plainAdd := diff.MinimalDiff{Action: diff.DiffAdd, NewValue: "single-line"}
	if isMultilineChange(plainAdd) {
		t.Error("expected isMultilineChange to return false for plain single-line add")
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
	if !strings.Contains(out, "details unavailable") {
		t.Errorf("expected details unavailable message, got %q", out)
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
	out := viewer.renderBlocks(diffs)
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

func TestBuildDiffHunksGroupsByRootPath(t *testing.T) {
	diffs := []diff.MinimalDiff{
		{Path: []string{"name"}, Action: diff.DiffChange, OldValue: "old", NewValue: "new"},
		{Path: []string{"tags", "env"}, Action: diff.DiffChange, OldValue: "dev", NewValue: "prod"},
		{Path: []string{"tags", "owner"}, Action: diff.DiffChange, OldValue: "ops", NewValue: "platform"},
	}

	hunks := buildDiffHunks(diffs)
	if len(hunks) != 2 {
		t.Fatalf("expected root + nested section hunks, got %d", len(hunks))
	}
	if hunks[0].title != "root" || len(hunks[0].diffs) != 3 {
		t.Fatalf("unexpected first hunk: %+v", hunks[0])
	}
	if hunks[1].title != "tags" || len(hunks[1].diffs) != 2 {
		t.Fatalf("unexpected second hunk: %+v", hunks[1])
	}
}

func TestBuildDiffHunksFlatPathsCreateSingleRootSection(t *testing.T) {
	diffs := []diff.MinimalDiff{
		{Path: []string{"atomic"}, Action: diff.DiffAdd, NewValue: false},
		{Path: []string{"chart"}, Action: diff.DiffAdd, NewValue: "nginx"},
		{Path: []string{"lint"}, Action: diff.DiffAdd, NewValue: false},
	}

	hunks := buildDiffHunks(diffs)
	if len(hunks) != 1 {
		t.Fatalf("expected single root section for flat paths, got %d", len(hunks))
	}
	if hunks[0].title != "root" {
		t.Fatalf("expected root section title, got %+v", hunks[0])
	}
}

func TestBuildDiffHunksListItemMultilineGetsOwnSection(t *testing.T) {
	diffs := []diff.MinimalDiff{
		{Path: []string{"values", "__item_0"}, Action: diff.DiffAdd, NewValue: "single-line"},
		{Path: []string{"values", "__item_1"}, Action: diff.DiffAdd, NewValue: `\"controller\":\n  \"config\": true`},
	}

	hunks := buildDiffHunks(diffs)
	if len(hunks) < 3 {
		t.Fatalf("expected root, values, and multiline list-item sections; got %d", len(hunks))
	}
	if hunks[1].title != "values" {
		t.Fatalf("expected values section, got %+v", hunks[1])
	}
	found := false
	for _, h := range hunks {
		if h.title == "[1]" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected multiline list item section [1], got %+v", hunks)
	}
}

func TestDiffViewerHunkNavigationAndToggle(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.hunks = []diffHunk{{key: "name"}, {key: "tags"}}
	viewer.hunkLineStarts = []int{0, 6}
	viewer.height = 4

	if moved := viewer.NextHunk(); !moved {
		t.Fatal("expected NextHunk to move")
	}
	if viewer.activeHunk != 1 {
		t.Fatalf("expected activeHunk=1, got %d", viewer.activeHunk)
	}
	if viewer.scrollOffset != 6 {
		t.Fatalf("expected scrollOffset=6, got %d", viewer.scrollOffset)
	}

	if toggled := viewer.ToggleCurrentHunk(); !toggled {
		t.Fatal("expected ToggleCurrentHunk to toggle")
	}
	if !viewer.foldedHunks["tags"] {
		t.Fatal("expected selected hunk to be folded")
	}

	if moved := viewer.PrevHunk(); !moved {
		t.Fatal("expected PrevHunk to move")
	}
	if viewer.activeHunk != 0 {
		t.Fatalf("expected activeHunk=0, got %d", viewer.activeHunk)
	}
}

func TestDiffViewerSelectOrToggleAtVisibleRowSelectsThenToggles(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.hunks = []diffHunk{{key: "name"}, {key: "tags"}}
	viewer.hunkLineStarts = []int{2, 5}
	viewer.activeHunk = 0
	viewer.scrollOffset = 2

	if acted := viewer.SelectOrToggleAtVisibleRow(3); !acted {
		t.Fatal("expected click on inactive hunk row to select")
	}
	if viewer.activeHunk != 1 {
		t.Fatalf("expected active hunk 1 after select, got %d", viewer.activeHunk)
	}
	if viewer.foldedHunks["tags"] {
		t.Fatal("expected selected hunk to remain expanded after first click")
	}

	if acted := viewer.SelectOrToggleAtVisibleRow(3); !acted {
		t.Fatal("expected click on active hunk row to toggle")
	}
	if !viewer.foldedHunks["tags"] {
		t.Fatal("expected active hunk to be folded after second click")
	}
}

func TestDiffViewerSelectOrToggleAtVisibleRowWithRenderedTree(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(120, 30)

	resource := &terraform.ResourceChange{
		Address: "aws_instance.example",
		Action:  terraform.ActionUpdate,
		Change: &terraform.Change{
			Before: map[string]any{
				"name": "old",
				"tags": map[string]any{"env": "dev"},
			},
			After: map[string]any{
				"name": "new",
				"tags": map[string]any{"env": "prod"},
			},
		},
	}

	_ = viewer.View(resource)
	if len(viewer.hunkLineStarts) < 2 {
		t.Fatalf("expected at least two tree hunks, got %d", len(viewer.hunkLineStarts))
	}

	secondHunkRow := viewer.hunkLineStarts[1] - viewer.scrollOffset
	if secondHunkRow < 0 {
		t.Fatalf("expected visible row for second hunk, got %d", secondHunkRow)
	}

	if acted := viewer.SelectOrToggleAtVisibleRow(secondHunkRow); !acted {
		t.Fatal("expected click on rendered second hunk to select")
	}
	current, _ := viewer.GetHunkInfo()
	if current != 2 {
		t.Fatalf("expected active hunk 2 after click, got %d", current)
	}

	if acted := viewer.SelectOrToggleAtVisibleRow(secondHunkRow); !acted {
		t.Fatal("expected second click on rendered hunk to toggle")
	}
	if !viewer.foldedHunks[viewer.hunks[1].key] {
		t.Fatal("expected rendered second hunk to be folded after second click")
	}
}

func TestRenderInlineChangeWithTrimRemovesParentPrefix(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	item := diff.MinimalDiff{
		Path:     []string{"metadata", "labels", "platform.domain"},
		Action:   diff.DiffChange,
		OldValue: "ops",
		NewValue: "ops_internal",
	}
	out := stripANSIDiffViewer(viewer.renderInlineChangeWithTrim(item, nil, 2))
	if !strings.Contains(out, `"platform.domain": "ops" → "ops_internal"`) {
		t.Fatalf("expected trimmed leaf path, got %q", out)
	}
	if strings.Contains(out, "metadata.labels") {
		t.Fatalf("expected parent prefix removed, got %q", out)
	}
}

func TestDiffViewerTreeParentAndChild(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.hunks = []diffHunk{
		{key: "(root)", title: "root", parentKey: ""},
		{key: "metadata", title: "metadata", parentKey: "(root)"},
		{key: "metadata/labels", title: "labels", parentKey: "metadata"},
	}
	viewer.hunkLineStarts = []int{0, 3, 6}
	viewer.foldedHunks = map[string]bool{}

	viewer.activeHunk = 1
	if moved := viewer.TreeChild(); !moved {
		t.Fatal("expected TreeChild to move to first child")
	}
	if viewer.activeHunk != 2 {
		t.Fatalf("expected active hunk 2, got %d", viewer.activeHunk)
	}

	if moved := viewer.TreeParent(); !moved {
		t.Fatal("expected TreeParent to move to parent")
	}
	if viewer.activeHunk != 1 {
		t.Fatalf("expected active hunk 1, got %d", viewer.activeHunk)
	}

	if moved := viewer.TreeParent(); !moved {
		t.Fatal("expected TreeParent to move to root")
	}
	if viewer.activeHunk != 0 {
		t.Fatalf("expected active hunk 0, got %d", viewer.activeHunk)
	}
	if viewer.foldedHunks["metadata"] {
		t.Fatal("did not expect metadata section to be collapsed by TreeParent")
	}
}

func TestRenderHeaderIncludesTreeInfo(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.hunks = []diffHunk{{key: "a"}, {key: "b"}}
	viewer.activeHunk = 1
	resource := &terraform.ResourceChange{Address: "aws_instance.web", Action: terraform.ActionUpdate}

	out := viewer.renderHeader(resource, []diff.MinimalDiff{{Path: []string{"name"}, Action: diff.DiffChange}})
	out = stripANSIDiffViewer(out)
	if !strings.Contains(out, "[tree 2/2]") {
		t.Fatalf("expected tree info in header, got %q", out)
	}
}

func TestExpandCompositeDiffsForDisplayFlattensMapAdd(t *testing.T) {
	diffs := []diff.MinimalDiff{{
		Path:   []string{"metadata"},
		Action: diff.DiffAdd,
		NewValue: map[string]any{
			"labels": map[string]any{"team": "ops"},
		},
	}}

	expanded := expandCompositeDiffsForDisplay(diffs)
	if len(expanded) != 1 {
		t.Fatalf("expected 1 leaf diff, got %d", len(expanded))
	}
	if got := strings.Join(expanded[0].Path, "."); got != "metadata.labels.team" {
		t.Fatalf("expected flattened path metadata.labels.team, got %q", got)
	}
	if expanded[0].Action != diff.DiffAdd {
		t.Fatalf("expected add action, got %s", expanded[0].Action)
	}
}

func TestViewCreateExpandsNestedMetadataAndSpec(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(120, 24)

	resource := &terraform.ResourceChange{
		Address: "module.orders_domain.module.orders_service.kubernetes_service.service",
		Action:  terraform.ActionCreate,
		Change: &terraform.Change{
			Before: map[string]any{},
			After: map[string]any{
				"metadata": map[string]any{
					"name": "orders-service",
				},
				"spec": map[string]any{
					"type": "ClusterIP",
				},
				"wait_for_load_balancer": true,
			},
		},
	}

	out := stripANSIDiffViewer(viewer.View(resource))
	if !strings.Contains(out, "metadata") || !strings.Contains(out, "spec") {
		t.Fatalf("expected metadata/spec sections, got %q", out)
	}
	if !strings.Contains(out, `+ name: "orders-service"`) {
		t.Fatalf("expected nested metadata leaf, got %q", out)
	}
	if !strings.Contains(out, `+ type: "ClusterIP"`) {
		t.Fatalf("expected nested spec leaf, got %q", out)
	}
	if strings.Contains(out, "+ metadata: {...}") || strings.Contains(out, "+ spec: {...}") {
		t.Fatalf("did not expect collapsed top-level maps, got %q", out)
	}
}

func TestViewCreateRendersEscapedBlobAsMultilineBlock(t *testing.T) {
	viewer := NewDiffViewer(styles.DefaultStyles(), diff.NewEngine())
	viewer.SetSize(140, 30)

	resource := &terraform.ResourceChange{
		Address: "module.foundation.module.ingress.helm_release.release",
		Action:  terraform.ActionCreate,
		Change: &terraform.Change{
			Before: map[string]any{},
			After: map[string]any{
				"values": []any{
					"",
					`\"controller\":\n  \"config\":\n    \"compute-full-forwarded-for\": \"true\"`,
				},
			},
		},
	}

	out := stripANSIDiffViewer(viewer.View(resource))
	if !strings.Contains(out, "[1] (1)") {
		t.Fatalf("expected [1] tree section header, got %q", out)
	}
	if !strings.Contains(out, "-----") {
		t.Fatalf("expected multiline block separator under list item, got %q", out)
	}
	if !strings.Contains(out, `+ "controller":`) {
		t.Fatalf("expected multiline decoded content, got %q", out)
	}
	if strings.Contains(out, "⏎") {
		t.Fatalf("did not expect compact newline markers in multiline block, got %q", out)
	}
}
