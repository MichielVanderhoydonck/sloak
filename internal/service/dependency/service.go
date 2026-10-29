package dependency

import (
	"math"
	"strings"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/dependency"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type AvailabilityService struct{}

var _ domain.Service = (*AvailabilityService)(nil)

func NewAvailabilityService() domain.Service {
	return &AvailabilityService{}
}

func (s *AvailabilityService) CalculateCompositeAvailability(params domain.CalculationParams) (domain.Result, error) {
	if err := params.Validate(); err != nil {
		return domain.Result{}, err
	}

	stages, componentCount := resolveStages(params)

	var total float64
	if params.Type == domain.Serial {
		decimalTotal := 1.0
		for _, st := range stages {
			decimalTotal *= (st.Availability / 100.0)
		}
		total = decimalTotal * 100.0
	} else {
		probabilityOfTotalFailure := 1.0
		for _, st := range stages {
			probFail := 1.0 - (st.Availability / 100.0)
			probabilityOfTotalFailure *= probFail
		}
		total = (1.0 - probabilityOfTotalFailure) * 100.0
	}

	roundedTotal := util.RoundPercentage(total)
	res := domain.Result{
		TotalAvailability: roundedTotal,
		CalculationType:   params.Type,
		ComponentCount:    componentCount,
	}

	if params.Type == domain.Serial && len(stages) > 0 {
		weakest := stages[0]
		for _, st := range stages[1:] {
			if st.Availability < weakest.Availability {
				weakest = st
			}
		}
		res.Bottleneck = &domain.Component{
			Name:         weakest.Name,
			Availability: util.RoundPercentage(weakest.Availability),
		}
	}

	if params.TimeWindow > 0 {
		windowDur := time.Duration(params.TimeWindow)
		unavailabilityRatio := 1.0 - (total / 100.0)
		if unavailabilityRatio < 0 {
			unavailabilityRatio = 0
		}
		downtime := util.Duration(time.Duration(math.Round(float64(windowDur) * unavailabilityRatio)))
		winCopy := params.TimeWindow
		res.AllowedDowntime = &downtime
		res.TimeWindow = &winCopy
	}

	if params.TargetSLO != nil {
		targetCopy := *params.TargetSLO
		meets := roundedTotal >= targetCopy.Value
		res.TargetSLO = &targetCopy
		res.MeetsTarget = &meets
		if meets {
			res.Status = domain.StatusMeetsTarget
		} else {
			res.Status = domain.StatusBelowTarget
		}
	}

	return res, nil
}

func resolveStages(params domain.CalculationParams) ([]domain.Stage, int) {
	if len(params.Stages) > 0 {
		resolved := make([]domain.Stage, len(params.Stages))
		count := 0
		for i, st := range params.Stages {
			if len(st.RedundantMembers) > 0 {
				probFail := 1.0
				var names []string
				for _, m := range st.RedundantMembers {
					probFail *= (1.0 - (m.Availability / 100.0))
					if m.Name != "" {
						names = append(names, m.Name)
					}
					count++
				}
				name := st.Name
				if name == "" && len(names) > 0 {
					name = strings.Join(names, "|")
				}
				resolved[i] = domain.Stage{
					Name:             name,
					Availability:     (1.0 - probFail) * 100.0,
					RedundantMembers: st.RedundantMembers,
				}
			} else {
				resolved[i] = st
				count++
			}
		}
		return resolved, count
	}

	resolved := make([]domain.Stage, len(params.Components))
	for i, c := range params.Components {
		resolved[i] = domain.Stage{Availability: c}
	}
	return resolved, len(params.Components)
}
