package feasibility

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/feasibility"
)

type FeasibilityService interface {
	CalculateFeasibility(params domain.FeasibilityParams) (domain.FeasibilityResult, error)
}