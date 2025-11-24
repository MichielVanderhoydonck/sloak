package translator

import (
	"errors"
	"time"
)

// CalculationMode determines which direction we are translating.
type CalculationMode string

const (
	ModeFromNines    CalculationMode = "from_nines"
	ModeFromDowntime CalculationMode = "from_downtime"
)

// TranslationParams holds the user input.
type TranslationParams struct {
	Mode CalculationMode
	
	// Nines is used if Mode == ModeFromNines
	Nines float64
	
	// Downtime & Window are used if Mode == ModeFromDowntime
	Downtime      time.Duration
	CustomWindow  time.Duration
}

// TranslationResult holds the standardized output.
type TranslationResult struct {
	AvailabilityPercent float64
	
	// Standard Windows
	DailyDowntime   time.Duration // 24h
	WeeklyDowntime  time.Duration // 7d
	MonthlyDowntime time.Duration // 30d
	QuarterlyDowntime time.Duration // 90d
	YearlyDowntime  time.Duration // 365d
}

func (p TranslationParams) Validate() error {
	switch p.Mode {
	case ModeFromNines:
		if p.Nines < 0 || p.Nines > 100 {
			return errors.New("percentage must be between 0 and 100")
		}
	case ModeFromDowntime:
		if p.CustomWindow <= 0 {
			return errors.New("custom window must be greater than zero")
		}
		if p.Downtime < 0 {
			return errors.New("downtime cannot be negative")
		}
		if p.Downtime > p.CustomWindow {
			return errors.New("downtime cannot exceed the window duration")
		}
	default:
		return errors.New("unknown calculation mode")
	}
	return nil
}