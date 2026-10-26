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
	} else if p.Mode == ModeFromDowntime {
		if p.CustomWindow <= 0 {
			return errors.New("window must be greater than zero")
		}
		if p.Downtime < 0 {
			return errors.New("downtime cannot be negative")
		}
		if p.Downtime > p.CustomWindow {
			return errors.New("downtime cannot exceed window")
		}
	}
	return nil
}

type ConversionResult struct {
	AvailabilityPercent float64       `json:"availability_percent"`
	DailyDowntime       time.Duration `json:"daily_downtime"`
	WeeklyDowntime      time.Duration `json:"weekly_downtime"`
	MonthlyDowntime     time.Duration `json:"monthly_downtime"`
	QuarterlyDowntime   time.Duration `json:"quarterly_downtime"`
	YearlyDowntime      time.Duration `json:"yearly_downtime"`
}

type Service interface {
	Convert(params ConversionParams) (ConversionResult, error)
}
