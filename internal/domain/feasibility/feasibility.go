package feasibility

import (
	"errors"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type Status string

const (
	StatusExtremelyHard Status = "EXTREMELY HARD"
	StatusChallenging   Status = "CHALLENGING"
	StatusRealistic     Status = "REALISTIC"
)

type FeasibilityParams struct {
	TargetSLO common.SLOTarget
	MTTR      util.Duration
}

func (p FeasibilityParams) Validate() error {
	if p.MTTR <= 0 {
		return errors.New("MTTR must be greater than zero")
	}
	if err := p.TargetSLO.Validate(); err != nil {
		return err
	}
	return nil
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
	Status              Status           `json:"status"`
}

func (r FeasibilityResult) ResolveStatus() Status {
	if r.Status != "" {
		return r.Status
	}
	return ClassifyFeasibility(r.IncidentsPerQuarter)
}

func ClassifyFeasibility(incidentsPerQuarter float64) Status {
	if incidentsPerQuarter < 1.0 {
		return StatusExtremelyHard
	}
	if incidentsPerQuarter < 3.0 {
		return StatusChallenging
	}
	return StatusRealistic
}

type Service interface {
	CalculateFeasibility(params FeasibilityParams) (FeasibilityResult, error)
}
