package util

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Duration is a wrapper around time.Duration that provides better JSON and String formatting.
type Duration time.Duration

var Inf = math.Inf(1)

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d Duration) String() string {
	return FormatDuration(time.Duration(d))
}

// ParseTimeWindow parses a duration string, including custom units like 'd', 'w', 'y'.
// Negative durations are rejected as invalid.
func ParseTimeWindow(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// First, try parsing with the standard library.
	d, err := time.ParseDuration(s)
	if err == nil {
		if d < 0 {
			return 0, fmt.Errorf("duration cannot be negative: %q", s)
		}
		return d, nil
	}
	originalErr := err

	// If standard parsing fails, check for custom units (d, w, y).
	unitIndex := strings.LastIndexAny(s, "dwy")
	if unitIndex == -1 || unitIndex != len(s)-1 {
		return 0, originalErr
	}

	numberStr := s[:unitIndex]
	unitStr := s[unitIndex:]

	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q for custom duration %q", numberStr, s)
	}
	if number < 0 {
		return 0, fmt.Errorf("duration cannot be negative: %q", s)
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

	nanos := number * unitMultiplier
	if nanos > float64(math.MaxInt64) {
		return 0, fmt.Errorf("duration %q overflows maximum duration limit", s)
	}
	return time.Duration(nanos), nil
}

// FormatDuration formats a time.Duration into a human-readable string.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d == 0 {
		return "0s"
	}
	if d < 0 {
		return "-" + FormatDuration(-d)
	}

	hours := d.Hours()
	if hours >= 24*365 { // 1 year or more
		return fmt.Sprintf("%.1fy", hours/(24*365))
	}
	if hours >= 24*2 { // 2 days or more
		return fmt.Sprintf("%.1fd", hours/24)
	}

	return d.String()
}

// RoundPercentage rounds a percentage to 4 decimal places.
func RoundPercentage(val float64) float64 {
	return math.Round(val*10000) / 10000
}

// RoundValue rounds a general float to 2 decimal places.
func RoundValue(val float64) float64 {
	return math.Round(val*100) / 100
}
