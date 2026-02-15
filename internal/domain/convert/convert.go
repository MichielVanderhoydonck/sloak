package convert

import (
	"errors"
	"time"
)

type ConversionMode int

const (
	ModeFromNines ConversionMode = iota
	ModeFromDowntime
)

type ConversionParams struct {
	Mode         ConversionMode
	Nines        float64
	Downtime     time.Duration
	CustomWindow time.Duration
}

func (p *ConversionParams) Validate() error {
	if p.Mode == ModeFromNines {
		if p.Nines <= 0 || p.Nines >= 100 {
			return errors.New("availability must be between 0 and 100")
		}
	}
	return nil
}

type ConversionResult struct {
	AvailabilityPercent float64
	DailyDowntime       time.Duration
	WeeklyDowntime      time.Duration
	MonthlyDowntime     time.Duration
	QuarterlyDowntime   time.Duration
	YearlyDowntime      time.Duration
}
