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
func ParseTimeWindow(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// First, try parsing with the standard library.
	d, err := time.ParseDuration(s)
	if err == nil {
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
	if hours >= 24*365*2 { // 2 years or more
		return fmt.Sprintf("%.1fy", hours/(24*365))
	}
	if hours >= 24*60 { // 60 days or more
		return fmt.Sprintf("%.1fw", hours/(24*7))
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
