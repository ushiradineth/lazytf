package components

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/utils"
)

func TestFormatValue(t *testing.T) {
	if got := formatValue(nil); got != "(null)" {
		t.Fatalf("expected null, got %q", got)
	}
	if got := formatValue("hi"); got != "\"hi\"" {
		t.Fatalf("expected quoted string, got %q", got)
	}
	if got := formatValue(diff.UnknownValue{}); got != "(known after apply)" {
		t.Fatalf("expected known after apply, got %q", got)
	}
	if got := formatValue(map[string]any{"a": 1}); got != "{...}" {
		t.Fatalf("expected map placeholder, got %q", got)
	}
	if got := formatValue([]any{"one"}); got != "\"one\"" {
		t.Fatalf("expected single string list to format to string, got %q", got)
	}
	if got := formatValue([]any{"one", "two"}); got != "[...]" {
		t.Fatalf("expected list placeholder, got %q", got)
	}
}

func TestFormatMultilineStringDiff(t *testing.T) {
	before := "- a\n- b\n- c"
	after := "- a\n- x\n- c"
	got := formatMultilineStringDiff("list", before, after)
	if got == "" {
		t.Fatalf("expected multiline diff")
	}
	if !strings.Contains(got, "~ list:") || !strings.Contains(got, "- b") || !strings.Contains(got, "+ x") {
		t.Fatalf("unexpected multiline diff: %q", got)
	}
}

func TestFormatMultilineStringDiff_MismatchOrNoDiff(t *testing.T) {
	if got := formatMultilineStringDiff("list", "a\nb", "a"); got != "" {
		t.Fatalf("expected empty diff for mismatched lines, got %q", got)
	}
	if got := formatMultilineStringDiff("list", "a\nb", "a\nb"); got != "" {
		t.Fatalf("expected empty diff for identical lines, got %q", got)
	}
}

func TestTruncateEndAndStripListMarker(t *testing.T) {
	if got := utils.TruncateEnd("hello", 3); got != "hel" {
		t.Fatalf("unexpected truncation: %q", got)
	}
	if got := utils.TruncateEnd("hello", 5); got != "hello" {
		t.Fatalf("unexpected truncation: %q", got)
	}
	if got := stripListMarker("- item"); got != "item" {
		t.Fatalf("unexpected strip result: %q", got)
	}
	if got := stripListMarker("item"); got != "item" {
		t.Fatalf("unexpected strip result: %q", got)
	}
}

func TestInterfaceHelpers(t *testing.T) {
	if !utils.IsMap(map[string]any{"a": 1}) {
		t.Fatalf("expected map to be detected")
	}
	if utils.IsMap(nil) {
		t.Fatalf("expected nil not to be a map")
	}
	if !utils.IsList([]int{1, 2}) {
		t.Fatalf("expected list to be detected")
	}
	if utils.IsList(nil) {
		t.Fatalf("expected nil not to be a list")
	}
	if got := utils.InterfaceToList([]string{"a", "b"}); len(got) != 2 {
		t.Fatalf("unexpected list length: %d", len(got))
	}
}

func TestResourceListFilteringAndSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := []terraform.ResourceChange{
		{Address: "aws_instance.a", Action: terraform.ActionCreate},
		{Address: "aws_instance.b", Action: terraform.ActionDelete},
	}
	r.SetResources(resources)
	r.SetFilter(terraform.ActionDelete, false)

	r.MoveDown()
	if got := r.GetSelectedResource(); got == nil || got.Address != "aws_instance.a" {
		t.Fatalf("unexpected selected resource: %#v", got)
	}

	r.SetFilter(terraform.ActionCreate, false)
	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected nil selection when filtered out, got %#v", got)
	}
}

func TestResourceListViewEmpty(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(40, 5)
	r.SetResources(nil)
	got := r.View()
	// Check that the view contains the expected text
	if !strings.Contains(got, "No resources") || !strings.Contains(got, "display") {
		t.Fatalf("unexpected empty view: %q", got)
	}
	// Check that it has a border/title (new panel format)
	if !strings.Contains(got, "Resources") {
		t.Fatalf("expected Resources title in view: %q", got)
	}
}

func TestResourceListShowStatusToggle(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	if r.ShowStatus() {
		t.Fatalf("expected status hidden by default")
	}
	r.SetShowStatus(true)
	if !r.ShowStatus() {
		t.Fatalf("expected status enabled")
	}
}

func TestResourceListRenderGroupAndVisibleItems(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(40, 5)
	group := r.renderGroup(consts.ModuleAlpha, 2, true, true, 2, 38)
	if !strings.Contains(group, consts.ModuleAlpha) {
		t.Fatalf("expected group label")
	}

	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionUpdate},
	})
	out := r.renderVisibleItems()
	if out == "" {
		t.Fatalf("expected visible items output")
	}
}

func TestResourceListStatusDisplayAndDuration(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	state := terraform.NewOperationState()
	state.StartResource("aws_instance.web", terraform.ActionCreate)
	state.CompleteResource("aws_instance.web", "id")
	r.SetOperationState(state)
	r.SetShowStatus(true)

	resource := terraform.ResourceChange{Address: "aws_instance.web", Action: terraform.ActionCreate}
	status, elapsed := r.getStatusDisplay(resource)
	if status == "" {
		t.Fatalf("expected status badge")
	}
	if elapsed == "" {
		t.Fatalf("expected elapsed text")
	}
	if formatShortDuration(500*time.Millisecond) == "" {
		t.Fatalf("expected short duration format")
	}
	if padAfterStyledWithBackground("x", 3, r.styles.SelectedLineBackground) == "" {
		t.Fatalf("expected padded line")
	}
}

func TestRenderResourceDoesNotExpandDiffs(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resource := terraform.ResourceChange{
		Address: "aws_vpc.main",
		Action:  terraform.ActionUpdate,
		Change: &terraform.Change{
			Before: map[string]any{"name": "old"},
			After:  map[string]any{"name": "new"},
		},
	}

	out := r.renderResource(&resource, false, 0, 80)
	if !strings.Contains(out, "aws_vpc.main") || strings.Contains(out, "~ name") {
		t.Fatalf("unexpected render output: %q", out)
	}
}

func TestRenderResourceTrimsModulePrefix(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resource := terraform.ResourceChange{
		Address: "module.alpha.aws_instance.web",
		Action:  terraform.ActionUpdate,
		Change:  &terraform.Change{Before: map[string]any{"a": 1}, After: map[string]any{"a": 2}},
	}

	out := r.renderResource(&resource, false, 2, 80)
	if !strings.Contains(out, "aws_instance.web") || strings.Contains(out, consts.ModuleAlpha) {
		t.Fatalf("expected trimmed module prefix, got %q", out)
	}
}

func TestRenderDiffUnknownAction(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	line := r.renderDiff(diff.MinimalDiff{Path: []string{"a"}, Action: diff.DiffAction("weird")})
	if !strings.Contains(line, "? a") {
		t.Fatalf("unexpected diff line: %q", line)
	}
}

func TestRenderDiffMultilineColorsAndPrefixes(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	multi := diff.MinimalDiff{
		Path:     []string{"values[0]"},
		Action:   diff.DiffChange,
		OldValue: "retry: true\nlog: info\n",
		NewValue: "retry: false\nlog: debug\n",
	}

	out := r.renderDiff(multi)
	plain := stripANSI(out)
	if !strings.Contains(plain, "~ values[0]:") {
		t.Fatalf("expected header, got %q", plain)
	}
	if !strings.Contains(plain, "- retry: true") || !strings.Contains(plain, "+ retry: false") {
		t.Fatalf("expected +/- lines, got %q", plain)
	}
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func selectFirstResource(r *ResourceList) *terraform.ResourceChange {
	for i := 0; i < 20; i++ {
		if res := r.GetSelectedResource(); res != nil {
			return res
		}
		r.MoveDown()
	}
	return nil
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func isSorted(list []string) bool {
	for i := 1; i < len(list); i++ {
		if list[i-1] > list[i] {
			return false
		}
	}
	return true
}

func indexOfGroup(items []listItem, label string) int {
	for i, item := range items {
		if item.kind == itemGroup && item.label == label {
			return i
		}
	}
	return -1
}

func TestFuzzyMatchNonContiguous(t *testing.T) {
	if !fuzzyMatch("awsins", "aws_instance") {
		t.Fatalf("expected non-contiguous match to pass")
	}
	if fuzzyMatch("azs", "aws_instance") {
		t.Fatalf("expected non-match to fail")
	}
}

func TestFuzzyMatchCaseInsensitive(t *testing.T) {
	if !fuzzyMatch("aws", strings.ToLower("AWS_INSTANCE")) {
		t.Fatalf("expected case-insensitive match to pass")
	}
}

func TestFuzzyMatchOrderRequired(t *testing.T) {
	// Test that characters must appear in order
	tests := []struct {
		query       string
		candidate   string
		shouldMatch bool
	}{
		// Should match: e-g-g appears in order
		{"egg", "gselllllglllawdaeaewdwag", true},
		// Should NOT match: only one 'g' after 'e'
		{"egg", "gawdhuaowdhoaweawdbiuawbdiauwbdg", false},
		// Should NOT match: 'e' comes after both 'g's
		{"egg", "gge", false},
		// Additional test cases
		{"abc", "aabbcc", true},
		{"abc", "acb", false}, // 'b' comes after 'c'
		{"xyz", "xaybzc", true},
		{"xyz", "xzy", false}, // 'y' comes after 'z'
		// Real-world terraform examples
		{"edge4", "module.gamma.kubernetes_config_map.settings_4", false},          // no 'd' after 'e' and before 'g'
		{"edge4", "module.zeta.aws_instance.node_4", false},                        // no 'g' in address
		{"edge4", "module.beta.module.db.aws_security_group.legacy_4", true},       // e(modul-e) d(mo-d-ule) g(security_-g-roup) e(l-e-gacy) 4(-4)
		{"edge4", "module.alpha.module.net.module.edge.aws_instance.node_4", true}, // has actual 'edge' substring
		{"edge", "module.alpha.module.net.module.edge.aws_instance.node_0", true},
	}

	for _, tt := range tests {
		result := fuzzyMatch(tt.query, tt.candidate)
		if result != tt.shouldMatch {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v",
				tt.query, tt.candidate, result, tt.shouldMatch)
		}
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("AWS")
	if got := selectFirstResource(r); got == nil {
		t.Fatalf("expected case-insensitive search to match")
	}
}

func TestSearchClearsSelectionWhenEmpty(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("nomatch")

	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected nil selection, got %#v", got)
	}
}

func TestSearchEmptyQueryRestoresSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.db", Action: terraform.ActionCreate},
	})
	r.selectedIndex = 1
	r.SetSearchQuery("nomatch")
	if r.selectedIndex != 0 {
		t.Fatalf("expected selection reset on empty list, got %d", r.selectedIndex)
	}
	r.SetSearchQuery("")
	if got := selectFirstResource(r); got == nil {
		t.Fatalf("expected selection restored to resource, got %#v", got)
	}
}

func TestSearchQueryTrimsWhitespace(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("  web ")
	if got := selectFirstResource(r); got == nil || got.Address != "aws_instance.web" {
		t.Fatalf("expected trimmed query to match, got %#v", got)
	}
}

func TestSearchQueryMatchesDotAndBracketSegments(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.app.aws_instance.web[0]", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("web[0]")
	if got := selectFirstResource(r); got == nil {
		t.Fatalf("expected bracketed query to match")
	}
	r.SetSearchQuery("module.app")
	if got := selectFirstResource(r); got == nil {
		t.Fatalf("expected dot query to match")
	}
}

func TestSearchMatchesResourceTypeAndName(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{
			Address:      "aws_instance.web",
			ResourceType: "aws_instance",
			ResourceName: "web",
			Action:       terraform.ActionCreate,
		},
	})
	r.SetSearchQuery("aws_instance")
	if got := selectFirstResource(r); got == nil {
		t.Fatalf("expected resource type to match")
	}
}

func TestFilterAndSearchCombined(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionUpdate},
	})
	r.SetFilter(terraform.ActionUpdate, false)
	r.SetSearchQuery("web")
	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected filtered resource to be hidden, got %#v", got)
	}
}

func TestRenderDiffQuotesPathSegments(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	item := diff.MinimalDiff{
		Path:     []string{"data", "app.yaml"},
		OldValue: "a",
		NewValue: "b",
		Action:   diff.DiffChange,
	}

	out := stripANSI(r.renderDiff(item))
	if !strings.Contains(out, `data."app.yaml"`) {
		t.Fatalf("expected quoted path, got %q", out)
	}
}

func TestRenderLongResourceAddressTruncates(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(20, 5)
	resource := terraform.ResourceChange{
		Address: strings.Repeat("a", 80),
		Action:  terraform.ActionCreate,
		Change:  &terraform.Change{Actions: []string{"create"}, Before: nil, After: map[string]any{"x": "y"}},
	}
	out := stripANSI(r.renderResource(&resource, false, 0, 18))
	lines := strings.Split(out, "\n")
	if len(lines) == 0 || len(lines[0]) > 20 {
		t.Fatalf("expected truncated header line, got %q", out)
	}
}

func TestExpandedResourceZeroDiffsNoExtraLine(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resource := terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionUpdate,
		Change:  &terraform.Change{Actions: []string{"update"}, Before: map[string]any{"a": 1}, After: map[string]any{"a": 1}},
	}
	out := r.renderResource(&resource, false, 0, 80)
	if strings.Count(out, "\n") != 0 {
		t.Fatalf("expected no extra diff lines, got %q", out)
	}
}

func TestFilterPreservesSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.a", Action: terraform.ActionCreate},
		{Address: "aws_instance.b", Action: terraform.ActionDelete},
	})
	r.selectedIndex = 1
	r.SetFilter(terraform.ActionDelete, false)
	if got := r.GetSelectedResource(); got == nil || got.Address != "aws_instance.a" {
		t.Fatalf("expected selection preserved, got %#v", got)
	}
}

func TestToggleAllGroups(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	if len(r.visibleItems) == 0 {
		t.Fatalf("expected visible items")
	}

	r.ToggleAllGroups()
	if r.allExpanded {
		t.Fatalf("expected all groups collapsed")
	}

	r.ToggleAllGroups()
	if !r.allExpanded {
		t.Fatalf("expected all groups expanded")
	}
}

func TestModulePathParsing(t *testing.T) {
	if got := modulePath("aws_instance.web"); len(got) != 0 {
		t.Fatalf("expected empty module path, got %#v", got)
	}
	got := modulePath("module.alpha.module.net.aws_instance.web")
	if len(got) != 2 || got[0] != "alpha" || got[1] != "net" {
		t.Fatalf("unexpected module path: %#v", got)
	}
}

func TestDeepGroupingCreatesNestedItems(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.aws_instance.db", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.security.aws_instance.bastion", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	foundGroup := false
	foundSubGroup := false
	for _, item := range r.visibleItems {
		if item.kind == itemGroup && item.label == consts.ModuleAlpha {
			foundGroup = true
		}
		if item.kind == itemGroup && item.label == "module.net" {
			foundSubGroup = true
		}
	}
	if !foundGroup || !foundSubGroup {
		t.Fatalf("expected nested groups, got %#v", r.visibleItems)
	}
}

func TestGroupLabelLocalNamePerDepth(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	labels := []string{}
	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			labels = append(labels, item.label)
		}
	}
	if !containsString(labels, consts.ModuleAlpha) || !containsString(labels, "module.net") || !containsString(labels, "module.edge") {
		t.Fatalf("unexpected group labels: %#v", labels)
	}
}

func TestGroupIndentIncrementsPerDepth(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	indents := []int{}
	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			indents = append(indents, item.indent)
		}
	}
	if len(indents) < 3 || indents[0] != 0 || indents[1] != 2 || indents[2] != 4 {
		t.Fatalf("unexpected indents: %#v", indents)
	}
}

func TestToggleGroupCollapseHidesDescendants(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	if groupIdx < 0 {
		t.Fatalf("expected module.alpha group")
	}
	r.selectedIndex = groupIdx
	r.ToggleGroup()
	for _, item := range r.visibleItems {
		if item.kind == itemGroup && item.label != consts.ModuleAlpha {
			t.Fatalf("expected descendants hidden, got %#v", r.visibleItems)
		}
		if item.kind == itemResource {
			t.Fatalf("expected no resources when collapsed, got %#v", r.visibleItems)
		}
	}
}

func TestToggleGroupExpandRestoresSortedChildren(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.zeta", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.alpha", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	r.selectedIndex = groupIdx
	r.ToggleGroup()
	r.ToggleGroup()

	resourceAddrs := []string{}
	for _, item := range r.visibleItems {
		if item.kind == itemResource {
			resourceAddrs = append(resourceAddrs, item.resource.Address)
		}
	}
	if len(resourceAddrs) < 2 || resourceAddrs[0] > resourceAddrs[1] {
		t.Fatalf("expected sorted children, got %#v", resourceAddrs)
	}
}

func TestGroupsSortedAlphabeticallyPerDepth(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.beta.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.gamma.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupLabels := []string{}
	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			groupLabels = append(groupLabels, item.label)
		}
	}
	if !isSorted(groupLabels) {
		t.Fatalf("expected groups sorted, got %#v", groupLabels)
	}
}

func TestResourcesSortedWithinGroup(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.zeta", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.alpha", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	resourceAddrs := []string{}
	for _, item := range r.visibleItems {
		if item.kind == itemResource {
			resourceAddrs = append(resourceAddrs, item.resource.Address)
		}
	}
	if len(resourceAddrs) < 2 || resourceAddrs[0] > resourceAddrs[1] {
		t.Fatalf("expected sorted resources, got %#v", resourceAddrs)
	}
}

func TestUngroupedResourcesAfterGroups(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.root", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	lastGroupIdx := -1
	rootIdx := -1
	for i, item := range r.visibleItems {
		if item.kind == itemGroup {
			lastGroupIdx = i
		}
		if item.kind == itemResource && item.resource.Address == "aws_instance.root" {
			rootIdx = i
		}
	}
	if rootIdx <= lastGroupIdx {
		t.Fatalf("expected ungrouped resource after groups: %#v", r.visibleItems)
	}
}

func TestSelectionOnGroupHeaderHasNoResource(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	r.selectedIndex = groupIdx
	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected nil resource for group selection, got %#v", got)
	}
}

func TestSelectionMovesOverCollapsedGroup(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	r.selectedIndex = groupIdx
	r.ToggleGroup()
	r.MoveDown()
	if r.selectedIndex == groupIdx {
		t.Fatalf("expected selection to move past collapsed group")
	}
}

func TestCollapseSelectedGroupClearsDiffSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	r.selectedIndex = groupIdx
	r.ToggleGroup()
	if r.GetSelectedResource() != nil {
		t.Fatalf("expected no selected resource after collapsing group")
	}
}

func TestSingleResourceLeafNoGroupHeader(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			t.Fatalf("expected no group header for single resource, got %#v", r.visibleItems)
		}
	}
}

func TestSingleResourceLeafShowsGroupHeaderDuringSearch(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("web")
	r.SetSize(80, 10)

	// During search, even single resources should show group headers for context
	foundGroup := false
	for _, item := range r.visibleItems {
		if item.kind == itemGroup && item.label == consts.ModuleAlpha {
			foundGroup = true
			break
		}
	}
	if !foundGroup {
		t.Fatalf("expected group header during search, got %#v", r.visibleItems)
	}
}

func TestNestedSingleResourceShowsAllGroupsDuringSearch(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.module.edge.aws_instance.node_4", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.other", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("node_4")
	r.SetSize(80, 10)

	// Should show all nested groups: alpha, net, and edge
	foundAlpha := false
	foundNet := false
	foundEdge := false
	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			switch item.label {
			case consts.ModuleAlpha:
				foundAlpha = true
			case "module.net":
				foundNet = true
			case "module.edge":
				foundEdge = true
			}
		}
	}
	if !foundAlpha || !foundNet || !foundEdge {
		t.Fatalf("expected all nested groups during search (alpha=%v, net=%v, edge=%v), got %#v",
			foundAlpha, foundNet, foundEdge, r.visibleItems)
	}
}

func TestMixedDepthModulesRenderNestedGroups(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	if indexOfGroup(r.visibleItems, "module.net") < 0 || indexOfGroup(r.visibleItems, "module.edge") < 0 {
		t.Fatalf("expected nested groups for mixed depth, got %#v", r.visibleItems)
	}
}

func TestNonModuleContextDoesNotGroup(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "example.module_value.resource", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			t.Fatalf("expected no grouping for non-module context, got %#v", r.visibleItems)
		}
	}
}

func TestToggleAllGroupsDeepTree(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.beta.module.db.module.read.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	r.ToggleAllGroups()
	if r.allExpanded {
		t.Fatalf("expected all groups collapsed")
	}
	r.ToggleAllGroups()
	if !r.allExpanded {
		t.Fatalf("expected all groups expanded")
	}
}

func TestSearchWithGroupingCountsAndVisibility(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.web2", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("web")
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	if groupIdx < 0 || r.visibleItems[groupIdx].count != 2 {
		t.Fatalf("expected filtered count for module.alpha, got %#v", r.visibleItems)
	}
}

func TestFilterWithGroupingCountsAndVisibility(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionDelete},
	})
	r.SetFilter(terraform.ActionDelete, false)
	r.SetSize(80, 10)

	if indexOfGroup(r.visibleItems, consts.ModuleAlpha) >= 0 {
		t.Fatalf("expected single resource without group header, got %#v", r.visibleItems)
	}
	if got := selectFirstResource(r); got == nil || got.Address != "module.alpha.aws_instance.web" {
		t.Fatalf("expected remaining resource, got %#v", got)
	}
}

func TestResourceListMoveUpAndUpdate(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(40, 5)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.db", Action: terraform.ActionCreate},
	})

	r.MoveDown()
	if r.selectedIndex != 1 {
		t.Fatalf("expected selection to move down, got %d", r.selectedIndex)
	}
	r.MoveUp()
	if r.selectedIndex != 0 {
		t.Fatalf("expected selection to move up, got %d", r.selectedIndex)
	}

	r.selectedIndex = 1
	r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if r.selectedIndex != 0 {
		t.Fatalf("expected key update to move selection up, got %d", r.selectedIndex)
	}
}

func TestResourceListInitReturnsNil(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	if cmd := r.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got %#v", cmd)
	}
}

func TestFirstResourceIndex(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.visibleItems = []listItem{
		{kind: itemGroup, label: consts.ModuleAlpha},
		{kind: itemResource, resource: &terraform.ResourceChange{Address: "aws_instance.web"}},
	}
	if idx := r.firstResourceIndex(); idx != 1 {
		t.Fatalf("expected first resource index 1, got %d", idx)
	}
}

func TestPadMultilinePadsEachLine(t *testing.T) {
	out := padMultiline("a\nbb", 4)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two lines, got %#v", lines)
	}
	for _, line := range lines {
		if len(line) != 4 {
			t.Fatalf("expected padded line width 4, got %q", line)
		}
	}
}

func TestResourceListViewportOffsetOnJump(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := make([]terraform.ResourceChange, 0, 30)
	for i := 0; i < 30; i++ {
		resources = append(resources, terraform.ResourceChange{
			Address: fmt.Sprintf("aws_instance.%d", i),
			Action:  terraform.ActionCreate,
		})
	}
	r.SetResources(resources)
	r.SetSize(40, 5)

	for i := 0; i < 15; i++ {
		r.MoveDown()
	}
	if r.viewport.YOffset == 0 {
		t.Fatalf("expected viewport to scroll, got offset %d", r.viewport.YOffset)
	}
}

func TestToggleGroupWhileSearchRetainsSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := []terraform.ResourceChange{
		{Address: "module.foo.aws_instance.a", Action: terraform.ActionCreate},
		{Address: "module.foo.aws_instance.b", Action: terraform.ActionCreate},
	}
	r.SetResources(resources)
	r.SetSize(60, 10)
	r.SetSearchQuery("a")

	r.MoveDown()
	selected := r.GetSelectedResource()
	if selected == nil || selected.Address != "module.foo.aws_instance.a" {
		t.Fatalf("unexpected selected resource: %#v", selected)
	}
	r.ToggleGroup()

	selectedAfter := r.GetSelectedResource()
	if selectedAfter == nil || selectedAfter.Address != "module.foo.aws_instance.a" {
		t.Fatalf("selection changed after toggle: %#v", selectedAfter)
	}
}

func TestSearchGroupingAndFilterCounts(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := []terraform.ResourceChange{
		{Address: "module.foo.aws_instance.a", Action: terraform.ActionCreate},
		{Address: "module.foo.aws_instance.b", Action: terraform.ActionUpdate},
		{Address: "module.bar.aws_instance.c", Action: terraform.ActionCreate},
	}
	r.SetResources(resources)
	r.SetSize(80, 10)
	r.SetFilter(terraform.ActionUpdate, false)
	r.SetSearchQuery("module.foo")

	if len(r.visibleItems) == 0 {
		t.Fatalf("expected visible items")
	}
	group := r.visibleItems[0]
	if group.kind != itemGroup || group.count != 1 {
		t.Fatalf("expected group count 1, got %#v", group)
	}
}
