package disruption

import (
	"errors"
	"math"
	"time"

	disruptionDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/disruption"
	disruptionPort "github.com/MichielVanderhoydonck/sloak/internal/core/port/disruption"
	"github.com/MichielVanderhoydonck/sloak/internal/util"
)

var _ disruptionPort.DisruptionService = (*DisruptionServiceImpl)(nil)

type DisruptionServiceImpl struct{}

func NewDisruptionService() disruptionPort.DisruptionService {
	return &DisruptionServiceImpl{}
}

func (s *DisruptionServiceImpl) CalculateCapacity(params disruptionDomain.CalculationParams) (disruptionDomain.Result, error) {
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
