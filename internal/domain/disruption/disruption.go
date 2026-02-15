package disruption

import (
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type CalculationParams struct {
	TargetSLO    common.SLOTarget
	TotalWindow  util.Duration
	CostPerEvent util.Duration
}

type Result struct {
	TotalErrorBudget        util.Duration `json:"total_error_budget"`
	TotalErrorBudgetSeconds float64       `json:"total_error_budget_seconds"`
	MaxDisruptions          int64         `json:"max_disruptions"`
	DailyDisruptions        float64       `json:"daily_disruptions"`
}
