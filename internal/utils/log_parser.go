package utils

import (
	"encoding/json"
	"strings"
	"time"
)

// FormatLogOutput parses terraform log output into a readable summary.
func FormatLogOutput(output string) string {
	if strings.TrimSpace(output) == "" {
		return ""
	}
	lines := make([]string, 0, 16)
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var payload map[string]any
		if json.Unmarshal([]byte(trimmed), &payload) == nil {
			message := ""
			if val, ok := payload["@message"].(string); ok {
				message = val
			} else if val, ok := payload["message"].(string); ok {
				message = val
			}
			timestamp := ""
			if val, ok := payload["@timestamp"].(string); ok {
				timestamp = val
			} else if val, ok := payload["timestamp"].(string); ok {
				timestamp = val
			}
			if message != "" && timestamp != "" {
				lines = append(lines, "["+formatLogTimestamp(timestamp)+"] "+message)
				continue
			}
			if message != "" {
				lines = append(lines, message)
				continue
			}
		}
		lines = append(lines, trimmed)
	}
	return strings.Join(lines, "\n")
}

func formatLogTimestamp(value string) string {
	ts, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		ts, err = time.Parse(time.RFC3339, value)
	}
	if err != nil {
		return value
	}
	return ts.Format("2006-01-02 15:04:05 -07:00")
}
