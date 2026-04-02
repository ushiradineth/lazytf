package components

import (
	"fmt"
	"strings"

	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/utils"
)

func formatValue(val any) string {
	if val == nil {
		return "(null)"
	}

	if s, ok := val.(string); ok {
		if len(s) > 200 {
			return fmt.Sprintf("%q...", s[:197])
		}
		return fmt.Sprintf("%q", s)
	}

	if _, ok := val.(diff.UnknownValue); ok {
		return "(known after apply)"
	}

	if utils.IsMap(val) {
		return "{...}"
	}
	if utils.IsList(val) {
		if asList := utils.InterfaceToList(val); len(asList) == 1 {
			if s, ok := asList[0].(string); ok {
				return formatValue(s)
			}
		}
		return "[...]"
	}

	return fmt.Sprintf("%v", val)
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
		b.WriteString(utils.TruncateEnd(oldLine, 140))
		b.WriteString("\n")
		b.WriteString("    + ")
		b.WriteString(utils.TruncateEnd(newLine, 140))
	}
	return b.String()
}

func stripListMarker(line string) string {
	if strings.HasPrefix(line, "- ") {
		return strings.TrimSpace(line[2:])
	}
	return line
}

func formatPathForDisplay(path []string) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, 0, len(path))
	for _, segment := range path {
		if strings.HasPrefix(segment, "__item_") {
			index := strings.TrimPrefix(segment, "__item_")
			if index == "" {
				index = "?"
			}
			itemToken := "[" + index + "]"
			if len(parts) > 0 {
				parts[len(parts)-1] += itemToken
			} else {
				parts = append(parts, itemToken)
			}
			continue
		}
		if strings.Contains(segment, ".") || strings.Contains(segment, " ") || strings.Contains(segment, "\"") || strings.Contains(segment, "\\") {
			parts = append(parts, fmt.Sprintf("%q", segment))
		} else {
			parts = append(parts, segment)
		}
	}
	if len(parts) == 0 {
		return "(list item)"
	}
	return strings.Join(parts, ".")
}
