package errorbudget

import (
	"math"
	"time"

	errorbudgetDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/errorbudget"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type CalculatorService struct{}

var _ errorbudgetDomain.Service = (*CalculatorService)(nil)

func NewCalculatorService() errorbudgetDomain.Service {
	return &CalculatorService{}
}

func (s *CalculatorService) CalculateBudget(params errorbudgetDomain.CalculationParams) (errorbudgetDomain.BudgetResult, error) {
	if err := params.Validate(); err != nil {
		return errorbudgetDomain.BudgetResult{}, err
	}

	window := time.Duration(params.TimeWindow)
	allowedError := params.TargetSLO.AllowedDowntime(window)
	errorBudgetRatio := params.TargetSLO.ErrorBudgetRatio()

	res := errorbudgetDomain.BudgetResult{
		TargetSLO:            params.TargetSLO,
		TotalDuration:        params.TimeWindow,
		TotalDurationSeconds: math.Round(window.Seconds()),
		AllowedError:         util.Duration(allowedError),
		AllowedErrorSeconds:  math.Round(allowedError.Seconds()),
		ErrorBudget:          util.RoundPercentage(errorBudgetRatio * 100.0),
		TargetSLORatio:       params.TargetSLO.RoundedRatio(),
	}

	if params.RPS > 0 {
		totalReqs := int64(math.Round(params.RPS * window.Seconds()))
		allowedFailed := int64(math.Round(params.RPS * allowedError.Seconds()))
		res.RPS = params.RPS
		res.TotalRequests = &totalReqs
		res.AllowedFailedRequests = &allowedFailed
	}

	return res, nil
}
