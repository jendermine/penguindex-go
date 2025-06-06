// File: penguindex-go/internal/utils/utils.go
package utils

import "fmt"

// HumanReadableSize converts bytes to a human-readable string (e.g., KiB, MiB).
func HumanReadableSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
