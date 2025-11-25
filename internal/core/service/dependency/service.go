package dependency

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/dependency"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/dependency"
)

var _ port.AvailabilityService = (*AvailabilityServiceImpl)(nil)

type AvailabilityServiceImpl struct{}

func NewAvailabilityService() port.AvailabilityService {
	return &AvailabilityServiceImpl{}
}

func (s *AvailabilityServiceImpl) CalculateCompositeAvailability(params domain.CalculationParams) (domain.Result, error) {
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
		TotalAvailability: total,
		CalculationType:   params.Type,
		ComponentCount:    len(params.Components),
	}, nil
}
