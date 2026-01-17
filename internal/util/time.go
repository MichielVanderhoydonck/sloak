package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseTimeWindow(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// First, try parsing with the standard library. This handles formats like "1h30m", "10s", etc.
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}
	originalErr := err // Keep original error for better messages

	// If standard parsing fails, it might be due to our custom units (d, w, y).
	// This custom logic expects a number followed by a single character unit, e.g., "30d", "1.5w".
	unitIndex := strings.LastIndexAny(s, "dwy")
	if unitIndex == -1 || unitIndex != len(s)-1 {
		// No custom unit found, or it's not at the end of the string.
		// Return the original, more descriptive error from time.ParseDuration.
		return 0, originalErr
	}

	numberStr := s[:unitIndex]
	unitStr := s[unitIndex:]

	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		// The part before the unit is not a valid float, e.g., "1h30d".
		return 0, fmt.Errorf("invalid number %q for custom duration %q", numberStr, s)
	}

	var unitMultiplier float64
	switch unitStr {
	case "d":
		unitMultiplier = float64(24 * time.Hour)
	case "w":
		unitMultiplier = float64(7 * 24 * time.Hour)
	case "y":
		unitMultiplier = float64(365 * 24 * time.Hour)
	}

	// Calculate duration in nanoseconds.
	nanos := number * unitMultiplier
	return time.Duration(nanos), nil
}

func FormatDuration(d time.Duration) string {
	return d.Round(time.Second).String()
}
