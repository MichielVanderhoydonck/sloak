package feasibility

import (
	"errors"
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/feasibility"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type FeasibilityService struct{}

func NewFeasibilityService() *FeasibilityService {
	return &FeasibilityService{}
}

func (s *FeasibilityService) CalculateFeasibility(params domain.FeasibilityParams) (domain.FeasibilityResult, error) {
	mttr := time.Duration(params.MTTR)
	errorBudgetPercent := 1.0 - (params.TargetSLO.Value / 100.0)

	if errorBudgetPercent <= 0 {
		return domain.FeasibilityResult{}, errors.New("SLO target 100% leaves no room for incidents; MTBF is infinite")
	}

	mtbfNanos := mttr.Seconds() / errorBudgetPercent

	incidentsPerYear := (365 * 24 * time.Hour).Seconds() / (mtbfNanos + mttr.Seconds())

	requiredMTBF := time.Duration(math.Round(mtbfNanos * float64(time.Second)))

	return domain.FeasibilityResult{
		TargetSLO:           params.TargetSLO,
		TargetSLORatio:      math.Round((params.TargetSLO.Value/100.0)*1000000) / 1000000,
		MTTR:                params.MTTR,
		MTTRSeconds:         math.Round(mttr.Seconds()),
		IncidentsPerYear:    util.RoundValue(incidentsPerYear),
		IncidentsPerQuarter: util.RoundValue(incidentsPerYear / 4.0),
		IncidentsPerMonth:   util.RoundValue(incidentsPerYear / 12.0),
		RequiredMTBF:        util.Duration(requiredMTBF),
		RequiredMTBFSeconds: math.Round(requiredMTBF.Seconds()),
	}, nil
}
