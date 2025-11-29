package disruption

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/disruption"
)

type DisruptionService interface {
	CalculateCapacity(params domain.CalculationParams) (domain.Result, error)
}