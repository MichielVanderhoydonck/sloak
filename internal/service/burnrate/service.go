package burnrate

import (
	"errors"
	"math"
	"time"

	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/burnrate"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type BurnRateService struct{}

func NewBurnRateService() *BurnRateService {
	return &BurnRateService{}
}

func (s *BurnRateService) CalculateBurnRate(params burnrateDomain.CalculationParams) (burnrateDomain.BurnRateResult, error) {
	totalWindow := time.Duration(params.TotalWindow)
	errorConsumed := time.Duration(params.ErrorConsumed)
	timeElapsed := time.Duration(params.TimeElapsed)

	totalBudget := totalWindow.Seconds() * (1.0 - (params.TargetSLO.Value / 100.0))
	totalBudgetDur := time.Duration(totalBudget * float64(time.Second))

	if totalBudgetDur <= 0 {
		return burnrateDomain.BurnRateResult{}, errors.New("total error budget is zero or negative")
	}

	budgetRemaining := totalBudgetDur - errorConsumed

	var consumedPercent float64
	if totalBudgetDur.Seconds() > 0 {
		consumedPercent = (errorConsumed.Seconds() / totalBudgetDur.Seconds()) * 100.0
	} else {
		consumedPercent = 0.0
	}

	var burnRate float64
	if timeElapsed.Seconds() > 0 && totalWindow.Seconds() > 0 {
		timeElapsedPercent := (timeElapsed.Seconds() / totalWindow.Seconds()) * 100.0
		if timeElapsedPercent > 0 {
			burnRate = consumedPercent / timeElapsedPercent
		} else {
			burnRate = 0.0 // Should not happen if timeElapsed.Seconds() > 0
		}
	} else {
		if errorConsumed > 0 {
			burnRate = util.Inf
		} else {
			burnRate = 0.0
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
		BurnRate:                util.RoundValue(burnRate),
		TimeToExhaustion:        util.Duration(tte),
		TimeToExhaustionSeconds: math.Round(tte.Seconds()),
		IsInfinite:              errorConsumed == 0,
	}, nil
}
