package feasibility

import (
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type FeasibilityParams struct {
	TargetSLO common.SLOTarget
	MTTR      util.Duration
}

type FeasibilityResult struct {
	TargetSLO           common.SLOTarget `json:"target_slo"`
	TargetSLORatio      float64          `json:"target_slo_ratio"`
	MTTR                util.Duration    `json:"mttr"`
	MTTRSeconds         float64          `json:"mttr_seconds"`
	IncidentsPerYear    float64          `json:"incidents_per_year"`
	IncidentsPerQuarter float64          `json:"incidents_per_quarter"`
	IncidentsPerMonth   float64          `json:"incidents_per_month"`
	RequiredMTBF        util.Duration    `json:"required_mtbf"`
	RequiredMTBFSeconds float64          `json:"required_mtbf_seconds"`
}
