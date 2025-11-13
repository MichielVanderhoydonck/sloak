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

	unitIndex := strings.LastIndexAny(s, "0123456789")
	if unitIndex == -1 {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	numberStr := s[:unitIndex+1]
	unitStr := s[unitIndex+1:]

	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in duration: %s", numberStr)
	}

	switch unitStr {
	case "d":
		unitStr = "h"
		number = number * 24
	case "w":
		unitStr = "h"
		number = number * 24 * 7
	case "y":
		unitStr = "h"
		number = number * 24 * 365
	}

	return time.ParseDuration(fmt.Sprintf("%f%s", number, unitStr))
}

func FormatDuration(d time.Duration) string {
	return d.Round(time.Second).String()
}
