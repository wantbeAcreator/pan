package gui

import "fmt"

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func humanCount(n int, label string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", label)
	}
	return fmt.Sprintf("%d %ss", n, label)
}
