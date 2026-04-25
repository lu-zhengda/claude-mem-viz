package ui

import (
	"fmt"
	"time"
)

// ageString formats a time as a compact relative age suitable for a chip.
// today / 3d / 2w / 4mo / 2y
func ageString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t).Hours() / 24
	switch {
	case d < 1:
		return "today"
	case d < 7:
		return fmt.Sprintf("%dd", int(d))
	case d < 30:
		return fmt.Sprintf("%dw", int(d/7))
	case d < 365:
		return fmt.Sprintf("%dmo", int(d/30))
	default:
		return fmt.Sprintf("%dy", int(d/365))
	}
}
