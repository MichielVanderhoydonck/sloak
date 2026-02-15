package errorbudget

import (
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type CalculationParams struct {
	TargetSLO common.SLOTarget

	TimeWindow util.Duration
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
