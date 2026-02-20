package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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

type stateDependencyNode struct {
	Address      string
	Dependencies []string
}

type statePullPayload struct {
	Resources []statePullResource `json:"resources"`
}

type statePullResource struct {
	Module    string              `json:"module"`
	Mode      string              `json:"mode"`
	Type      string              `json:"type"`
	Name      string              `json:"name"`
	Instances []statePullInstance `json:"instances"`
}

type statePullInstance struct {
	IndexKey     any      `json:"index_key"`
	Dependencies []string `json:"dependencies"`
}

// BuildDependencyGraphViewFromStateJSON renders a text dependency graph for state resources.
func BuildDependencyGraphViewFromStateJSON(stateJSON string) (string, error) {
	nodes, err := parseStateDependencyNodes(stateJSON)
	if err != nil {
		return "", err
	}
	if len(nodes) == 0 {
		return "No resources in state.", nil
	}

	lines := make([]string, 0, len(nodes)*3)
	for i := range nodes {
		node := nodes[i]
		lines = append(lines, node.Address)
		if len(node.Dependencies) == 0 {
			lines = append(lines, "  (no dependencies)")
			continue
		}
		for _, dep := range node.Dependencies {
			lines = append(lines, "  -> "+dep)
		}
	}

	return strings.Join(lines, "\n"), nil
}

func parseStateDependencyNodes(stateJSON string) ([]stateDependencyNode, error) {
	var payload statePullPayload
	if err := json.Unmarshal([]byte(stateJSON), &payload); err != nil {
		return nil, fmt.Errorf("parse state pull json: %w", err)
	}

	byAddress := make(map[string]map[string]struct{})
	for i := range payload.Resources {
		accumulateStateResourceNodes(byAddress, payload.Resources[i])
	}

	addresses := make([]string, 0, len(byAddress))
	for address := range byAddress {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	nodes := make([]stateDependencyNode, 0, len(addresses))
	for _, address := range addresses {
		depSet := byAddress[address]
		deps := make([]string, 0, len(depSet))
		for dep := range depSet {
			deps = append(deps, dep)
		}
		sort.Strings(deps)
		nodes = append(nodes, stateDependencyNode{Address: address, Dependencies: deps})
	}
	return nodes, nil
}

func accumulateStateResourceNodes(byAddress map[string]map[string]struct{}, resource statePullResource) {
	if resource.Mode != "managed" {
		return
	}
	if len(resource.Instances) == 0 {
		address := stateResourceAddress(resource.Module, resource.Type, resource.Name, nil)
		ensureDependencySet(byAddress, address)
		return
	}

	for j := range resource.Instances {
		instance := resource.Instances[j]
		address := stateResourceAddress(resource.Module, resource.Type, resource.Name, instance.IndexKey)
		deps := ensureDependencySet(byAddress, address)
		for _, dep := range instance.Dependencies {
			trimmed := strings.TrimSpace(dep)
			if trimmed != "" {
				deps[trimmed] = struct{}{}
			}
		}
	}
}

func ensureDependencySet(byAddress map[string]map[string]struct{}, address string) map[string]struct{} {
	deps, ok := byAddress[address]
	if ok {
		return deps
	}
	deps = make(map[string]struct{})
	byAddress[address] = deps
	return deps
}

func stateResourceAddress(module, resourceType, name string, indexKey any) string {
	base := resourceType + "." + name
	if strings.TrimSpace(module) != "" {
		base = module + "." + base
	}
	if indexKey == nil {
		return base
	}

	switch key := indexKey.(type) {
	case float64:
		if key == float64(int64(key)) {
			return base + "[" + strconv.FormatInt(int64(key), 10) + "]"
		}
		return base + "[\"" + strconv.FormatFloat(key, 'f', -1, 64) + "\"]"
	case string:
		return base + "[\"" + key + "\"]"
	case bool:
		return base + "[\"" + strconv.FormatBool(key) + "\"]"
	default:
		return base + "[\"" + fmt.Sprint(key) + "\"]"
	}
}
