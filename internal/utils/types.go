package utils

import "reflect"

// IsMap checks if a value is a map.
func IsMap(val any) bool {
	if val == nil {
		return false
	}
	v := reflect.ValueOf(val)
	return v.Kind() == reflect.Map
}

// IsList checks if a value is a list/array.
func IsList(val any) bool {
	if val == nil {
		return false
	}
	v := reflect.ValueOf(val)
	return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}

// ToMap converts an any to map[string]any.
func ToMap(val any) map[string]any {
	if m, ok := val.(map[string]any); ok {
		return m
	}
	return nil
}

// InterfaceToList converts an any to a slice.
func InterfaceToList(val any) []any {
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

// DeepEqual checks if two values are deeply equal.
func DeepEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}
