package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

// TextParser parses Terraform plan text output.
type TextParser struct {
	cleaner *Cleaner
}

// NewTextParser creates a new text plan parser.
func NewTextParser() *TextParser {
	return &TextParser{cleaner: NewCleaner()}
}

// Parse reads and parses text plan data from a reader.
func (p *TextParser) Parse(input io.Reader) (*terraform.Plan, error) {
	if input == nil {
		return nil, errors.New("no input provided")
	}

	scanner := bufio.NewScanner(input)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	builder := newPlanBuilder(p.cleaner)
	for scanner.Scan() {
		builder.consume(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read plan output: %w", err)
	}

	plan := builder.finish()
	if len(plan.Resources) == 0 && !builder.sawNoChanges {
		return nil, errors.New("no resource changes parsed from plan output")
	}
	return plan, nil
}

// ParseFile parses text plan output from a file path.
func (p *TextParser) ParseFile(filePath string) (*terraform.Plan, error) {
	file, err := openFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.Parse(file)
}

// ParseStream parses plan data from a line channel.
func (p *TextParser) ParseStream(lines <-chan string) (*terraform.Plan, error) {
	if lines == nil {
		return nil, errors.New("no input provided")
	}

	builder := newPlanBuilder(p.cleaner)
	for line := range lines {
		builder.consume(line)
	}

	plan := builder.finish()
	if len(plan.Resources) == 0 && !builder.sawNoChanges {
		return nil, errors.New("no resource changes parsed from plan output")
	}
	return plan, nil
}

type planBuilder struct {
	cleaner       *Cleaner
	plan          *terraform.Plan
	current       *terraform.ResourceChange
	before        map[string]any
	after         map[string]any
	afterUnknown  map[string]any
	pathStack     []string
	sawNoChanges  bool
	heredocActive bool
	heredocEnd    string
	heredocKey    string
	heredocPrefix string
	heredocBuffer []string
}

func newPlanBuilder(cleaner *Cleaner) *planBuilder {
	return &planBuilder{
		cleaner: cleaner,
		plan: &terraform.Plan{
			FormatVersion: "text",
			Metadata: terraform.PlanMetadata{
				Timestamp: time.Now(),
			},
		},
	}
}

func (b *planBuilder) consume(line string) {
	if b.cleaner != nil {
		line = b.cleaner.Normalize(line)
	}
	rawLine := line
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if strings.Contains(trimmed, "No changes.") {
		b.sawNoChanges = true
	}

	if b.heredocActive {
		if trimmed == b.heredocEnd {
			value := strings.Join(b.heredocBuffer, "\n")
			b.applyHeredocValue(b.heredocKey, b.heredocPrefix, value)
			b.heredocActive = false
			b.heredocEnd = ""
			b.heredocKey = ""
			b.heredocPrefix = ""
			b.heredocBuffer = nil
			return
		}
		b.heredocBuffer = append(b.heredocBuffer, rawLine)
		return
	}

	if address, action, ok := parseHeaderLine(trimmed); ok {
		b.startResource(address, action)
		return
	}

	if b.current == nil {
		return
	}

	if strings.Contains(trimmed, "resource \"") {
		b.parseResourceLine(trimmed)
	}

	prefix, rest := parseActionPrefix(trimmed)
	if prefix == "" {
		if isBlockClose(trimmed) {
			b.popPath()
		}
		return
	}

	if isBlockClose(rest) {
		b.popPath()
		return
	}

	key, valuePart, ok := parseKeyValue(rest)
	if !ok {
		return
	}

	if delimiter, ok := parseHeredocDelimiter(valuePart); ok {
		b.heredocActive = true
		b.heredocEnd = delimiter
		b.heredocKey = key
		b.heredocPrefix = prefix
		b.heredocBuffer = nil
		return
	}

	if isBlockOpen(valuePart) {
		b.pushPath(key)
		return
	}

	path := append([]string{}, b.pathStack...)
	path = append(path, key)

	if strings.Contains(valuePart, "->") {
		beforeStr, afterStr := splitArrow(valuePart)
		beforeVal, _ := parseTerraformValue(beforeStr)
		afterVal, unknown := parseTerraformValue(afterStr)
		b.setBefore(path, beforeVal)
		if unknown {
			b.setAfterUnknown(path)
		} else {
			b.setAfter(path, afterVal)
		}
		return
	}

	val, unknown := parseTerraformValue(valuePart)
	switch prefix {
	case "+":
		if unknown {
			b.setAfterUnknown(path)
		} else {
			b.setAfter(path, val)
		}
	case "-":
		b.setBefore(path, val)
	case "~":
		b.setBefore(path, val)
		if unknown {
			b.setAfterUnknown(path)
		} else {
			b.setAfter(path, val)
		}
	}
}

func (b *planBuilder) startResource(address string, action terraform.ActionType) {
	b.flushCurrent()
	b.current = &terraform.ResourceChange{
		Address: address,
		Action:  action,
	}
	b.before = make(map[string]any)
	b.after = make(map[string]any)
	b.afterUnknown = make(map[string]any)
	b.pathStack = nil
}

func (b *planBuilder) flushCurrent() {
	if b.current == nil {
		return
	}
	if len(b.before) > 0 || len(b.after) > 0 || len(b.afterUnknown) > 0 {
		b.current.Change = &terraform.Change{
			Before:       b.before,
			After:        b.after,
			AfterUnknown: b.afterUnknown,
		}
	}
	b.plan.Resources = append(b.plan.Resources, *b.current)
	b.current = nil
}

func (b *planBuilder) finish() *terraform.Plan {
	b.flushCurrent()
	return b.plan
}

func (b *planBuilder) parseResourceLine(line string) {
	start := strings.Index(line, "resource \"")
	if start == -1 {
		return
	}
	sub := line[start:]
	parts := strings.Split(sub, "\"")
	if len(parts) < 4 {
		return
	}
	if b.current != nil {
		b.current.ResourceType = parts[1]
		b.current.ResourceName = parts[3]
	}
}

func (b *planBuilder) pushPath(key string) {
	b.pathStack = append(b.pathStack, key)
}

func (b *planBuilder) popPath() {
	if len(b.pathStack) == 0 {
		return
	}
	b.pathStack = b.pathStack[:len(b.pathStack)-1]
}

func (b *planBuilder) setBefore(path []string, value any) {
	setPathValue(b.before, path, value)
}

func (b *planBuilder) setAfter(path []string, value any) {
	setPathValue(b.after, path, value)
}

func (b *planBuilder) setAfterUnknown(path []string) {
	setPathValue(b.afterUnknown, path, true)
}

func (b *planBuilder) applyHeredocValue(key, prefix, value string) {
	path := append([]string{}, b.pathStack...)
	path = append(path, key)
	switch prefix {
	case "+":
		b.setAfter(path, value)
	case "-":
		b.setBefore(path, value)
	case "~":
		b.setAfter(path, value)
	default:
		b.setAfter(path, value)
	}
}

func parseHeaderLine(line string) (string, terraform.ActionType, bool) {
	if !strings.HasPrefix(line, "# ") {
		return "", terraform.ActionNoOp, false
	}
	content := strings.TrimPrefix(line, "# ")

	actions := []struct {
		suffix string
		action terraform.ActionType
	}{
		{suffix: "will be created", action: terraform.ActionCreate},
		{suffix: "will be updated in-place", action: terraform.ActionUpdate},
		{suffix: "will be destroyed", action: terraform.ActionDelete},
		{suffix: "will be read during apply", action: terraform.ActionRead},
		{suffix: "must be replaced", action: terraform.ActionReplace},
		{suffix: "will be replaced", action: terraform.ActionReplace},
		{suffix: "will be destroyed and then created", action: terraform.ActionReplace},
	}

	for _, action := range actions {
		if idx := strings.Index(content, action.suffix); idx != -1 {
			address := strings.TrimSpace(content[:idx])
			if address == "" {
				return "", terraform.ActionNoOp, false
			}
			return address, action.action, true
		}
	}

	return "", terraform.ActionNoOp, false
}

func parseActionPrefix(line string) (string, string) {
	trimmed := strings.TrimLeftFunc(line, func(r rune) bool {
		return r == ' ' || r == 9
	})
	prefixes := []string{"-/+", "+", "-", "~"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			return prefix, rest
		}
	}
	return "", trimmed
}

func parseKeyValue(rest string) (string, string, bool) {
	parts := strings.SplitN(rest, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	key = strings.Trim(key, "\"")
	value := strings.TrimSpace(parts[1])
	return key, value, true
}

func parseHeredocDelimiter(value string) (string, bool) {
	if !strings.HasPrefix(value, "<<") {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "<<")
	trimmed = strings.TrimPrefix(trimmed, "-")
	if trimmed == "" {
		return "", false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", false
	}
	return fields[0], true
}

func splitArrow(value string) (string, string) {
	parts := strings.SplitN(value, "->", 2)
	if len(parts) != 2 {
		return value, ""
	}
	before := strings.TrimSpace(stripInlineComment(parts[0]))
	after := strings.TrimSpace(stripInlineComment(parts[1]))
	return before, after
}

func stripInlineComment(value string) string {
	if idx := strings.Index(value, " #"); idx != -1 {
		return strings.TrimSpace(value[:idx])
	}
	return strings.TrimSpace(value)
}

func isBlockOpen(value string) bool {
	return strings.HasSuffix(value, "{") || strings.HasSuffix(value, "[")
}

func isBlockClose(value string) bool {
	return value == "}" || value == "]" || strings.HasSuffix(value, "}")
}

func parseTerraformValue(value string) (any, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(value, ","))
	if value == "(known after apply)" {
		return nil, true
	}
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return strings.Trim(value, "\""), false
	}
	switch value {
	case "true":
		return true, false
	case "false":
		return false, false
	case "null":
		return nil, false
	}
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		return num, false
	}
	return value, false
}

func setPathValue(root map[string]any, path []string, value any) {
	if len(path) == 0 {
		return
	}
	current := root
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[key] = next
		}
		current = next
	}
	current[path[len(path)-1]] = value
}
