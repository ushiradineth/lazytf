package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/styles"
)

const (
	jsonSchemaDialectURL = "https://json-schema.org/draft/2020-12/schema"
	mainSchemaURL        = "https://raw.githubusercontent.com/ushiradineth/lazytf/main/internal/config/config.schema.json"
	schemaTagURLTemplate = "https://raw.githubusercontent.com/ushiradineth/lazytf/v%s/internal/config/config.schema.json"
	schemaTypeObject     = "object"
)

type orderedMap map[string]any

func (m orderedMap) MarshalJSON() ([]byte, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(normalize(m[k]))
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(val)
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}

func normalize(value any) any {
	switch v := value.(type) {
	case map[string]any:
		normalized := make(orderedMap, len(v))
		for key, val := range v {
			normalized[key] = normalize(val)
		}
		return normalized
	case []any:
		for i := range v {
			v[i] = normalize(v[i])
		}
		return v
	default:
		return value
	}
}

type schemaNode struct {
	Type                 string                 `json:"type"`
	Description          string                 `json:"description"`
	Properties           map[string]*schemaNode `json:"properties"`
	Items                *schemaNode            `json:"items"`
	AdditionalProperties *schemaNode            `json:"additionalProperties"`
	Enum                 []string               `json:"enum"`
}

type docRow struct {
	Path        string
	Type        string
	Default     string
	Description string
}

func main() {
	repo, err := repoRoot()
	if err != nil {
		panic(err)
	}

	schemaURL := taggedSchemaURL(repo)
	if schemaURL == "" {
		schemaURL = mainSchemaURL
	}

	schemaData, rootNode, err := generateSchema(schemaURL)
	if err != nil {
		panic(err)
	}

	schemaPath := filepath.Join(repo, "internal", "config", "config.schema.json")
	if err := os.WriteFile(schemaPath, schemaData, 0o600); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s\n", schemaPath)

	nixPath := filepath.Join(repo, "nix", "modules", "generated", "config-options.nix")
	if err := os.MkdirAll(filepath.Dir(nixPath), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(nixPath, []byte(renderNixOptionsFile(rootNode.Properties)), 0o600); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s\n", nixPath)

	configDocPath := filepath.Join(repo, "CONFIGURATION.md")
	if err := os.WriteFile(configDocPath, []byte(renderConfigDoc(rootNode, schemaURL)), 0o600); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s\n", configDocPath)
}

func generateSchema(schemaURL string) ([]byte, schemaNode, error) {
	root := map[string]any{
		"$schema": jsonSchemaDialectURL,
		"$id":     schemaURL,
		"title":   "lazytf config",
		"type":    schemaTypeObject,
	}
	root["properties"] = schemaForType(reflect.TypeOf(config.Config{}))["properties"]

	schema := normalize(root)
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, schemaNode{}, err
	}
	data = append(data, '\n')

	var parsed schemaNode
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, schemaNode{}, err
	}
	return data, parsed, nil
}

func taggedSchemaURL(repo string) string {
	versionPath := filepath.Join(repo, "VERSION")
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		return ""
	}
	version := strings.TrimSpace(string(versionData))
	if !isStrictSemver(version) {
		return ""
	}
	return fmt.Sprintf(schemaTagURLTemplate, version)
}

func isStrictSemver(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
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

func schemaForType(t reflect.Type) map[string]any {
	t = dereferenceType(t)
	if t == reflect.TypeOf(time.Duration(0)) {
		return map[string]any{"type": "string"}
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": schemaForType(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 schemaTypeObject,
			"additionalProperties": schemaForType(t.Elem()),
		}
	case reflect.Struct:
		return schemaForStruct(t)
	default:
		return map[string]any{"type": "string"}
	}
}

func dereferenceType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func schemaForStruct(t reflect.Type) map[string]any {
	properties := map[string]any{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name, include := schemaFieldName(field)
		if !include {
			continue
		}

		fieldSchema := schemaForType(field.Type)
		if description := strings.TrimSpace(field.Tag.Get("description")); description != "" {
			fieldSchema["description"] = description
		}
		if isThemeField(t, field, name) {
			fieldSchema["enum"] = styles.BuiltInThemeNames()
		}
		properties[name] = fieldSchema
	}

	return map[string]any{
		"type":       schemaTypeObject,
		"properties": properties,
	}
}

func schemaFieldName(field reflect.StructField) (string, bool) {
	if field.PkgPath != "" || field.Tag.Get("schema") == "-" {
		return "", false
	}

	tag := field.Tag.Get("yaml")
	if tag == "-" {
		return "", false
	}

	name := strings.Split(tag, ",")[0]
	if name == "" {
		name = strings.ToLower(field.Name)
	}
	return name, true
}

func isThemeField(parent reflect.Type, field reflect.StructField, name string) bool {
	if parent == reflect.TypeOf(config.ThemeConfig{}) && name == "name" {
		return true
	}
	if field.Type.Kind() != reflect.String {
		return false
	}
	return (parent == reflect.TypeOf(config.EnvironmentPreset{}) || parent == reflect.TypeOf(config.ProjectConfig{})) && name == "theme"
}

func renderNixOptionsFile(properties map[string]*schemaNode) string {
	var b strings.Builder
	b.WriteString("{ lib }:\n")
	b.WriteString("{\n")
	b.WriteString(renderNixProperties(properties, 1))
	b.WriteString("}\n")
	return b.String()
}

func renderNixProperties(properties map[string]*schemaNode, indent int) string {
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		node := properties[key]
		b.WriteString(indentNix(indent))
		b.WriteString("\"")
		b.WriteString(key)
		b.WriteString("\" = lib.mkOption {\n")
		b.WriteString(indentNix(indent + 1))
		b.WriteString("type = lib.types.nullOr (")
		b.WriteString(renderNixTypeExpr(node, indent+1))
		b.WriteString(");\n")
		b.WriteString(indentNix(indent + 1))
		b.WriteString("default = null;\n")
		b.WriteString(indentNix(indent + 1))
		description := strings.TrimSpace(node.Description)
		if description == "" {
			description = "Auto-generated from internal/config/config.schema.json"
		}
		b.WriteString("description = ")
		b.WriteString(nixString(description))
		b.WriteString(";\n")
		b.WriteString(indentNix(indent))
		b.WriteString("};\n")
	}
	return b.String()
}

func renderNixTypeExpr(node *schemaNode, indent int) string {
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
		return "lib.types.listOf (" + renderNixTypeExpr(node.Items, indent) + ")"
	case schemaTypeObject:
		if len(node.Properties) == 0 && node.AdditionalProperties != nil {
			return "lib.types.attrsOf (" + renderNixTypeExpr(node.AdditionalProperties, indent) + ")"
		}

		var b strings.Builder
		b.WriteString("lib.types.submodule {\n")
		if node.AdditionalProperties != nil {
			b.WriteString(indentNix(indent + 1))
			b.WriteString("freeformType = lib.types.attrsOf (")
			b.WriteString(renderNixTypeExpr(node.AdditionalProperties, indent+1))
			b.WriteString(");\n")
		}
		b.WriteString(indentNix(indent + 1))
		b.WriteString("options = {\n")
		b.WriteString(renderNixProperties(node.Properties, indent+2))
		b.WriteString(indentNix(indent + 1))
		b.WriteString("};\n")
		b.WriteString(indentNix(indent))
		b.WriteString("}")
		return b.String()
	default:
		return "lib.types.anything"
	}
}

func indentNix(level int) string {
	return strings.Repeat("  ", level)
}

func nixString(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", " ")
	return "\"" + escaped + "\""
}

func renderConfigDoc(root schemaNode, schemaURL string) string {
	rows := flattenDocRows("", &root, defaultValueByPath())
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Path < rows[j].Path
	})

	var b strings.Builder
	b.WriteString("# lazytf configuration reference\n\n")
	b.WriteString("_Generated by `go generate ./internal/config` from `internal/config/config.go`. Do not edit manually._\n\n")
	b.WriteString("- Schema URL for current VERSION: `" + schemaURL + "`\n")
	b.WriteString("- Main branch schema URL: `" + mainSchemaURL + "`\n\n")
	b.WriteString("## Configuration paths\n\n")
	b.WriteString("| Path | Type | Default | Description |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, row := range rows {
		b.WriteString("| `")
		b.WriteString(escapeMarkdownTable(row.Path))
		b.WriteString("` | ")
		b.WriteString(escapeMarkdownTable(row.Type))
		b.WriteString(" | ")
		b.WriteString(escapeMarkdownTable(row.Default))
		b.WriteString(" | ")
		b.WriteString(escapeMarkdownTable(row.Description))
		b.WriteString(" |\n")
	}

	return b.String()
}

func flattenDocRows(prefix string, node *schemaNode, defaults map[string]string) []docRow {
	if node == nil {
		return nil
	}

	rows := make([]docRow, 0)
	if prefix != "" {
		rows = append(rows, docRow{
			Path:        prefix,
			Type:        docType(node),
			Default:     defaultFor(prefix, defaults),
			Description: strings.TrimSpace(node.Description),
		})
	}

	if len(node.Properties) > 0 {
		keys := make([]string, 0, len(node.Properties))
		for key := range node.Properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childPath := key
			if prefix != "" {
				childPath = prefix + "." + key
			}
			rows = append(rows, flattenDocRows(childPath, node.Properties[key], defaults)...)
		}
	}

	if node.Items != nil && len(node.Items.Properties) > 0 {
		rows = append(rows, flattenDocRows(prefix+"[]", node.Items, defaults)...)
	}

	if node.AdditionalProperties != nil {
		wildcardPath := "*"
		if prefix != "" {
			wildcardPath = prefix + ".*"
		}
		rows = append(rows, flattenDocRows(wildcardPath, node.AdditionalProperties, defaults)...)
	}

	return rows
}

func docType(node *schemaNode) string {
	if node == nil {
		return "unknown"
	}
	switch node.Type {
	case "array":
		return "array<" + docType(node.Items) + ">"
	case schemaTypeObject:
		if len(node.Properties) > 0 {
			return "object"
		}
		if node.AdditionalProperties != nil {
			return "map<" + docType(node.AdditionalProperties) + ">"
		}
		return "object"
	default:
		if len(node.Enum) > 0 {
			return node.Type + " (enum)"
		}
		if node.Type == "" {
			return "unknown"
		}
		return node.Type
	}
}

func defaultValueByPath() map[string]string {
	defaults := make(map[string]string)
	collectDefaultValues(reflect.ValueOf(config.DefaultBootstrapConfig()), "", defaults)
	return defaults
}

func collectDefaultValues(value reflect.Value, path string, out map[string]string) {
	if !value.IsValid() {
		return
	}

	t := value.Type()
	for t.Kind() == reflect.Ptr {
		if value.IsNil() {
			return
		}
		value = value.Elem()
		t = value.Type()
	}

	if t == reflect.TypeOf(time.Duration(0)) {
		if path != "" {
			out[path] = value.Interface().(time.Duration).String()
		}
		return
	}

	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name, include := schemaFieldName(field)
			if !include {
				continue
			}
			childPath := name
			if path != "" {
				childPath = path + "." + name
			}
			collectDefaultValues(value.Field(i), childPath, out)
		}
	case reflect.Bool:
		if path != "" {
			out[path] = strconv.FormatBool(value.Bool())
		}
	case reflect.String:
		if path != "" && value.String() != "" {
			out[path] = value.String()
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if path != "" {
			out[path] = strconv.FormatInt(value.Int(), 10)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if path != "" {
			out[path] = strconv.FormatUint(value.Uint(), 10)
		}
	case reflect.Slice, reflect.Array:
		if path == "" || value.Len() == 0 {
			return
		}
		encoded, err := json.Marshal(value.Interface())
		if err != nil {
			return
		}
		out[path] = string(encoded)
	case reflect.Map:
		if path == "" || value.Len() == 0 {
			return
		}
		encoded, err := json.Marshal(value.Interface())
		if err != nil {
			return
		}
		out[path] = string(encoded)
	}
}

func defaultFor(path string, defaults map[string]string) string {
	if value, ok := defaults[path]; ok {
		return value
	}
	return "-"
}

func escapeMarkdownTable(value string) string {
	escaped := strings.ReplaceAll(strings.TrimSpace(value), "|", "\\|")
	escaped = strings.ReplaceAll(escaped, "\n", " ")
	if escaped == "" {
		return "-"
	}
	return escaped
}
