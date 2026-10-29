package errorbudget

import (
	"errors"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type CalculationParams struct {
	TargetSLO  common.SLOTarget
	TimeWindow util.Duration
	RPS        float64
}

func (p CalculationParams) Validate() error {
	if err := p.TargetSLO.Validate(); err != nil {
		return err
	}
	if p.TimeWindow <= 0 {
		return errors.New("time window must be greater than zero")
	}
	if p.RPS < 0 {
		return errors.New("rps cannot be negative")
	}
	return nil
}

type BudgetResult struct {
	TargetSLO             common.SLOTarget `json:"target_slo"`
	TotalDuration         util.Duration    `json:"total_duration"`
	TotalDurationSeconds  float64          `json:"total_duration_seconds"`
	AllowedError          util.Duration    `json:"allowed_error"`
	AllowedErrorSeconds   float64          `json:"allowed_error_seconds"`
	ErrorBudget           float64          `json:"error_budget_percentage"`
	TargetSLORatio        float64          `json:"target_slo_ratio"`
	RPS                   float64          `json:"rps,omitempty"`
	TotalRequests         *int64           `json:"total_requests,omitempty"`
	AllowedFailedRequests *int64           `json:"allowed_failed_requests,omitempty"`
}

type Service interface {
	CalculateBudget(params CalculationParams) (BudgetResult, error)
}
