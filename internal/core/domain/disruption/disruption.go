package disruption

import (
	"time"
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
)

type CalculationParams struct {
	TargetSLO      common.SLOTarget
	TotalWindow    time.Duration
	CostPerEvent   time.Duration
}

type Result struct {
	TotalErrorBudget time.Duration
	MaxDisruptions   int64
	DailyDisruptions float64
}