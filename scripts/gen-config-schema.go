package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/styles"
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

func main() {
	root := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title":   "lazytf config",
		"type":    "object",
	}
	root["properties"] = schemaForType(reflect.TypeOf(config.Config{}))["properties"]

	schema := normalize(root)
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	repo, err := repoRoot()
	if err != nil {
		panic(err)
	}

	target := filepath.Join(repo, "internal", "config", "config.schema.json")
	if err := os.WriteFile(target, data, 0o600); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s\n", target)
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
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
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
			"type":                 "object",
			"additionalProperties": schemaForType(t.Elem()),
		}
	case reflect.Struct:
		properties := map[string]any{}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			if field.Tag.Get("schema") == "-" {
				continue
			}
			tag := field.Tag.Get("yaml")
			if tag == "-" {
				continue
			}
			name := strings.Split(tag, ",")[0]
			if name == "" {
				name = strings.ToLower(field.Name)
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
			"type":       "object",
			"properties": properties,
		}
	default:
		return map[string]any{"type": "string"}
	}
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
