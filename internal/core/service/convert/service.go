package convert

import (
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/convert"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/convert"
)

var _ port.ConvertService = (*ConvertServiceImpl)(nil)

type ConvertServiceImpl struct{}

func NewConvertService() port.ConvertService {
	return &ConvertServiceImpl{}
}

func (s *ConvertServiceImpl) Convert(params domain.ConversionParams) (domain.ConversionResult, error) {
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

	calcDuration := func(window time.Duration) time.Duration {
		seconds := window.Seconds() * errorRatio
		return time.Duration(math.Round(seconds)) * time.Second
	}

	return domain.ConversionResult{
		AvailabilityPercent: availabilityPercent,
		DailyDowntime:       calcDuration(24 * time.Hour),
		WeeklyDowntime:      calcDuration(7 * 24 * time.Hour),
		MonthlyDowntime:     calcDuration(30 * 24 * time.Hour),
		QuarterlyDowntime:   calcDuration(90 * 24 * time.Hour),
		YearlyDowntime:      calcDuration(365 * 24 * time.Hour),
	}, nil
}
