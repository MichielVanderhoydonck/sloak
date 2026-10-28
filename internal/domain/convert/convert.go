package convert

import (
	"errors"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
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
		if _, err := common.NewSLOTarget(p.Nines); err != nil {
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
	AvailabilityPercent      float64       `json:"availability_percent"`
	DailyDowntime            util.Duration `json:"daily_downtime"`
	DailyDowntimeSeconds     float64       `json:"daily_downtime_seconds"`
	WeeklyDowntime           util.Duration `json:"weekly_downtime"`
	WeeklyDowntimeSeconds    float64       `json:"weekly_downtime_seconds"`
	MonthlyDowntime          util.Duration `json:"monthly_downtime"`
	MonthlyDowntimeSeconds   float64       `json:"monthly_downtime_seconds"`
	QuarterlyDowntime        util.Duration `json:"quarterly_downtime"`
	QuarterlyDowntimeSeconds float64       `json:"quarterly_downtime_seconds"`
	YearlyDowntime           util.Duration `json:"yearly_downtime"`
	YearlyDowntimeSeconds    float64       `json:"yearly_downtime_seconds"`
}

type Service interface {
	Convert(params ConversionParams) (ConversionResult, error)
}
