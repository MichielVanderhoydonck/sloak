package errorbudget

import (
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	"time"
)

type CalculationParams struct {
	TargetSLO common.SLOTarget

	TimeWindow time.Duration
}

type BudgetResult struct {
	TargetSLO common.SLOTarget

	TotalDuration time.Duration

	AllowedError time.Duration

	ErrorBudget float64
}
