package feasibility

import (
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
)

type FeasibilityParams struct {
	TargetSLO common.SLOTarget
	MTTR      time.Duration
}

type FeasibilityResult struct {
	TargetSLO    common.SLOTarget
	MTTR         time.Duration
	
	IncidentsPerYear    float64
	IncidentsPerQuarter float64
	IncidentsPerMonth   float64
	
	RequiredMTBF        time.Duration
}