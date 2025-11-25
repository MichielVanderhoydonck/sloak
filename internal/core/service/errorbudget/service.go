package errorbudget

import (
	"errors"
	errorbudgetDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/errorbudget"
	errorbudgetPort "github.com/MichielVanderhoydonck/sloak/internal/core/port/errorbudget"

	"math"
	"time"
)

var _ errorbudgetPort.CalculatorService = (*CalculatorServiceImpl)(nil)

type CalculatorServiceImpl struct{}

func NewCalculatorService() errorbudgetPort.CalculatorService {
	return &CalculatorServiceImpl{}
}

func (s *CalculatorServiceImpl) CalculateBudget(params errorbudgetDomain.CalculationParams) (errorbudgetDomain.BudgetResult, error) {
	if params.TargetSLO.Value < 0 || params.TargetSLO.Value > 100 {
		return errorbudgetDomain.BudgetResult{}, errors.New("SLO target must be between 0 and 100")
	}

	errorBudgetPercent := 1.0 - (params.TargetSLO.Value / 100.0)
	allowedErrorNanos := float64(params.TimeWindow) * errorBudgetPercent
	roundedNanos := int64(math.Round(allowedErrorNanos))
	allowedError := time.Duration(roundedNanos)

	return errorbudgetDomain.BudgetResult{
		TargetSLO:     params.TargetSLO,
		TotalDuration: params.TimeWindow,
		AllowedError:  allowedError,
		ErrorBudget:   errorBudgetPercent,
	}, nil
}
