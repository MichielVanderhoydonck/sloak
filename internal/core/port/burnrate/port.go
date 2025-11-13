package burnrate

import (
	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/burnrate"
)

type BurnRateService interface {
	CalculateBurnRate(params burnrateDomain.CalculationParams) (burnrateDomain.BurnRateResult, error)
}