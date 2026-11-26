package parser

import "strings"

// collectionFrame keeps list elements separate from the surrounding map path.
// Objects inside a list need their own frame so adjacent objects do not merge.
type collectionFrame struct {
	key        string
	prefix     string
	before     collectionSide
	after      collectionSide
	hasUnknown bool
}

func newCollectionFrame(key, prefix, opener string) *collectionFrame {
	frame := &collectionFrame{key: key, prefix: prefix}
	if opener == "[" {
		frame.before.list = []any{}
		frame.after.list = []any{}
	} else {
		frame.before.object = map[string]any{}
		frame.after.object = map[string]any{}
	}
	return frame
}

func (b *planBuilder) consumeCollectionLine(line string) {
	if strings.HasPrefix(line, "#") {
		return
	}
	frame := b.collections[len(b.collections)-1]
	prefix, rest := collectionActionPrefix(line)
	if prefix == "" {
		prefix = frame.prefix
	}
	rest = strings.TrimSpace(strings.TrimSuffix(stripCollectionComment(rest), ","))
	if rest == "]" || rest == "}" || strings.HasPrefix(rest, "] ->") || strings.HasPrefix(rest, "} ->") {
		b.closeCollection()
		return
	}

	key, value := "", rest
	if frame.before.object != nil {
		var ok bool
		key, value, ok = parseKeyValue(rest)
		if !ok {
			return
		}
	}
	if delimiter, ok := parseHeredocDelimiter(value); ok {
		b.startHeredoc(key, prefix, delimiter)
		return
	}
	if value == "[" || value == "{" {
		b.collections = append(b.collections, newCollectionFrame(key, prefix, value))
		return
	}

	beforeText, afterText, arrow := splitCollectionArrow(value)
	before, _ := parseTerraformValue(beforeText)
	after, unknown := parseTerraformValue(afterText)
	if arrow {
		prefix = "~"
	}
	frame.add(key, prefix, before, after, unknown)
}

func (b *planBuilder) closeCollection() {
	last := len(b.collections) - 1
	frame := b.collections[last]
	b.collections = b.collections[:last]
	if last > 0 {
		parent := b.collections[last-1]
		parent.add(frame.key, frame.prefix, frame.before.value(), frame.after.value(), frame.hasUnknown)
		return
	}
	path := append([]string{}, b.pathStack...)
	path = append(path, frame.key)
	if frame.prefix != "+" {
		b.setBefore(path, frame.before.value())
	}
	if frame.prefix != "-" {
		b.setAfter(path, frame.after.value())
		if frame.hasUnknown {
			// Text output can elide indices; mark the collection conservatively.
			b.setAfterUnknown(path)
		}
	}
}

func (f *collectionFrame) add(key, prefix string, before, after any, unknown bool) {
	if prefix != "+" {
		f.before.add(key, before)
	}
	if prefix != "-" {
		f.after.add(key, after)
		f.hasUnknown = f.hasUnknown || unknown
	}
}

type collectionSide struct {
	list   []any
	object map[string]any
}

func (s *collectionSide) add(key string, value any) {
	if s.object != nil {
		s.object[key] = value
	} else {
		s.list = append(s.list, value)
	}
}

func (s *collectionSide) value() any {
	if s.object != nil {
		return s.object
	}
	return s.list
}

func collectionActionPrefix(line string) (string, string) {
	for _, prefix := range []string{"+", "-", "~"} {
		if strings.HasPrefix(line, prefix+" ") {
			return prefix, strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return "", line
}

// Only arrows outside quoted strings separate before and after values.
func splitCollectionArrow(value string) (before, after string, arrow bool) {
	if i := collectionTokenIndex(value, "->"); i != -1 {
		return strings.TrimSpace(value[:i]), strings.TrimSpace(value[i+2:]), true
	}
	return value, value, false
}

func stripCollectionComment(value string) string {
	if i := collectionTokenIndex(value, " #"); i != -1 {
		return strings.TrimSpace(value[:i])
	}
	return strings.TrimSpace(value)
}

func collectionTokenIndex(value, token string) int {
	quoted, escaped := false, false
	for i := 0; i < len(value); i++ {
		switch {
		case escaped:
			escaped = false
		case value[i] == '\\' && quoted:
			escaped = true
		case value[i] == '"':
			quoted = !quoted
		case !quoted && strings.HasPrefix(value[i:], token):
			return i
		}
	}
	return -1
}
