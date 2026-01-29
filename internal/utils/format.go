package utils

import "fmt"

// FormatValue formats a value for display.
func FormatValue(val any, unknownChecker func(any) bool) string {
	if val == nil {
		return "(null)"
	}

	// Special handling for strings to add quotes
	if s, ok := val.(string); ok {
		if len(s) > 200 {
			return fmt.Sprintf("%q...", s[:197])
		}
		return fmt.Sprintf("%q", s)
	}

	// Check for unknown values using the provided checker
	if unknownChecker != nil && unknownChecker(val) {
		return "(known after apply)"
	}

	// For complex types, use a more compact representation
	if IsMap(val) {
		return "{...}"
	}
	if IsList(val) {
		if asList := InterfaceToList(val); len(asList) == 1 {
			if s, ok := asList[0].(string); ok {
				return FormatValue(s, unknownChecker)
			}
		}
		return "[...]"
	}

	return fmt.Sprintf("%v", val)
}

// FormatPath converts a path slice to dot notation.
func FormatPath(path []string) string {
	result := ""
	for i, segment := range path {
		if i == 0 {
			result = segment
		} else {
			result += "." + segment
		}
	}
	return result
}

// TruncateMiddle truncates a string in the middle if it exceeds maxLen.
func TruncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	half := (maxLen - 3) / 2
	return s[:half] + "..." + s[len(s)-half:]
}

// TruncateEnd truncates a string at the end if it exceeds maxLen.
func TruncateEnd(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
