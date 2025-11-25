package burnrate

import (
	"errors"
	"math"
	"time"

	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/burnrate"
	burnratePort "github.com/MichielVanderhoydonck/sloak/internal/core/port/burnrate"
)

var _ burnratePort.BurnRateService = (*BurnRateServiceImpl)(nil)

type BurnRateServiceImpl struct{}

func NewBurnRateService() burnratePort.BurnRateService {
	return &BurnRateServiceImpl{}
}

func (s *BurnRateServiceImpl) CalculateBurnRate(params burnrateDomain.CalculationParams) (burnrateDomain.BurnRateResult, error) {
	errorBudgetPercent := 1.0 - (params.TargetSLO.Value / 100.0)
	totalBudget := time.Duration(math.Round(float64(params.TotalWindow) * errorBudgetPercent))

	if totalBudget <= 0 {
		return burnrateDomain.BurnRateResult{}, errors.New("total error budget is zero or negative")
	}

	budgetConsumedPercent := (float64(params.ErrorConsumed) / float64(totalBudget)) * 100.0
	timeElapsedPercent := (float64(params.TimeElapsed) / float64(params.TotalWindow)) * 100.0

	var burnRate float64
	if timeElapsedPercent > 0 {
		burnRate = budgetConsumedPercent / timeElapsedPercent
	} else {
		if params.ErrorConsumed > 0 {
			burnRate = math.Inf(1)
		} else {
			burnRate = 0.0
		}
	}

	budgetRemaining := totalBudget - params.ErrorConsumed

	var tte time.Duration
	isInfinite := false

	if budgetRemaining <= 0 {
		tte = 0
	} else if params.ErrorConsumed <= 0 {
		isInfinite = true
	} else {
		consumptionRate := float64(params.ErrorConsumed) / float64(params.TimeElapsed)
		tteNanos := float64(budgetRemaining) / consumptionRate
		tte = time.Duration(math.Round(tteNanos))
	}

	return burnrateDomain.BurnRateResult{
		TotalErrorBudget: totalBudget,
		BudgetConsumed:   budgetConsumedPercent,
		BurnRate:         burnRate,
		BudgetRemaining:  budgetRemaining,
		TimeToExhaustion: tte,
		IsInfinite:       isInfinite,
	}, nil
}
