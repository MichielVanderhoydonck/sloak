package burnrate

import (
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type CalculationParams struct {
	TargetSLO     common.SLOTarget
	TotalWindow   util.Duration
	TimeElapsed   util.Duration
	ErrorConsumed util.Duration
}

type BurnRateResult struct {
	TotalErrorBudget        util.Duration `json:"total_error_budget"`
	TotalErrorBudgetSeconds float64       `json:"total_error_budget_seconds"`
	BudgetRemaining         util.Duration `json:"budget_remaining"`
	BudgetRemainingSeconds  float64       `json:"budget_remaining_seconds"`
	BudgetConsumed          float64       `json:"budget_consumed_percentage"`
	BurnRate                float64       `json:"burn_rate"`
	TimeToExhaustion        util.Duration `json:"time_to_exhaustion"`
	TimeToExhaustionSeconds float64       `json:"time_to_exhaustion_seconds"`
	IsInfinite              bool          `json:"is_infinite"`
}
