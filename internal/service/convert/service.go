package convert

import (
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/convert"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type ConvertService struct{}

var _ domain.Service = (*ConvertService)(nil)

func NewConvertService() *ConvertService {
	return &ConvertService{}
}

func (s *ConvertService) Convert(params domain.ConversionParams) (domain.ConversionResult, error) {
	if err := params.Validate(); err != nil {
		return domain.ConversionResult{}, err
	}

	var availabilityPercent float64

	if params.Mode == domain.ModeFromNines {
		availabilityPercent = params.Nines
	} else {
		ratio := float64(params.Downtime) / float64(params.CustomWindow)
		availabilityPercent = (1.0 - ratio) * 100.0
	}

	errorRatio := 1.0 - (availabilityPercent / 100.0)

	calcDuration := func(window time.Duration) (util.Duration, float64) {
		seconds := window.Seconds() * errorRatio
		dur := time.Duration(math.Round(seconds)) * time.Second
		return util.Duration(dur), dur.Seconds()
	}

	dailyDur, dailySec := calcDuration(24 * time.Hour)
	weeklyDur, weeklySec := calcDuration(7 * 24 * time.Hour)
	monthlyDur, monthlySec := calcDuration(30 * 24 * time.Hour)
	quarterlyDur, quarterlySec := calcDuration(90 * 24 * time.Hour)
	yearlyDur, yearlySec := calcDuration(365 * 24 * time.Hour)

	return domain.ConversionResult{
		AvailabilityPercent:      availabilityPercent,
		DailyDowntime:            dailyDur,
		DailyDowntimeSeconds:     dailySec,
		WeeklyDowntime:           weeklyDur,
		WeeklyDowntimeSeconds:    weeklySec,
		MonthlyDowntime:          monthlyDur,
		MonthlyDowntimeSeconds:   monthlySec,
		QuarterlyDowntime:        quarterlyDur,
		QuarterlyDowntimeSeconds: quarterlySec,
		YearlyDowntime:           yearlyDur,
		YearlyDowntimeSeconds:    yearlySec,
	}, nil
}
