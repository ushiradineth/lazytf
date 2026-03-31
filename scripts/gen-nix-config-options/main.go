package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type schemaNode struct {
	Type                 string                 `json:"type"`
	Description          string                 `json:"description"`
	Properties           map[string]*schemaNode `json:"properties"`
	Items                *schemaNode            `json:"items"`
	AdditionalProperties *schemaNode            `json:"additionalProperties"`
}

func main() {
	repo, err := repoRoot()
	if err != nil {
		panic(err)
	}

	rootPath := filepath.Join(repo, "internal", "config", "config.schema.json")
	data, err := os.ReadFile(rootPath)
	if err != nil {
		panic(err)
	}

	var root schemaNode
	if err := json.Unmarshal(data, &root); err != nil {
		panic(err)
	}

	outPath := filepath.Join(repo, "nix", "modules", "generated", "config-options.nix")
	content := renderOptionsFile(root.Properties)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o600); err != nil {
		panic(err)
	}

	fmt.Printf("wrote %s\n", outPath)
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
		wd = parent
	}
}

func renderOptionsFile(properties map[string]*schemaNode) string {
	var b strings.Builder
	b.WriteString("{ lib }:\n")
	b.WriteString("{\n")
	b.WriteString(renderProperties(properties, 1))
	b.WriteString("}\n")
	return b.String()
}

func renderProperties(properties map[string]*schemaNode, indent int) string {
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		node := properties[key]
		b.WriteString(ind(indent))
		b.WriteString("\"")
		b.WriteString(key)
		b.WriteString("\" = lib.mkOption {\n")
		b.WriteString(ind(indent + 1))
		b.WriteString("type = lib.types.nullOr (")
		b.WriteString(renderTypeExpr(node, indent+1))
		b.WriteString(");\n")
		b.WriteString(ind(indent + 1))
		b.WriteString("default = null;\n")
		b.WriteString(ind(indent + 1))
		description := strings.TrimSpace(node.Description)
		if description == "" {
			description = "Auto-generated from internal/config/config.schema.json"
		}
		b.WriteString("description = ")
		b.WriteString(nixString(description))
		b.WriteString(";\n")
		b.WriteString(ind(indent))
		b.WriteString("};\n")
	}
	return b.String()
}

func renderTypeExpr(node *schemaNode, indent int) string {
	if node == nil {
		return "lib.types.anything"
	}

	switch node.Type {
	case "string":
		return "lib.types.str"
	case "integer":
		return "lib.types.int"
	case "number":
		return "lib.types.float"
	case "boolean":
		return "lib.types.bool"
	case "array":
		return "lib.types.listOf (" + renderTypeExpr(node.Items, indent) + ")"
	case "object":
		if len(node.Properties) == 0 && node.AdditionalProperties != nil {
			return "lib.types.attrsOf (" + renderTypeExpr(node.AdditionalProperties, indent) + ")"
		}

		var b strings.Builder
		b.WriteString("lib.types.submodule {\n")
		if node.AdditionalProperties != nil {
			b.WriteString(ind(indent + 1))
			b.WriteString("freeformType = lib.types.attrsOf (")
			b.WriteString(renderTypeExpr(node.AdditionalProperties, indent+1))
			b.WriteString(");\n")
		}
		b.WriteString(ind(indent + 1))
		b.WriteString("options = {\n")
		b.WriteString(renderProperties(node.Properties, indent+2))
		b.WriteString(ind(indent + 1))
		b.WriteString("};\n")
		b.WriteString(ind(indent))
		b.WriteString("}")
		return b.String()
	default:
		return "lib.types.anything"
	}
}

func ind(level int) string {
	return strings.Repeat("  ", level)
}

func nixString(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", " ")
	return "\"" + escaped + "\""
}
