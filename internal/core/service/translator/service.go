package translator

import (
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/translator"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/translator"
)

var _ port.TranslatorService = (*TranslatorServiceImpl)(nil)

type TranslatorServiceImpl struct{}

func NewTranslatorService() port.TranslatorService {
	return &TranslatorServiceImpl{}
}

func (s *TranslatorServiceImpl) Translate(params domain.TranslationParams) (domain.TranslationResult, error) {
	if err := params.Validate(); err != nil {
		return domain.TranslationResult{}, err
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
		nanos := float64(window) * errorRatio
		return time.Duration(math.Round(nanos))
	}

	return domain.TranslationResult{
		AvailabilityPercent: availabilityPercent,
		DailyDowntime:       calcDuration(24 * time.Hour),
		WeeklyDowntime:      calcDuration(7 * 24 * time.Hour),
		MonthlyDowntime:     calcDuration(30 * 24 * time.Hour),
		QuarterlyDowntime:   calcDuration(90 * 24 * time.Hour),
		YearlyDowntime:      calcDuration(365 * 24 * time.Hour),
	}, nil
}