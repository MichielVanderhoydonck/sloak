package errorbudget

import (
	"errors"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type CalculationParams struct {
	TargetSLO  common.SLOTarget
	TimeWindow util.Duration
}

func (p CalculationParams) Validate() error {
	if err := p.TargetSLO.Validate(); err != nil {
		return err
	}
	if p.TimeWindow <= 0 {
		return errors.New("time window must be greater than zero")
	}
	return nil
}

type BudgetResult struct {
	TargetSLO            common.SLOTarget `json:"target_slo"`
	TotalDuration        util.Duration    `json:"total_duration"`
	TotalDurationSeconds float64          `json:"total_duration_seconds"`
	AllowedError         util.Duration    `json:"allowed_error"`
	AllowedErrorSeconds  float64          `json:"allowed_error_seconds"`
	ErrorBudget          float64          `json:"error_budget_percentage"`
	TargetSLORatio       float64          `json:"target_slo_ratio"`
}

type Service interface {
	CalculateBudget(params CalculationParams) (BudgetResult, error)
}
