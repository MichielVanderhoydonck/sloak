package feasibility

import (
	"errors"
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/feasibility"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/feasibility"
)

var _ port.FeasibilityService = (*FeasibilityServiceImpl)(nil)

type FeasibilityServiceImpl struct{}

func NewFeasibilityService() port.FeasibilityService {
	return &FeasibilityServiceImpl{}
}

func (s *FeasibilityServiceImpl) CalculateFeasibility(params domain.FeasibilityParams) (domain.FeasibilityResult, error) {
	if params.MTTR <= 0 {
		return domain.FeasibilityResult{}, errors.New("MTTR must be greater than zero")
	}

	unavailability := 1.0 - (params.TargetSLO.Value / 100.0)
	yearDuration := 8760 * time.Hour
	yearlyBudget := float64(yearDuration) * unavailability
	
	incidentsPerYear := yearlyBudget / float64(params.MTTR)

	availabilityRatio := params.TargetSLO.Value / 100.0
	mtbfNanos := float64(params.MTTR) * (availabilityRatio / unavailability)

	return domain.FeasibilityResult{
		TargetSLO:           params.TargetSLO,
		MTTR:                params.MTTR,
		IncidentsPerYear:    incidentsPerYear,
		IncidentsPerQuarter: incidentsPerYear / 4.0,
		IncidentsPerMonth:   incidentsPerYear / 12.0,
		RequiredMTBF:        time.Duration(math.Round(mtbfNanos)),
	}, nil
}