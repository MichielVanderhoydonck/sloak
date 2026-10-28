package common

import (
	"errors"
	"fmt"
	"math"
	"time"
)

type SLOTarget struct {
	Value float64 `json:"value"`
}

var ErrInvalidSLOTarget = errors.New("SLO target must be strictly between 0.0 and 100.0")

func NewSLOTarget(value float64) (SLOTarget, error) {
	if value <= 0.0 || value >= 100.0 {
		return SLOTarget{}, ErrInvalidSLOTarget
	}
	return SLOTarget{Value: value}, nil
}

func (s SLOTarget) Validate() error {
	if s.Value <= 0.0 || s.Value >= 100.0 {
		return ErrInvalidSLOTarget
	}
	return nil
}

// Ratio returns the target availability as a fraction in (0.0, 1.0).
func (s SLOTarget) Ratio() float64 {
	return s.Value / 100.0
}

// RoundedRatio returns the target availability ratio rounded to 6 decimal places.
func (s SLOTarget) RoundedRatio() float64 {
	return math.Round(s.Ratio()*1000000) / 1000000
}

// ErrorBudgetRatio returns the allowed error budget fraction in (0.0, 1.0),
// rounded to 10 decimal places to avoid floating-point representation noise.
func (s SLOTarget) ErrorBudgetRatio() float64 {
	budget := 1.0 - (s.Value / 100.0)
	return math.Round(budget*1e10) / 1e10
}

// AllowedDowntime returns the total allowed error duration for the given time window.
func (s SLOTarget) AllowedDowntime(window time.Duration) time.Duration {
	allowedNanos := float64(window) * (1.0 - (s.Value / 100.0))
	return time.Duration(math.Round(allowedNanos))
}

func (s SLOTarget) String() string {
	return fmt.Sprintf("%.3f%%", s.Value)
}
