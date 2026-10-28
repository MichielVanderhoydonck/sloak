package feasibility

import (
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/feasibility"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type FeasibilityService struct{}

var _ domain.Service = (*FeasibilityService)(nil)

func NewFeasibilityService() domain.Service {
	return &FeasibilityService{}
}

func (s *FeasibilityService) CalculateFeasibility(params domain.FeasibilityParams) (domain.FeasibilityResult, error) {
	if err := params.Validate(); err != nil {
		return domain.FeasibilityResult{}, err
	}

	mttr := time.Duration(params.MTTR)
	sloRatio := params.TargetSLO.Ratio()
	unavailabilityRatio := 1.0 - sloRatio

	yearSeconds := (365 * 24 * time.Hour).Seconds()
	incidentsPerYear := yearSeconds * unavailabilityRatio / mttr.Seconds()
	incidentsPerQuarter := util.RoundValue(incidentsPerYear / 4.0)

	mtbfSeconds := mttr.Seconds() * (sloRatio / unavailabilityRatio)
	requiredMTBF := time.Duration(math.Round(mtbfSeconds * float64(time.Second)))

	return domain.FeasibilityResult{
		TargetSLO:           params.TargetSLO,
		TargetSLORatio:      params.TargetSLO.RoundedRatio(),
		MTTR:                params.MTTR,
		MTTRSeconds:         math.Round(mttr.Seconds()),
		IncidentsPerYear:    util.RoundValue(incidentsPerYear),
		IncidentsPerQuarter: incidentsPerQuarter,
		IncidentsPerMonth:   util.RoundValue(incidentsPerYear / 12.0),
		RequiredMTBF:        util.Duration(requiredMTBF),
		RequiredMTBFSeconds: math.Round(requiredMTBF.Seconds()),
		Status:              domain.ClassifyFeasibility(incidentsPerQuarter),
	}, nil
}
