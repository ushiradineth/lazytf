package components

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/utils"
)

const multilineContextLines = 6

// DiffViewer renders a side-by-side diff for the selected resource.
type DiffViewer struct {
	styles       *styles.Styles
	diffEngine   *diff.Engine
	width        int
	height       int
	scrollOffset int
	totalLines   int

	activeResource string
	hunks          []diffHunk
	activeHunk     int
	hunkLineStarts []int
	foldedHunks    map[string]bool
}

type diffHunk struct {
	key       string
	title     string
	parentKey string
	diffs     []diff.MinimalDiff
}

type diffTreeNode struct {
	key      string
	title    string
	children []*diffTreeNode
	childMap map[string]*diffTreeNode
	leafDiff []diff.MinimalDiff
}

// NewDiffViewer creates a diff viewer.
func NewDiffViewer(s *styles.Styles, engine *diff.Engine) *DiffViewer {
	return &DiffViewer{
		styles:      s,
		diffEngine:  engine,
		foldedHunks: make(map[string]bool),
	}
}

// SetSize updates the viewer dimensions.
func (d *DiffViewer) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetStyles updates the component styles.
func (d *DiffViewer) SetStyles(s *styles.Styles) {
	d.styles = s
}

// ScrollUp scrolls the view up by n lines.
func (d *DiffViewer) ScrollUp(n int) {
	d.scrollOffset -= n
	if d.scrollOffset < 0 {
		d.scrollOffset = 0
	}
}

// ScrollDown scrolls the view down by n lines.
func (d *DiffViewer) ScrollDown(n int) {
	d.scrollOffset += n
	maxOffset := d.totalLines - d.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if d.scrollOffset > maxOffset {
		d.scrollOffset = maxOffset
	}
}

// ScrollToTop scrolls to the top.
func (d *DiffViewer) ScrollToTop() {
	d.scrollOffset = 0
}

// ScrollToBottom scrolls to the bottom.
func (d *DiffViewer) ScrollToBottom() {
	maxOffset := d.totalLines - d.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	d.scrollOffset = maxOffset
}

// ResetScroll resets the scroll position.
func (d *DiffViewer) ResetScroll() {
	d.scrollOffset = 0
}

// GetScrollInfo returns scroll position info for scrollbar.
func (d *DiffViewer) GetScrollInfo() (offset, total, visible int) {
	return d.scrollOffset, d.totalLines, d.height
}

// NextHunk moves selection to the next hunk.
func (d *DiffViewer) NextHunk() bool {
	visible := d.visibleHunkIndices()
	if len(visible) == 0 {
		return false
	}
	pos := -1
	for i, idx := range visible {
		if idx == d.activeHunk {
			pos = i
			break
		}
	}
	if pos == -1 {
		d.activeHunk = visible[0]
		d.scrollToActiveHunk()
		return true
	}
	if pos >= len(visible)-1 {
		return false
	}
	d.activeHunk = visible[pos+1]
	d.scrollToActiveHunk()
	return true
}

// PrevHunk moves selection to the previous hunk.
func (d *DiffViewer) PrevHunk() bool {
	visible := d.visibleHunkIndices()
	if len(visible) == 0 {
		return false
	}
	pos := -1
	for i, idx := range visible {
		if idx == d.activeHunk {
			pos = i
			break
		}
	}
	if pos <= 0 {
		return false
	}
	d.activeHunk = visible[pos-1]
	d.scrollToActiveHunk()
	return true
}

// TreeParent navigates to parent section.
func (d *DiffViewer) TreeParent() bool {
	if len(d.hunks) == 0 || d.activeHunk < 0 || d.activeHunk >= len(d.hunks) {
		return false
	}
	current := d.hunks[d.activeHunk]
	if current.parentKey == "" {
		return false
	}
	if idx := d.indexByKey(current.parentKey); idx >= 0 {
		d.activeHunk = idx
		d.scrollToActiveHunk()
		return true
	}
	return false
}

// TreeChild expands current section or navigates to its first child section.
func (d *DiffViewer) TreeChild() bool {
	if len(d.hunks) == 0 || d.activeHunk < 0 || d.activeHunk >= len(d.hunks) {
		return false
	}
	current := d.hunks[d.activeHunk]
	if d.foldedHunks[current.key] {
		d.foldedHunks[current.key] = false
		return true
	}
	for i, h := range d.hunks {
		if h.parentKey == current.key {
			d.activeHunk = i
			d.scrollToActiveHunk()
			return true
		}
	}
	return false
}

func (d *DiffViewer) visibleHunkIndices() []int {
	if len(d.hunks) == 0 {
		return nil
	}
	visible := make([]int, 0, len(d.hunks))
	for i, h := range d.hunks {
		parentKey := h.parentKey
		hidden := false
		for parentKey != "" {
			if d.foldedHunks[parentKey] {
				hidden = true
				break
			}
			parentIdx := d.indexByKey(parentKey)
			if parentIdx < 0 {
				break
			}
			parentKey = d.hunks[parentIdx].parentKey
		}
		if !hidden {
			visible = append(visible, i)
		}
	}
	return visible
}

func (d *DiffViewer) indexByKey(key string) int {
	for i, h := range d.hunks {
		if h.key == key {
			return i
		}
	}
	return -1
}

// ToggleCurrentHunk collapses or expands the selected hunk.
func (d *DiffViewer) ToggleCurrentHunk() bool {
	if len(d.hunks) == 0 || d.activeHunk < 0 || d.activeHunk >= len(d.hunks) {
		return false
	}
	if d.foldedHunks == nil {
		d.foldedHunks = make(map[string]bool)
	}
	key := d.hunks[d.activeHunk].key
	d.foldedHunks[key] = !d.foldedHunks[key]
	return true
}

// GetHunkInfo returns 1-based selected hunk index and total hunk count.
func (d *DiffViewer) GetHunkInfo() (current, total int) {
	total = len(d.hunks)
	if total == 0 {
		return 0, 0
	}
	return d.activeHunk + 1, total
}

// View renders the diff viewer content.
func (d *DiffViewer) View(resource *terraform.ResourceChange) string {
	if resource == nil {
		d.activeResource = ""
		d.hunks = nil
		d.activeHunk = 0
		d.hunkLineStarts = nil
		d.totalLines = 0
		return d.pad("")
	}

	diffs := d.diffEngine.GetResourceDiffs(resource)

	if shouldRenderRawFallback(resource, len(diffs)) {
		d.syncHunks(resource.Address, nil)
		content := d.renderRawFallback(resource)
		return d.renderScrollableContent(content)
	}

	if len(diffs) == 0 {
		d.syncHunks(resource.Address, nil)
		var content string
		if resource.Action != terraform.ActionNoOp {
			action := actionLabel(resource.Action)
			if action == "" {
				action = string(resource.Action)
			}
			msg := fmt.Sprintf("Planned %s (details unavailable)", action)
			content = d.styles.Dimmed.Render(msg)
		} else {
			content = d.styles.Dimmed.Render("No changes for selected resource")
		}
		return d.renderScrollableContent(content)
	}

	header := d.renderHeader(resource, diffs)
	var body string
	switch {
	case resource.Action == terraform.ActionDelete:
		d.syncHunks(resource.Address, nil)
		body = d.renderDestroyReason(resource)
	default:
		displayDiffs := expandCompositeDiffsForDisplay(diffs)
		tree := buildDiffTree(displayDiffs)
		hunks := buildTreeHunks(tree)
		d.syncHunks(resource.Address, hunks)
		body = d.renderTreeBody(tree, resource.Change)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return d.renderScrollableContent(content)
}

func (d *DiffViewer) renderDestroyReason(resource *terraform.ResourceChange) string {
	lines := extractDestroyReasonLines(resource)
	if len(lines) == 0 {
		return d.styles.Dimmed.Render("No destroy details available")
	}

	rows := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if d.width > 0 {
			trimmed = PadLine(trimmed, d.width)
		}
		rows = append(rows, d.styles.Comment.Render(trimmed))
	}
	if len(rows) == 0 {
		return d.styles.Dimmed.Render("No destroy details available")
	}
	return strings.Join(rows, "\n")
}

func extractDestroyReasonLines(resource *terraform.ResourceChange) []string {
	if resource == nil {
		return nil
	}

	raw := strings.TrimSpace(resource.RawBlock)
	if raw != "" {
		lines := strings.Split(raw, "\n")
		collected := make([]string, 0, 4)
		started := false
		for _, line := range lines {
			trimmedLeft := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(trimmedLeft, "#") {
				collected = append(collected, strings.TrimSpace(trimmedLeft))
				started = true
				continue
			}
			if started && strings.TrimSpace(trimmedLeft) != "" {
				break
			}
		}
		if len(collected) > 0 {
			return collected
		}
	}

	address := strings.TrimSpace(resource.Address)
	if address == "" {
		address = "resource"
	}
	target := strings.TrimSpace(resource.ResourceType)
	if target == "" {
		target = strings.TrimSpace(resource.ResourceName)
	}
	if strings.TrimSpace(resource.ResourceType) != "" && strings.TrimSpace(resource.ResourceName) != "" {
		target = resource.ResourceType + "." + resource.ResourceName
	}
	if target == "" {
		target = address
	}

	return []string{
		"# " + address + " will be destroyed",
		"# (because " + target + " is not in configuration)",
	}
}

func (d *DiffViewer) syncHunks(resourceAddress string, hunks []diffHunk) {
	if d.foldedHunks == nil {
		d.foldedHunks = make(map[string]bool)
	}

	resourceChanged := d.activeResource != resourceAddress
	d.activeResource = resourceAddress

	if len(hunks) == 0 {
		d.hunks = nil
		d.activeHunk = 0
		d.hunkLineStarts = nil
		d.foldedHunks = make(map[string]bool)
		return
	}

	nextFolded := make(map[string]bool, len(hunks))
	if !resourceChanged {
		for _, h := range hunks {
			if d.foldedHunks[h.key] {
				nextFolded[h.key] = true
			}
		}
	}
	d.foldedHunks = nextFolded
	d.hunks = hunks

	if resourceChanged {
		d.activeHunk = 0
		d.scrollOffset = 0
	}

	if d.activeHunk < 0 {
		d.activeHunk = 0
	}
	if d.activeHunk >= len(d.hunks) {
		d.activeHunk = len(d.hunks) - 1
	}
}

func buildDiffHunks(diffs []diff.MinimalDiff) []diffHunk {
	return buildTreeHunks(buildDiffTree(expandCompositeDiffsForDisplay(diffs)))
}

func expandCompositeDiffsForDisplay(diffs []diff.MinimalDiff) []diff.MinimalDiff {
	if len(diffs) == 0 {
		return nil
	}

	out := make([]diff.MinimalDiff, 0, len(diffs))
	for _, item := range diffs {
		expanded := expandCompositeDiffItem(item)
		if len(expanded) == 0 {
			out = append(out, item)
			continue
		}
		out = append(out, expanded...)
	}
	return out
}

func expandCompositeDiffItem(item diff.MinimalDiff) []diff.MinimalDiff {
	switch item.Action {
	case diff.DiffAdd:
		return flattenCompositeValue(item.Path, nil, item.NewValue, diff.DiffAdd)
	case diff.DiffRemove:
		return flattenCompositeValue(item.Path, item.OldValue, nil, diff.DiffRemove)
	default:
		return []diff.MinimalDiff{item}
	}
}

func flattenCompositeValue(path []string, oldValue, newValue any, action diff.DiffAction) []diff.MinimalDiff {
	value := valueForAction(oldValue, newValue, action)

	if utils.IsMap(value) {
		entries := sortedMapEntries(value)
		if len(entries) == 0 {
			return []diff.MinimalDiff{leafCompositeDiff(path, oldValue, newValue, action)}
		}

		out := make([]diff.MinimalDiff, 0, len(entries))
		for _, entry := range entries {
			childPath := append(append([]string{}, path...), entry.key)
			childOld, childNew := compositeChildValues(action, entry.value)
			out = append(out, flattenCompositeValue(childPath, childOld, childNew, action)...)
		}
		return out
	}

	if utils.IsList(value) {
		list := utils.InterfaceToList(value)
		if len(list) == 0 {
			return []diff.MinimalDiff{leafCompositeDiff(path, oldValue, newValue, action)}
		}

		out := make([]diff.MinimalDiff, 0, len(list))
		for i, elem := range list {
			childPath := append(append([]string{}, path...), fmt.Sprintf("__item_%d", i))
			childOld, childNew := compositeChildValues(action, elem)
			out = append(out, flattenCompositeValue(childPath, childOld, childNew, action)...)
		}
		return out
	}

	return []diff.MinimalDiff{leafCompositeDiff(path, oldValue, newValue, action)}
}

func valueForAction(oldValue, newValue any, action diff.DiffAction) any {
	if action == diff.DiffRemove {
		return oldValue
	}
	return newValue
}

func compositeChildValues(action diff.DiffAction, value any) (oldValue, newValue any) {
	if action == diff.DiffAdd {
		return nil, value
	}
	return value, nil
}

func leafCompositeDiff(path []string, oldValue, newValue any, action diff.DiffAction) diff.MinimalDiff {
	return diff.MinimalDiff{
		Path:     append([]string{}, path...),
		OldValue: oldValue,
		NewValue: newValue,
		Action:   action,
	}
}

type mapEntry struct {
	key   string
	value any
}

func sortedMapEntries(value any) []mapEntry {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return nil
	}

	entries := make([]mapEntry, 0, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		entries = append(entries, mapEntry{
			key:   fmt.Sprintf("%v", iter.Key().Interface()),
			value: iter.Value().Interface(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	return entries
}

func buildDiffTree(diffs []diff.MinimalDiff) *diffTreeNode {
	root := &diffTreeNode{
		key:      "(root)",
		title:    "root",
		childMap: make(map[string]*diffTreeNode),
	}
	if len(diffs) == 0 {
		return root
	}

	for _, item := range diffs {
		segments := normalizeDiffPath(item.Path)
		if len(segments) == 0 {
			root.leafDiff = append(root.leafDiff, item)
			continue
		}

		node := root
		prefix := ""
		for _, segment := range segments {
			if prefix == "" {
				prefix = segment
			} else {
				prefix += "/" + segment
			}
			node = node.ensureChild(prefix, treeSegmentTitle(segment))
		}
		node.leafDiff = append(node.leafDiff, item)
	}

	return root
}

func normalizeDiffPath(path []string) []string {
	segments := make([]string, 0, len(path))
	for _, segment := range path {
		if strings.TrimSpace(segment) == "" {
			continue
		}
		segments = append(segments, segment)
	}
	return segments
}

func treeSegmentTitle(segment string) string {
	if strings.HasPrefix(segment, "__item_") {
		idx := strings.TrimPrefix(segment, "__item_")
		if idx == "" {
			idx = "?"
		}
		return fmt.Sprintf("[%s]", idx)
	}
	return segment
}

func (n *diffTreeNode) ensureChild(key, title string) *diffTreeNode {
	if n.childMap == nil {
		n.childMap = make(map[string]*diffTreeNode)
	}
	if existing, ok := n.childMap[key]; ok {
		return existing
	}
	child := &diffTreeNode{
		key:      key,
		title:    title,
		childMap: make(map[string]*diffTreeNode),
	}
	n.childMap[key] = child
	n.children = append(n.children, child)
	return child
}

func buildTreeHunks(root *diffTreeNode) []diffHunk {
	if root == nil {
		return nil
	}

	hunks := make([]diffHunk, 0, 8)
	var walk func(*diffTreeNode, string)
	walk = func(node *diffTreeNode, parentKey string) {
		if shouldRenderTreeSection(node, root) {
			hunks = append(hunks, diffHunk{
				key:       node.key,
				title:     node.title,
				parentKey: parentKey,
				diffs:     collectNodeDiffs(node),
			})
		}
		nextParent := parentKey
		if shouldRenderTreeSection(node, root) {
			nextParent = node.key
		}
		for _, child := range node.children {
			walk(child, nextParent)
		}
	}
	walk(root, "")

	return hunks
}

func shouldRenderTreeSection(node, root *diffTreeNode) bool {
	if node == nil {
		return false
	}
	if node == root || len(node.children) > 0 {
		return true
	}
	if !strings.HasPrefix(lastKeySegment(node.key), "__item_") {
		return false
	}
	for _, item := range node.leafDiff {
		if isMultilineChange(item) {
			return true
		}
	}
	return false
}

func lastKeySegment(key string) string {
	if key == "" {
		return ""
	}
	parts := strings.Split(key, "/")
	return parts[len(parts)-1]
}

func collectNodeDiffs(node *diffTreeNode) []diff.MinimalDiff {
	if node == nil {
		return nil
	}
	out := make([]diff.MinimalDiff, 0, len(node.leafDiff))
	out = append(out, node.leafDiff...)
	for _, child := range node.children {
		out = append(out, collectNodeDiffs(child)...)
	}
	return out
}

func (d *DiffViewer) renderTreeBody(root *diffTreeNode, change *terraform.Change) string {
	if root == nil {
		d.hunkLineStarts = nil
		return d.styles.Dimmed.Render("No change details")
	}

	rows := make([]string, 0, len(d.hunks)*2)
	d.hunkLineStarts = make([]int, len(d.hunks))
	indexByKey := make(map[string]int, len(d.hunks))
	for i, h := range d.hunks {
		indexByKey[h.key] = i
	}

	d.renderTreeNode(root, change, 0, indexByKey, &rows)
	if len(rows) == 0 {
		return d.styles.Dimmed.Render("No change details")
	}
	return strings.Join(rows, "\n")
}

func (d *DiffViewer) renderTreeNode(
	node *diffTreeNode,
	change *terraform.Change,
	depth int,
	indexByKey map[string]int,
	rows *[]string,
) {
	if node == nil {
		return
	}

	contentDepth := depth
	if idx, ok := indexByKey[node.key]; ok {
		d.hunkLineStarts[idx] = len(*rows)
		*rows = append(*rows, d.renderHunkHeader(idx, d.hunks[idx], depth))
		if d.foldedHunks[node.key] {
			return
		}
		contentDepth = depth + 1
	}

	trimSegments := segmentDepthForKey(node.key)
	d.appendIndentedDiffRows(rows, node.leafDiff, change, contentDepth, trimSegments)

	for _, child := range node.children {
		if _, isSection := indexByKey[child.key]; isSection {
			d.renderTreeNode(child, change, contentDepth, indexByKey, rows)
			continue
		}
		d.appendIndentedDiffRows(rows, child.leafDiff, change, contentDepth, trimSegments)
		for _, grandChild := range child.children {
			d.renderTreeNode(grandChild, change, contentDepth, indexByKey, rows)
		}
	}
}

func segmentDepthForKey(key string) int {
	if key == "" || key == "(root)" {
		return 0
	}
	return len(strings.Split(key, "/"))
}

func displayPathForDepth(path []string, trimSegments int) []string {
	if trimSegments <= 0 {
		return path
	}

	trimmed := make([]string, 0, len(path))
	removed := 0
	for _, segment := range path {
		if removed < trimSegments && !strings.HasPrefix(segment, "__item_") && strings.TrimSpace(segment) != "" {
			removed++
			continue
		}
		trimmed = append(trimmed, segment)
	}
	if len(trimmed) == 0 {
		return path
	}
	return trimmed
}

func (d *DiffViewer) appendIndentedDiffRows(rows *[]string, diffs []diff.MinimalDiff, change *terraform.Change, depth, trimSegments int) {
	if len(diffs) == 0 {
		return
	}

	var block string
	if hasMultilineDiff(diffs) {
		block = d.renderBlocksWithTrim(diffs, change, trimSegments)
	} else {
		block = d.renderCompactListWithTrim(diffs, change, trimSegments)
	}
	if strings.TrimSpace(block) == "" {
		return
	}

	indent := strings.Repeat("  ", max(depth, 0))
	for _, line := range strings.Split(block, "\n") {
		*rows = append(*rows, indent+line)
	}
}

func (d *DiffViewer) renderHunkHeader(index int, h diffHunk, depth int) string {
	marker := "▼"
	if d.foldedHunks[h.key] {
		marker = "▶"
	}

	prefix := strings.Repeat("  ", max(depth, 0))
	line := fmt.Sprintf("%s%s %s (%d)", prefix, marker, h.title, len(h.diffs))
	if d.width > 0 {
		line = PadLine(line, d.width)
	}

	if index == d.activeHunk {
		return d.styles.FocusedPanelTitle.Render(line)
	}
	return d.styles.PanelTitle.Render(line)
}

func (d *DiffViewer) scrollToActiveHunk() {
	if d.activeHunk < 0 || d.activeHunk >= len(d.hunkLineStarts) {
		return
	}
	d.scrollOffset = d.hunkLineStarts[d.activeHunk]
	if d.scrollOffset < 0 {
		d.scrollOffset = 0
	}
}

func (d *DiffViewer) renderScrollableContent(content string) string {
	lines := strings.Split(content, "\n")
	d.totalLines = len(lines)

	maxOffset := d.totalLines - d.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if d.scrollOffset > maxOffset {
		d.scrollOffset = maxOffset
	}

	start := d.scrollOffset
	end := start + d.height
	if end > len(lines) {
		end = len(lines)
	}
	if start > len(lines) {
		start = len(lines)
	}

	visibleContent := strings.Join(lines[start:end], "\n")
	return d.pad(visibleContent)
}

func shouldRenderRawFallback(resource *terraform.ResourceChange, diffCount int) bool {
	if resource == nil {
		return false
	}
	if strings.TrimSpace(resource.RawBlock) == "" {
		return false
	}
	if resource.ParseStatus == terraform.ParseStatusPartial {
		return true
	}
	return resource.Action != terraform.ActionNoOp && diffCount == 0
}

func (d *DiffViewer) renderRawFallback(resource *terraform.ResourceChange) string {
	label := actionLabel(resource.Action)
	headerText := resource.Address + "  (raw terraform block)"
	if label != "" {
		headerText = label + ": " + headerText
	}
	if d.width > 0 && lipgloss.Width(headerText) > d.width {
		headerText = utils.TruncateEnd(headerText, d.width)
	}

	header := RenderSectionHeader(headerText, d.width, d.styles.DiffChange, d.styles.Theme.BorderColor)
	noticeText := "Parser could not fully parse this resource. Showing raw Terraform output to avoid hidden changes."
	if resource.ParseStatus != terraform.ParseStatusPartial {
		noticeText = "Structured diff details were unavailable for this resource. Showing raw Terraform output instead."
	}
	notice := d.styles.Dimmed.Render(noticeText)
	body := resource.RawBlock
	if strings.TrimSpace(body) == "" {
		body = "(raw block unavailable)"
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, "", notice, "", d.renderRawBlock(body))
}

func (d *DiffViewer) renderRawBlock(block string) string {
	lines := strings.Split(block, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		style := d.styles.LineItemText
		switch {
		case strings.HasPrefix(trimmed, "+"):
			style = d.styles.DiffAdd
		case strings.HasPrefix(trimmed, "-"):
			style = d.styles.DiffRemove
		case strings.HasPrefix(trimmed, "~"):
			style = d.styles.DiffChange
		case strings.HasPrefix(trimmed, "#"):
			style = d.styles.Comment
		}
		if d.width > 0 {
			line = PadLine(line, d.width)
		}
		out = append(out, style.Render(line))
	}
	return strings.Join(out, "\n")
}

func (d *DiffViewer) renderHeader(resource *terraform.ResourceChange, diffs []diff.MinimalDiff) string {
	changeCount := len(diffs)
	label := actionLabel(resource.Action)

	// Build header text: "update: module.alpha.aws_instance.node_0  (2 changes)"
	var headerText string
	if label != "" {
		headerText = label + ": " + resource.Address
	} else {
		headerText = resource.Address
	}

	// Add change count
	if changeCount == 1 {
		headerText += "  (1 change)"
	} else {
		headerText += fmt.Sprintf("  (%d changes)", changeCount)
	}

	if len(d.hunks) > 0 {
		headerText += fmt.Sprintf("  [tree %d/%d]", d.activeHunk+1, len(d.hunks))
	}

	// Truncate if needed
	if d.width > 0 && lipgloss.Width(headerText) > d.width {
		headerText = utils.TruncateEnd(headerText, d.width)
	}

	// Use shared header rendering function
	actionStyle := d.actionStyle(resource.Action)
	return RenderSectionHeader(headerText, d.width, actionStyle, d.styles.Theme.BorderColor) + "\n"
}

func (d *DiffViewer) actionStyle(action terraform.ActionType) lipgloss.Style {
	switch action {
	case terraform.ActionCreate:
		return d.styles.DiffAdd
	case terraform.ActionUpdate:
		return d.styles.DiffChange
	case terraform.ActionDelete:
		return d.styles.DiffRemove
	case terraform.ActionReplace:
		return d.styles.DiffChange
	default:
		return d.styles.Dimmed
	}
}

func (d *DiffViewer) renderTable(diffs []diff.MinimalDiff, change *terraform.Change) string {
	rows := make([]string, 0, len(diffs))
	for _, item := range diffs {
		rows = append(rows, d.renderInlineChange(item, change))
	}
	return strings.Join(rows, "\n")
}

func (d *DiffViewer) renderBlocks(diffs []diff.MinimalDiff, change *terraform.Change) string {
	return d.renderBlocksWithTrim(diffs, change, 0)
}

func (d *DiffViewer) renderBlocksWithTrim(diffs []diff.MinimalDiff, change *terraform.Change, trimSegments int) string {
	var rows []string
	prevRoot := ""
	lastSpacer := false
	for i, item := range diffs {
		root := ""
		if len(item.Path) > 0 {
			root = item.Path[0]
		}
		if prevRoot != "" && root != prevRoot && !lastSpacer {
			rows = append(rows, "")
			lastSpacer = true
		}
		switch {
		case isMultilineChange(item):
			rows = append(rows, d.renderMultilineBlockWithTrim(item, change, trimSegments)...)
			if i < len(diffs)-1 {
				rows = append(rows, "")
				lastSpacer = true
			}
		default:
			rows = append(rows, d.renderInlineChangeWithTrim(item, change, trimSegments))
			lastSpacer = false
		}
		prevRoot = root
	}
	return strings.Join(rows, "\n")
}

func (d *DiffViewer) renderCompactList(diffs []diff.MinimalDiff, change *terraform.Change) string {
	return d.renderCompactListWithTrim(diffs, change, 0)
}

func (d *DiffViewer) renderCompactListWithTrim(diffs []diff.MinimalDiff, change *terraform.Change, trimSegments int) string {
	rows := make([]string, 0, len(diffs))
	for _, item := range diffs {
		rows = append(rows, d.renderInlineChangeWithTrim(item, change, trimSegments))
	}
	return strings.Join(rows, "\n")
}

func (d *DiffViewer) renderInlineChange(item diff.MinimalDiff, change *terraform.Change) string {
	return d.renderInlineChangeWithTrim(item, change, 0)
}

func (d *DiffViewer) renderInlineChangeWithTrim(item diff.MinimalDiff, change *terraform.Change, trimSegments int) string {
	path := formatPathForDisplay(displayPathForDepth(item.Path, trimSegments))
	if strings.TrimSpace(path) == "" {
		path = formatPathForDisplay(item.Path)
	}
	symbol := item.Action.GetActionSymbol()
	var style lipgloss.Style
	var line string
	marker := replaceMarker(item.Path, change)
	var markerPlain, markerStyled string
	if marker != "" {
		markerPlain = "  " + marker
		markerStyled = d.styles.Comment.Render(markerPlain)
	}

	switch item.Action {
	case diff.DiffAdd:
		style = d.styles.DiffAdd
		line = fmt.Sprintf("%s %s: %s", symbol, path, formatSingleLineValue(item.NewValue))
	case diff.DiffRemove:
		style = d.styles.DiffRemove
		line = fmt.Sprintf("%s %s: %s", symbol, path, formatSingleLineValue(item.OldValue))
	case diff.DiffChange:
		style = d.styles.DiffChange
		line = fmt.Sprintf("%s %s: %s → %s", symbol, path, formatSingleLineValue(item.OldValue), formatSingleLineValue(item.NewValue))
	default:
		style = d.styles.Dimmed
		line = "? " + path
	}

	if d.width > 0 {
		line = PadLine(line, d.width)
	}
	if markerStyled != "" {
		return style.Render(strings.TrimRight(line, " ")) + markerStyled
	}
	return style.Render(line)
}

func (d *DiffViewer) renderMultilineBlock(item diff.MinimalDiff, change *terraform.Change) []string {
	return d.renderMultilineBlockWithTrim(item, change, 0)
}

func (d *DiffViewer) renderMultilineBlockWithTrim(item diff.MinimalDiff, change *terraform.Change, trimSegments int) []string {
	path := formatPathForDisplay(displayPathForDepth(item.Path, trimSegments))
	if strings.TrimSpace(path) == "" {
		path = formatPathForDisplay(item.Path)
	}
	symbol := item.Action.GetActionSymbol()
	marker := replaceMarker(item.Path, change)
	oldStr, _ := item.OldValue.(string)
	newStr, _ := item.NewValue.(string)
	oldStr = normalizeEscapedWhitespace(oldStr)
	newStr = normalizeEscapedWhitespace(newStr)
	oldStr, newStr = normalizeMultilineForDisplay(oldStr, newStr)
	lines := buildContextDiff(oldStr, newStr, multilineContextLines)

	var output []string
	if isListItemPath(path) {
		separator := "-----"
		if d.width > 0 {
			separator = PadLine(separator, d.width)
		}
		output = append(output, d.styles.Dimmed.Render(separator))
	} else {
		header := fmt.Sprintf("%s %s", symbol, path)
		if d.width > 0 {
			header = PadLine(header, d.width)
		}
		headerStyle := d.styles.DiffChange
		switch item.Action {
		case diff.DiffAdd:
			headerStyle = d.styles.DiffAdd
		case diff.DiffRemove:
			headerStyle = d.styles.DiffRemove
		case diff.DiffChange:
			headerStyle = d.styles.DiffChange
		}
		header = headerStyle.Render(header)
		if marker != "" {
			header = header + d.styles.Comment.Render("  "+marker)
		}
		output = append(output, header)
	}
	for _, line := range lines {
		prefix := linePrefix(line)
		line = "  " + line
		if d.width > 0 {
			line = PadLine(line, d.width)
		}
		switch prefix {
		case "-":
			output = append(output, d.styles.DiffRemove.Render(line))
		case "+":
			output = append(output, d.styles.DiffAdd.Render(line))
		case ".":
			output = append(output, d.styles.Dimmed.Render(line))
		default:
			output = append(output, d.styles.Dimmed.Render(line))
		}
	}
	return output
}

func isListItemPath(path string) bool {
	if len(path) < 3 {
		return false
	}
	if path[0] != '[' || path[len(path)-1] != ']' {
		return false
	}
	return true
}

func (d *DiffViewer) renderDiffRow(columns []int, item diff.MinimalDiff) string {
	path := formatPathForDisplay(item.Path)
	symbol := item.Action.GetActionSymbol()

	before := ""
	after := ""
	switch item.Action {
	case diff.DiffAdd:
		after = formatSingleLineValue(item.NewValue)
		return d.renderRow(columns, d.styles.DiffAdd, symbol, path, before, after)
	case diff.DiffRemove:
		before = formatSingleLineValue(item.OldValue)
		return d.renderRow(columns, d.styles.DiffRemove, symbol, path, before, after)
	case diff.DiffChange:
		before = formatSingleLineValue(item.OldValue)
		after = formatSingleLineValue(item.NewValue)
		return d.renderRow(columns, d.styles.DiffChange, symbol, path, before, after)
	default:
		return d.renderRow(columns, d.styles.Dimmed, symbol, path, before, after)
	}
}

func (d *DiffViewer) renderRow(columns []int, style lipgloss.Style, symbol, path, before, after string) string {
	symbol = utils.TruncateEnd(symbol, columns[0]-1)
	path = utils.TruncateEnd(path, columns[1]-1)
	before = utils.TruncateEnd(before, columns[2]-1)
	after = utils.TruncateEnd(after, columns[3]-1)

	symbolCell := style.Width(columns[0]).MaxWidth(columns[0]).Render(symbol)
	pathCell := style.Width(columns[1]).MaxWidth(columns[1]).Render(path)
	beforeCell := style.Width(columns[2]).MaxWidth(columns[2]).Render(before)
	afterCell := style.Width(columns[3]).MaxWidth(columns[3]).Render(after)

	return lipgloss.JoinHorizontal(lipgloss.Left, symbolCell, pathCell, beforeCell, afterCell)
}

func (d *DiffViewer) columnWidths() []int {
	if d.width <= 0 {
		return []int{2, 18, 18, 18}
	}

	symbolWidth := 2
	pathWidth := int(float64(d.width) * 0.32)
	beforeWidth := int(float64(d.width) * 0.31)
	afterWidth := d.width - symbolWidth - pathWidth - beforeWidth

	if pathWidth < 16 {
		pathWidth = 16
	}
	if beforeWidth < 14 {
		beforeWidth = 14
	}
	if afterWidth < 14 {
		afterWidth = 14
	}

	remaining := d.width - symbolWidth - pathWidth - beforeWidth - afterWidth
	if remaining != 0 {
		afterWidth += remaining
	}

	return []int{symbolWidth, pathWidth, beforeWidth, afterWidth}
}

func (d *DiffViewer) pad(content string) string {
	if d.width <= 0 || d.height <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(d.width).Height(d.height).Render(content)
}

func formatSingleLineValue(val any) string {
	if s, ok := val.(string); ok {
		if hasEscapedOrActualWhitespace(s) {
			s = normalizeEscapedWhitespace(s)
			s = strings.ReplaceAll(s, "\n", " ⏎ ")
			s = strings.ReplaceAll(s, "\t", " ⇥ ")
			s = strings.Join(strings.Fields(s), " ")
			s = truncateMiddle(s, 180)
			return s
		}
	}
	return formatValue(val)
}

func hasEscapedOrActualWhitespace(s string) bool {
	return strings.Contains(s, "\n") || strings.Contains(s, "\t") || strings.Contains(s, `\n`) || strings.Contains(s, `\t`)
}

func normalizeEscapedWhitespace(s string) string {
	// Normalize escaped JSON/YAML blobs for compact single-line display.
	r := strings.NewReplacer(
		`\\n`, "\n",
		`\\t`, "\t",
		`\\r`, "\r",
		`\n`, "\n",
		`\t`, "\t",
		`\r`, "\r",
		`\\\"`, `"`,
		`\\"`, `"`,
		`\"`, `"`,
	)
	return r.Replace(s)
}

func truncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen || maxLen <= 3 {
		return s
	}
	head := maxLen / 2
	tail := maxLen - head - 3
	if tail < 1 {
		tail = 1
	}
	return s[:head] + "..." + s[len(s)-tail:]
}

func hasMultilineDiff(diffs []diff.MinimalDiff) bool {
	for _, item := range diffs {
		if isMultilineChange(item) {
			return true
		}
	}
	return false
}

func isMultilineChange(item diff.MinimalDiff) bool {
	switch item.Action {
	case diff.DiffAdd:
		newStr, ok := item.NewValue.(string)
		return ok && hasLineBreakHint(newStr)
	case diff.DiffRemove:
		oldStr, ok := item.OldValue.(string)
		return ok && hasLineBreakHint(oldStr)
	case diff.DiffChange:
		oldStr, okOld := item.OldValue.(string)
		newStr, okNew := item.NewValue.(string)
		return okOld && okNew && (hasLineBreakHint(oldStr) || hasLineBreakHint(newStr))
	default:
		return false
	}
}

func hasLineBreakHint(s string) bool {
	if s == "" {
		return false
	}
	return strings.Contains(s, "\n") || strings.Contains(s, `\n`) || strings.Contains(s, `\\n`)
}

func buildContextDiff(before, after string, context int) []string {
	beforeLines := splitLines(before)
	afterLines := splitLines(after)
	maxLen := maxLineCount(beforeLines, afterLines)

	diffIdx := diffLineIndexes(beforeLines, afterLines, maxLen)
	if len(diffIdx) == 0 {
		return []string{"  (no diff)"}
	}

	spans := diffSpans(diffIdx, context, maxLen)
	return renderDiffSpans(beforeLines, afterLines, spans)
}

type diffSpan struct {
	start int
	end   int
}

func splitLines(value string) []string {
	return strings.Split(strings.TrimSuffix(value, "\n"), "\n")
}

func maxLineCount(beforeLines, afterLines []string) int {
	if len(afterLines) > len(beforeLines) {
		return len(afterLines)
	}
	return len(beforeLines)
}

func diffLineIndexes(beforeLines, afterLines []string, maxLen int) []int {
	diffIdx := make([]int, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		oldLine := lineAt(beforeLines, i)
		newLine := lineAt(afterLines, i)
		if oldLine != newLine {
			diffIdx = append(diffIdx, i)
		}
	}
	return diffIdx
}

func diffSpans(indexes []int, context, maxLen int) []diffSpan {
	spans := []diffSpan{}
	for _, idx := range indexes {
		start := max(0, idx-context)
		end := min(maxLen-1, idx+context)
		if len(spans) == 0 || start > spans[len(spans)-1].end+1 {
			spans = append(spans, diffSpan{start: start, end: end})
			continue
		}
		if end > spans[len(spans)-1].end {
			spans[len(spans)-1].end = end
		}
	}
	return spans
}

func renderDiffSpans(beforeLines, afterLines []string, spans []diffSpan) []string {
	var out []string
	lastEnd := -1
	for _, sp := range spans {
		if lastEnd >= 0 && sp.start > lastEnd+1 {
			out = append(out, "  ...")
		}
		out = append(out, renderSpanLines(beforeLines, afterLines, sp)...)
		lastEnd = sp.end
	}
	return out
}

func renderSpanLines(beforeLines, afterLines []string, sp diffSpan) []string {
	var out []string
	for i := sp.start; i <= sp.end; i++ {
		oldLine := lineAt(beforeLines, i)
		newLine := lineAt(afterLines, i)
		if oldLine == newLine {
			out = append(out, "  "+oldLine)
			continue
		}
		if oldLine != "" {
			out = append(out, "- "+oldLine)
		}
		if newLine != "" {
			out = append(out, "+ "+newLine)
		}
	}
	return out
}

func lineAt(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return ""
	}
	return lines[index]
}

func linePrefix(line string) string {
	if strings.HasPrefix(line, "- ") {
		return "-"
	}
	if strings.HasPrefix(line, "+ ") {
		return "+"
	}
	if strings.HasPrefix(line, "  ...") {
		return "."
	}
	return ""
}

func normalizeMultilineForDisplay(before, after string) (string, string) {
	beforeLines := splitLines(before)
	afterLines := splitLines(after)
	minIndent := minCommonIndent(beforeLines, afterLines)
	if minIndent <= 0 {
		return before, after
	}
	return strings.Join(trimCommonIndent(beforeLines, minIndent), "\n"), strings.Join(trimCommonIndent(afterLines, minIndent), "\n")
}

func minCommonIndent(beforeLines, afterLines []string) int {
	minIndent := -1
	apply := func(lines []string) {
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			indent := leadingIndentWidth(line)
			if minIndent == -1 || indent < minIndent {
				minIndent = indent
			}
		}
	}
	apply(beforeLines)
	apply(afterLines)
	if minIndent < 0 {
		return 0
	}
	return minIndent
}

func leadingIndentWidth(line string) int {
	count := 0
	for _, r := range line {
		if r == ' ' || r == '\t' {
			count++
			continue
		}
		break
	}
	return count
}

func trimCommonIndent(lines []string, n int) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		trim := n
		idx := 0
		for idx < len(line) && trim > 0 {
			if line[idx] != ' ' && line[idx] != '\t' {
				break
			}
			idx++
			trim--
		}
		out[i] = line[idx:]
	}
	return out
}

func actionLabel(action terraform.ActionType) string {
	switch action {
	case terraform.ActionCreate:
		return "create"
	case terraform.ActionUpdate:
		return "update"
	case terraform.ActionDelete:
		return "destroy"
	case terraform.ActionReplace:
		return "replace"
	default:
		return ""
	}
}

func replaceMarker(path []string, change *terraform.Change) string {
	if change == nil || len(change.ReplacePaths) == 0 {
		return ""
	}
	for _, rp := range change.ReplacePaths {
		if pathMatchesReplace(path, rp) {
			return " # forces replacement"
		}
	}
	return ""
}

func pathMatchesReplace(path, replace []string) bool {
	if len(replace) == 0 || len(path) < len(replace) {
		return false
	}
	for i := range replace {
		if path[i] != replace[i] {
			return false
		}
	}
	return true
}
