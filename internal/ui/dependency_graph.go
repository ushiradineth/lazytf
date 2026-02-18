package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

// BuildDependencyGraphView renders a text dependency graph for plan resources.
func BuildDependencyGraphView(resources []terraform.ResourceChange) string {
	if len(resources) == 0 {
		return "No resources in plan."
	}

	lines := make([]string, 0, len(resources)*3)
	for i := range resources {
		resource := resources[i]
		lines = append(lines, fmt.Sprintf("%s [%s]", resource.Address, resource.Action))
		deps := dependencyList(resource)
		if len(deps) == 0 {
			lines = append(lines, "  (no dependencies)")
			continue
		}
		for _, dep := range deps {
			lines = append(lines, "  -> "+dep)
		}
	}

	return strings.Join(lines, "\n")
}

func dependencyList(resource terraform.ResourceChange) []string {
	if resource.Change == nil {
		return nil
	}
	deps := extractDependencies(resource.Change.After)
	if len(deps) == 0 {
		deps = extractDependencies(resource.Change.Before)
	}
	if len(deps) == 0 {
		return nil
	}

	unique := make(map[string]struct{}, len(deps))
	for _, dep := range deps {
		trimmed := strings.TrimSpace(dep)
		if trimmed == "" {
			continue
		}
		unique[trimmed] = struct{}{}
	}

	result := make([]string, 0, len(unique))
	for dep := range unique {
		result = append(result, dep)
	}
	sort.Strings(result)
	return result
}

func extractDependencies(obj map[string]any) []string {
	if obj == nil {
		return nil
	}
	raw, ok := obj["depends_on"]
	if !ok {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	deps := make([]string, 0, len(items))
	for _, item := range items {
		dep, ok := item.(string)
		if !ok {
			continue
		}
		deps = append(deps, dep)
	}
	return deps
}
