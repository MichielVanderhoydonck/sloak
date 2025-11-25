package burnrate

import (
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	"time"
)

type CalculationParams struct {
	TargetSLO   common.SLOTarget
	TotalWindow time.Duration

	TimeElapsed   time.Duration
	ErrorConsumed time.Duration
}

type BurnRateResult struct {
	TotalErrorBudget time.Duration
	BudgetConsumed   float64
	BurnRate         float64
	BudgetRemaining  time.Duration
	TimeToExhaustion time.Duration
	IsInfinite       bool
}
