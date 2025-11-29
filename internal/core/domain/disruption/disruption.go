package disruption

import (
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	"time"
)

type CalculationParams struct {
	TargetSLO    common.SLOTarget
	TotalWindow  time.Duration
	CostPerEvent time.Duration
}

type Result struct {
	TotalErrorBudget time.Duration
	MaxDisruptions   int64
	DailyDisruptions float64
}
