package benchmark

import "strings"

// isNotFoundError checks if the error is a 404 not found error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "status 404") ||
		strings.Contains(errorStr, "was not found") ||
		strings.Contains(errorStr, "NOT_FOUND")
}
