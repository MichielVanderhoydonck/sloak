package disruption

import (
	"errors"
	"math"
	"time"

	disruptionDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/disruption"
	"github.com/MichielVanderhoydonck/sloak/internal/util"
)

type DisruptionService struct{}

func NewDisruptionService() *DisruptionService {
	return &DisruptionService{}
}

func (s *DisruptionService) CalculateCapacity(params disruptionDomain.CalculationParams) (disruptionDomain.Result, error) {
	if params.TotalWindow <= 0 {
		return disruptionDomain.Result{}, errors.New("window must be greater than zero")
	}
	if params.CostPerEvent <= 0 {
		return disruptionDomain.Result{}, errors.New("cost per disruption must be greater than zero")
	}

	totalWindow := time.Duration(params.TotalWindow)
	costPerEvent := time.Duration(params.CostPerEvent)

	errorBudgetPercent := 1.0 - (params.TargetSLO.Value / 100.0)
	totalBudgetNanos := float64(totalWindow) * errorBudgetPercent
	totalBudget := time.Duration(math.Round(totalBudgetNanos))

	var maxDisruptions int64
	var dailyDisruptions float64

	if costPerEvent > 0 {
		maxDisruptions = int64(totalBudget / costPerEvent)
		dailyDisruptions = float64(maxDisruptions) / (totalWindow.Hours() / 24.0)
	}

	return disruptionDomain.Result{
		TotalErrorBudget:        util.Duration(totalBudget),
		TotalErrorBudgetSeconds: math.Round(totalBudget.Seconds()),
		MaxDisruptions:          maxDisruptions,
		DailyDisruptions:        util.RoundValue(dailyDisruptions),
	}, nil
}
