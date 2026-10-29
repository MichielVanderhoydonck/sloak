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

func NewBurnRateService() burnrateDomain.Service {
	return &BurnRateService{}
}

func (s *BurnRateService) CalculateBurnRate(params burnrateDomain.CalculationParams) (burnrateDomain.BurnRateResult, error) {
	if err := params.Validate(); err != nil {
		return burnrateDomain.BurnRateResult{}, err
	}

	totalWindow := time.Duration(params.TotalWindow)
	errorConsumed := time.Duration(params.ErrorConsumed)
	errorConsumedSec := errorConsumed.Seconds()
	if params.FailedRequests != nil && params.RPS > 0 {
		errorConsumedSec = float64(*params.FailedRequests) / params.RPS
		errorConsumed = time.Duration(math.Round(errorConsumedSec * float64(time.Second)))
	}
	timeElapsed := time.Duration(params.TimeElapsed)

	totalBudgetDur := params.TargetSLO.AllowedDowntime(totalWindow)
	if totalBudgetDur <= 0 {
		return burnrateDomain.BurnRateResult{}, errors.New("total error budget is zero or negative")
	}

	budgetRemaining := totalBudgetDur - errorConsumed
	consumedPercent := (errorConsumedSec / totalBudgetDur.Seconds()) * 100.0
	timeElapsedPercent := (timeElapsed.Seconds() / totalWindow.Seconds()) * 100.0

	var burnRate *float64
	var burnRateUnbounded bool
	if timeElapsed > 0 {
		// Calculate burn rate directly from unrounded ratios: (consumed/budget) / (elapsed/window)
		rawBurnRate := (errorConsumedSec / totalBudgetDur.Seconds()) / (timeElapsed.Seconds() / totalWindow.Seconds())
		val := util.RoundValue(rawBurnRate)
		burnRate = &val
	} else {
		if errorConsumedSec > 0 {
			burnRate = nil
			burnRateUnbounded = true
		} else {
			zero := 0.0
			burnRate = &zero
		}
	}

	var tte time.Duration
	if errorConsumedSec > 0 && timeElapsed.Seconds() > 0 {
		consumptionRate := errorConsumedSec / timeElapsed.Seconds()
		if consumptionRate > 0 {
			remainingSec := totalBudgetDur.Seconds() - errorConsumedSec
			tte = time.Duration(math.Round(remainingSec/consumptionRate)) * time.Second
		}
	}

	if tte < 0 {
		tte = 0
	}

	noBudgetConsumed := errorConsumedSec == 0
	status := burnrateDomain.ClassifyBurnRate(burnRate, burnRateUnbounded, noBudgetConsumed)

	res := burnrateDomain.BurnRateResult{
		TotalErrorBudget:        util.Duration(totalBudgetDur),
		TotalErrorBudgetSeconds: math.Round(totalBudgetDur.Seconds()),
		BudgetRemaining:         util.Duration(budgetRemaining),
		BudgetRemainingSeconds:  math.Round(budgetRemaining.Seconds()),
		BudgetConsumed:          util.RoundPercentage(consumedPercent),
		TimeElapsedPercent:      util.RoundPercentage(timeElapsedPercent),
		BurnRate:                burnRate,
		BurnRateUnbounded:       burnRateUnbounded,
		TimeToExhaustion:        util.Duration(tte),
		TimeToExhaustionSeconds: math.Round(tte.Seconds()),
		NoBudgetConsumed:        noBudgetConsumed,
		Status:                  status,
	}

	if params.RPS > 0 {
		totalBudgetReqs := int64(math.Round(totalBudgetDur.Seconds() * params.RPS))
		var consumedReqs int64
		if params.FailedRequests != nil {
			consumedReqs = *params.FailedRequests
		} else {
			consumedReqs = int64(math.Round(errorConsumedSec * params.RPS))
		}
		remainingReqs := totalBudgetReqs - consumedReqs
		res.RPS = params.RPS
		res.TotalBudgetRequests = &totalBudgetReqs
		res.ConsumedRequests = &consumedReqs
		res.RemainingRequests = &remainingReqs
	}

	return res, nil
}
