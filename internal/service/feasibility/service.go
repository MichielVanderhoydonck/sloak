package feasibility

import (
	"errors"
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/feasibility"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type FeasibilityService struct{}

var _ domain.Service = (*FeasibilityService)(nil)

func NewFeasibilityService() *FeasibilityService {
	return &FeasibilityService{}
}

func (s *FeasibilityService) CalculateFeasibility(params domain.FeasibilityParams) (domain.FeasibilityResult, error) {
	mttr := time.Duration(params.MTTR)
	if mttr <= 0 {
		return domain.FeasibilityResult{}, errors.New("MTTR must be greater than zero")
	}

	if params.TargetSLO.Value <= 0 || params.TargetSLO.Value >= 100 {
		return domain.FeasibilityResult{}, errors.New("SLO target must be strictly between 0 and 100")
	}

	sloRatio := params.TargetSLO.Value / 100.0
	unavailabilityRatio := 1.0 - sloRatio

	yearSeconds := (365 * 24 * time.Hour).Seconds()
	incidentsPerYear := yearSeconds * unavailabilityRatio / mttr.Seconds()

	mtbfSeconds := mttr.Seconds() * (sloRatio / unavailabilityRatio)
	requiredMTBF := time.Duration(math.Round(mtbfSeconds * float64(time.Second)))

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
