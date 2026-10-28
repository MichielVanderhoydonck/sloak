package disruption

import (
	"errors"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type Status string

const (
	StatusBlocked Status = "BLOCKED"
	StatusRisky   Status = "RISKY"
	StatusSafe    Status = "SAFE"
)

type CalculationParams struct {
	TargetSLO    common.SLOTarget
	TotalWindow  util.Duration
	CostPerEvent util.Duration
}

func (p CalculationParams) Validate() error {
	if err := p.TargetSLO.Validate(); err != nil {
		return err
	}
	if p.TotalWindow <= 0 {
		return errors.New("window must be greater than zero")
	}
	if p.CostPerEvent <= 0 {
		return errors.New("cost per disruption must be greater than zero")
	}
	if p.CostPerEvent > p.TotalWindow {
		return errors.New("cost per disruption cannot exceed total window")
	}
	return nil
}

type Result struct {
	TotalErrorBudget        util.Duration `json:"total_error_budget"`
	TotalErrorBudgetSeconds float64       `json:"total_error_budget_seconds"`
	MaxDisruptions          int64         `json:"max_disruptions"`
	DailyDisruptions        float64       `json:"daily_disruptions"`
	Status                  Status        `json:"status"`
}

func (r Result) ResolveStatus() Status {
	if r.Status != "" {
		return r.Status
	}
	return ClassifyDisruption(r.MaxDisruptions, r.DailyDisruptions)
}

func ClassifyDisruption(maxDisruptions int64, dailyDisruptions float64) Status {
	if maxDisruptions < 1 {
		return StatusBlocked
	}
	if dailyDisruptions < 1.0 {
		return StatusRisky
	}
	return StatusSafe
}

type Service interface {
	CalculateCapacity(params CalculationParams) (Result, error)
}
