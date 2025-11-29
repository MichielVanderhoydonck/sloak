package disruption

import (
	"errors"
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/disruption"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/disruption"
)

var _ port.DisruptionService = (*DisruptionServiceImpl)(nil)

type DisruptionServiceImpl struct{}

func NewDisruptionService() port.DisruptionService {
	return &DisruptionServiceImpl{}
}

func (s *DisruptionServiceImpl) CalculateCapacity(params domain.CalculationParams) (domain.Result, error) {
	if params.TotalWindow <= 0 {
		return domain.Result{}, errors.New("window must be greater than zero")
	}
	if params.CostPerEvent <= 0 {
		return domain.Result{}, errors.New("cost per disruption must be greater than zero")
	}

	errorBudgetPercent := 1.0 - (params.TargetSLO.Value / 100.0)
	totalBudgetNanos := float64(params.TotalWindow) * errorBudgetPercent
	totalBudget := time.Duration(math.Round(totalBudgetNanos))

	maxEvents := int64(totalBudget) / int64(params.CostPerEvent)

	daysInWindow := params.TotalWindow.Hours() / 24.0
	dailyAverage := float64(maxEvents) / daysInWindow

	return domain.Result{
		TotalErrorBudget: totalBudget,
		MaxDisruptions:   maxEvents,
		DailyDisruptions: dailyAverage,
	}, nil
}