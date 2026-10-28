package disruption

import (
	"math"
	"time"

	disruptionDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/disruption"
	"github.com/MichielVanderhoydonck/sloak/internal/util"
)

type DisruptionService struct{}

var _ disruptionDomain.Service = (*DisruptionService)(nil)

func NewDisruptionService() disruptionDomain.Service {
	return &DisruptionService{}
}

func (s *DisruptionService) CalculateCapacity(params disruptionDomain.CalculationParams) (disruptionDomain.Result, error) {
	if err := params.Validate(); err != nil {
		return disruptionDomain.Result{}, err
	}

	totalWindow := time.Duration(params.TotalWindow)
	costPerEvent := time.Duration(params.CostPerEvent)

	totalBudget := params.TargetSLO.AllowedDowntime(totalWindow)

	maxDisruptions := int64(totalBudget / costPerEvent)
	dailyDisruptions := util.RoundValue(float64(maxDisruptions) / (totalWindow.Hours() / 24.0))

	return disruptionDomain.Result{
		TotalErrorBudget:        util.Duration(totalBudget),
		TotalErrorBudgetSeconds: math.Round(totalBudget.Seconds()),
		MaxDisruptions:          maxDisruptions,
		DailyDisruptions:        dailyDisruptions,
		Status:                  disruptionDomain.ClassifyDisruption(maxDisruptions, dailyDisruptions),
	}, nil
}
