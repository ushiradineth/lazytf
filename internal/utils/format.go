package utils

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
