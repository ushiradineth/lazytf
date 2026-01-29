package diff

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// CalculateDiffs computes the minimal set of changed attributes between before and after states.
// This powers the diff viewer and resource summaries.
func CalculateDiffs(before, after, afterUnknown map[string]any, beforeOrder, afterOrder, afterUnknownOrder map[string][]string, pathPointer string) []MinimalDiff {
	diffs := []MinimalDiff{}

	// Collect all unique keys from both maps
	allKeys := orderedKeys(before, after, afterUnknown, beforeOrder, afterOrder, afterUnknownOrder, pathPointer)

	for _, key := range allKeys {
		beforeVal, beforeExists := before[key]
		afterVal, afterExists := after[key]
		unknownVal, unknownExists := afterUnknown[key]

		if unknownExists && isUnknown(unknownVal) && deepEqual(beforeVal, afterVal) {
			continue
		}

		if unknownExists && isUnknown(unknownVal) && (!afterExists || afterVal == nil) {
			afterVal = UnknownValue{}
			afterExists = true
		}

		switch {
		case !beforeExists:
			// New attribute added
			diffs = append(diffs, MinimalDiff{
				Path:     []string{key},
				OldValue: nil,
				NewValue: afterVal,
				Action:   DiffAdd,
			})
		case !afterExists:
			// Attribute removed
			diffs = append(diffs, MinimalDiff{
				Path:     []string{key},
				OldValue: beforeVal,
				NewValue: nil,
				Action:   DiffRemove,
			})
		case isUnknownValue(afterVal):
			// Value is known after apply
			diffs = append(diffs, MinimalDiff{
				Path:     []string{key},
				OldValue: beforeVal,
				NewValue: afterVal,
				Action:   DiffChange,
			})
		case !deepEqual(beforeVal, afterVal):
			// Attribute changed - check if we need to recurse
			switch {
			case isMap(beforeVal) && isMap(afterVal):
				// Recurse into nested objects
				unknownMap := toMap(unknownVal)
				nextPath := joinJSONPointer(pathPointer, key)
				nestedDiffs := CalculateDiffs(
					toMap(beforeVal),
					toMap(afterVal),
					unknownMap,
					beforeOrder,
					afterOrder,
					afterUnknownOrder,
					nextPath,
				)
				for _, nd := range nestedDiffs {
					nd.Path = append([]string{key}, nd.Path...)
					diffs = append(diffs, nd)
				}
			case isList(beforeVal) && isList(afterVal):
				// Handle lists/arrays
				listDiffs := calculateListDiff(key, beforeVal, afterVal)
				diffs = append(diffs, listDiffs...)
			default:
				// Simple value change
				diffs = append(diffs, MinimalDiff{
					Path:     []string{key},
					OldValue: beforeVal,
					NewValue: afterVal,
					Action:   DiffChange,
				})
			}
		}
		// If values are equal, no diff is added
	}

	return diffs
}

// unionKeys returns all unique keys from two maps, sorted alphabetically
func orderedKeys(before, after, afterUnknown map[string]any, beforeOrder, afterOrder, afterUnknownOrder map[string][]string, pathPointer string) []string {
	keySet := make(map[string]bool)
	ordered := make([]string, 0)

	appendOrder := func(order map[string][]string) {
		if order == nil {
			return
		}
		for _, key := range order[pathPointer] {
			if !keySet[key] {
				keySet[key] = true
				ordered = append(ordered, key)
			}
		}
	}

	appendOrder(beforeOrder)
	appendOrder(afterOrder)
	appendOrder(afterUnknownOrder)

	remaining := make([]string, 0)
	for k := range before {
		if !keySet[k] {
			keySet[k] = true
			remaining = append(remaining, k)
		}
	}
	for k := range after {
		if !keySet[k] {
			keySet[k] = true
			remaining = append(remaining, k)
		}
	}
	for k := range afterUnknown {
		if !keySet[k] {
			keySet[k] = true
			remaining = append(remaining, k)
		}
	}

	sort.Strings(remaining)
	return append(ordered, remaining...)
}

// deepEqual checks if two values are deeply equal
func deepEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

func isUnknown(val any) bool {
	switch v := val.(type) {
	case bool:
		return v
	case map[string]any:
		for _, sub := range v {
			if isUnknown(sub) {
				return true
			}
		}
		return false
	case []any:
		for _, sub := range v {
			if isUnknown(sub) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func isUnknownValue(val any) bool {
	_, ok := val.(UnknownValue)
	return ok
}

// isMap checks if a value is a map
func isMap(val any) bool {
	if val == nil {
		return false
	}
	v := reflect.ValueOf(val)
	return v.Kind() == reflect.Map
}

// isList checks if a value is a list/array
func isList(val any) bool {
	if val == nil {
		return false
	}
	v := reflect.ValueOf(val)
	return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}

// toMap converts an any to map[string]any
func toMap(val any) map[string]any {
	if m, ok := val.(map[string]any); ok {
		return m
	}
	return nil
}

// calculateListDiff handles differences in lists/arrays
func calculateListDiff(key string, before, after any) []MinimalDiff {
	beforeList := interfaceToList(before)
	afterList := interfaceToList(after)

	if len(beforeList) == len(afterList) && len(beforeList) == 1 && allStrings(beforeList) && allStrings(afterList) {
		diffs := []MinimalDiff{}
		for i := range beforeList {
			if !deepEqual(beforeList[i], afterList[i]) {
				diffs = append(diffs, MinimalDiff{
					Path:     []string{fmt.Sprintf("%s[%d]", key, i)},
					OldValue: beforeList[i],
					NewValue: afterList[i],
					Action:   DiffChange,
				})
			}
		}
		if len(diffs) > 0 {
			return diffs
		}
	}

	if len(beforeList) > 0 && len(afterList) > 0 && allStrings(beforeList) && allStrings(afterList) {
		return stringListDiffs(key, interfaceToStrings(beforeList), interfaceToStrings(afterList))
	}

	// For simplicity in MVP, treat list changes as a single change
	// Future enhancement: use LCS algorithm for better granularity
	if !deepEqual(beforeList, afterList) {
		return []MinimalDiff{{
			Path:     []string{key},
			OldValue: before,
			NewValue: after,
			Action:   DiffChange,
		}}
	}

	return nil
}

// interfaceToList converts an any to a slice
func interfaceToList(val any) []any {
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}

	result := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}
	return result
}

func interfaceToStrings(list []any) []string {
	result := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func allStrings(list []any) bool {
	for _, item := range list {
		if _, ok := item.(string); !ok {
			return false
		}
	}
	return true
}

func stringListDiffs(key string, before, after []string) []MinimalDiff {
	type op struct {
		kind  string
		value string
	}

	m := len(before)
	n := len(after)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			switch {
			case before[i] == after[j]:
				dp[i][j] = dp[i+1][j+1] + 1
			case dp[i+1][j] >= dp[i][j+1]:
				dp[i][j] = dp[i+1][j]
			default:
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	ops := make([]op, 0, m+n)
	i, j := 0, 0
	for i < m && j < n {
		switch {
		case before[i] == after[j]:
			ops = append(ops, op{kind: "equal", value: before[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, op{kind: "remove", value: before[i]})
			i++
		default:
			ops = append(ops, op{kind: "add", value: after[j]})
			j++
		}
	}
	for i < m {
		ops = append(ops, op{kind: "remove", value: before[i]})
		i++
	}
	for j < n {
		ops = append(ops, op{kind: "add", value: after[j]})
		j++
	}

	var diffs []MinimalDiff
	beforeIndex := 0
	afterIndex := 0
	for _, op := range ops {
		switch op.kind {
		case "equal":
			beforeIndex++
			afterIndex++
		case "remove":
			diffs = append(diffs, MinimalDiff{
				Path:     []string{fmt.Sprintf("%s[%d]", key, beforeIndex)},
				OldValue: op.value,
				NewValue: nil,
				Action:   DiffRemove,
			})
			beforeIndex++
		case "add":
			diffs = append(diffs, MinimalDiff{
				Path:     []string{fmt.Sprintf("%s[%d]", key, afterIndex)},
				OldValue: nil,
				NewValue: op.value,
				Action:   DiffAdd,
			})
			afterIndex++
		}
	}

	return diffs
}

func joinJSONPointer(base, key string) string {
	segment := strings.ReplaceAll(key, "~", "~0")
	segment = strings.ReplaceAll(segment, "/", "~1")
	if base == "" {
		return "/" + segment
	}
	return base + "/" + segment
}

// FormatDiff returns a human-readable string representation of a diff
func FormatDiff(diff MinimalDiff) string {
	pathStr := formatPath(diff.Path)

	switch diff.Action {
	case DiffAdd:
		return fmt.Sprintf("  + %s: %v", pathStr, formatValue(diff.NewValue))
	case DiffRemove:
		return fmt.Sprintf("  - %s: %v", pathStr, formatValue(diff.OldValue))
	case DiffChange:
		return fmt.Sprintf("  ~ %s: %v → %v", pathStr, formatValue(diff.OldValue), formatValue(diff.NewValue))
	default:
		return "  ? " + pathStr
	}
}

// formatPath converts a path slice to dot notation
func formatPath(path []string) string {
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

// formatValue formats a value for display
func formatValue(val any) string {
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

	if _, ok := val.(UnknownValue); ok {
		return "(known after apply)"
	}

	// For complex types, use a more compact representation
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
