package dependency

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/dependency"
)

type AvailabilityService interface {
	CalculateCompositeAvailability(params domain.CalculationParams) (domain.Result, error)
}