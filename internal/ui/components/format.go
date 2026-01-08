package components

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ushiradineth/tftui/internal/diff"
)

func formatValue(val interface{}) string {
	if val == nil {
		return "(null)"
	}

	if s, ok := val.(string); ok {
		if len(s) > 200 {
			return fmt.Sprintf("%q...", s[:197])
		}
		return fmt.Sprintf(`"%s"`, s)
	}

	if _, ok := val.(diff.UnknownValue); ok {
		return "(known after apply)"
	}

	if isMap(val) {
		return "{...}"
	}
	if isList(val) {
		if asList := interfaceToList(val); len(asList) == 1 {
			if s, ok := asList[0].(string); ok {
				return formatValue(s)
			}
		}
		return "[...]"
	}

	return fmt.Sprintf("%v", val)
}

func isMap(val interface{}) bool {
	if val == nil {
		return false
	}
	return reflect.TypeOf(val).Kind() == reflect.Map
}

func isList(val interface{}) bool {
	if val == nil {
		return false
	}
	kind := reflect.TypeOf(val).Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

func interfaceToList(val interface{}) []interface{} {
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}

	result := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}
	return result
}

func formatMultilineStringDiff(path, before, after string) string {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	if len(beforeLines) != len(afterLines) {
		return ""
	}

	diffIndexes := make([]int, 0, 4)
	for i := range beforeLines {
		if beforeLines[i] != afterLines[i] {
			diffIndexes = append(diffIndexes, i)
			if len(diffIndexes) >= 4 {
				break
			}
		}
	}
	if len(diffIndexes) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  ~ %s:", path))
	for _, idx := range diffIndexes {
		oldLine := stripListMarker(strings.TrimSpace(beforeLines[idx]))
		newLine := stripListMarker(strings.TrimSpace(afterLines[idx]))
		b.WriteString("\n")
		b.WriteString("    - ")
		b.WriteString(truncateLine(oldLine, 140))
		b.WriteString("\n")
		b.WriteString("    + ")
		b.WriteString(truncateLine(newLine, 140))
	}
	return b.String()
}

func truncateLine(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func stripListMarker(line string) string {
	if strings.HasPrefix(line, "- ") {
		return strings.TrimSpace(line[2:])
	}
	return line
}
