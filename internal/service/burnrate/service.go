package burnrate

import (
	"errors"
	"math"
	"time"

	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/burnrate"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type BurnRateService struct{}

var _ burnrateDomain.Service = (*BurnRateService)(nil)

func NewBurnRateService() *BurnRateService {
	return &BurnRateService{}
}

func (s *BurnRateService) CalculateBurnRate(params burnrateDomain.CalculationParams) (burnrateDomain.BurnRateResult, error) {
	totalWindow := time.Duration(params.TotalWindow)
	errorConsumed := time.Duration(params.ErrorConsumed)
	timeElapsed := time.Duration(params.TimeElapsed)

	if totalWindow <= 0 {
		return burnrateDomain.BurnRateResult{}, errors.New("total window must be greater than zero")
	}
	if timeElapsed < 0 {
		return burnrateDomain.BurnRateResult{}, errors.New("time elapsed cannot be negative")
	}
	if errorConsumed < 0 {
		return burnrateDomain.BurnRateResult{}, errors.New("error consumed cannot be negative")
	}

	totalBudget := totalWindow.Seconds() * (1.0 - (params.TargetSLO.Value / 100.0))
	totalBudgetDur := time.Duration(totalBudget * float64(time.Second))

	if totalBudgetDur <= 0 {
		return burnrateDomain.BurnRateResult{}, errors.New("total error budget is zero or negative")
	}

	budgetRemaining := totalBudgetDur - errorConsumed
	consumedPercent := (errorConsumed.Seconds() / totalBudgetDur.Seconds()) * 100.0

	var burnRate *float64
	var burnRateUnbounded bool
	if timeElapsed.Seconds() > 0 {
		timeElapsedPercent := (timeElapsed.Seconds() / totalWindow.Seconds()) * 100.0
		val := util.RoundValue(consumedPercent / timeElapsedPercent)
		burnRate = &val
	} else {
		if errorConsumed > 0 {
			burnRate = nil
			burnRateUnbounded = true
		} else {
			zero := 0.0
			burnRate = &zero
		}
	}

	var tte time.Duration
	if errorConsumed > 0 && timeElapsed.Seconds() > 0 {
		consumptionRate := errorConsumed.Seconds() / timeElapsed.Seconds()
		if consumptionRate > 0 {
			tte = time.Duration(budgetRemaining.Seconds()/consumptionRate) * time.Second
		}
	}

	if tte < 0 {
		tte = 0
	}

	return burnrateDomain.BurnRateResult{
		TotalErrorBudget:        util.Duration(totalBudgetDur),
		TotalErrorBudgetSeconds: math.Round(totalBudgetDur.Seconds()),
		BudgetRemaining:         util.Duration(budgetRemaining),
		BudgetRemainingSeconds:  math.Round(budgetRemaining.Seconds()),
		BudgetConsumed:          util.RoundPercentage(consumedPercent),
		BurnRate:                burnRate,
		BurnRateUnbounded:       burnRateUnbounded,
		TimeToExhaustion:        util.Duration(tte),
		TimeToExhaustionSeconds: math.Round(tte.Seconds()),
		IsInfinite:              errorConsumed == 0,
	}, nil
}
