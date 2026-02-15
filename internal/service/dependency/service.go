package dependency

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/dependency"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type AvailabilityService struct{}

func NewAvailabilityService() *AvailabilityService {
	return &AvailabilityService{}
}

func (s *AvailabilityService) CalculateCompositeAvailability(params domain.CalculationParams) (domain.Result, error) {
	if err := params.Validate(); err != nil {
		return domain.Result{}, err
	}

	var total float64

	if params.Type == domain.Serial {
		decimalTotal := 1.0
		for _, c := range params.Components {
			decimalTotal *= (c / 100.0)
		}
		total = decimalTotal * 100.0
	} else {
		probabilityOfTotalFailure := 1.0
		for _, c := range params.Components {
			probFail := 1.0 - (c / 100.0)
			probabilityOfTotalFailure *= probFail
		}
		total = (1.0 - probabilityOfTotalFailure) * 100.0
	}

	return domain.Result{
		TotalAvailability: util.RoundPercentage(total),
		CalculationType:   params.Type,
		ComponentCount:    len(params.Components),
	}, nil
}
