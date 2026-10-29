package dependency

import (
	"errors"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type CalculationType string

const (
	Serial   CalculationType = "serial"
	Parallel CalculationType = "parallel"
)

type Status string

const (
	StatusMeetsTarget Status = "MEETS_TARGET"
	StatusBelowTarget Status = "BELOW_TARGET"
)

type Component struct {
	Name         string  `json:"name,omitempty"`
	Availability float64 `json:"availability"`
}

// Stage represents a single component or a parallel redundant group of components within a dependency chain.
type Stage struct {
	Name             string      `json:"name,omitempty"`
	Availability     float64     `json:"availability"`
	RedundantMembers []Component `json:"redundant_members,omitempty"`
}

type CalculationParams struct {
	Components []float64
	Stages     []Stage
	Type       CalculationType
	TargetSLO  *common.SLOTarget
	TimeWindow util.Duration
}

type Result struct {
	TotalAvailability float64           `json:"total_availability"`
	CalculationType   CalculationType   `json:"calculation_type"`
	ComponentCount    int               `json:"component_count"`
	AllowedDowntime   *util.Duration    `json:"allowed_downtime,omitempty"`
	TimeWindow        *util.Duration    `json:"time_window,omitempty"`
	Bottleneck        *Component        `json:"bottleneck,omitempty"`
	TargetSLO         *common.SLOTarget `json:"target_slo,omitempty"`
	MeetsTarget       *bool             `json:"meets_target,omitempty"`
	Status            Status            `json:"status,omitempty"`
}

func (p CalculationParams) Validate() error {
	if p.Type != Serial && p.Type != Parallel {
		return errors.New("type must be 'serial' or 'parallel'")
	}

	totalComponents := 0
	if len(p.Stages) > 0 {
		for _, st := range p.Stages {
			if len(st.RedundantMembers) > 0 {
				for _, m := range st.RedundantMembers {
					if m.Availability < 0 || m.Availability > 100 {
						return errors.New("component availability must be between 0 and 100")
					}
					totalComponents++
				}
			} else {
				if st.Availability < 0 || st.Availability > 100 {
					return errors.New("component availability must be between 0 and 100")
				}
				totalComponents++
			}
		}
	} else {
		totalComponents = len(p.Components)
		for _, c := range p.Components {
			if c < 0 || c > 100 {
				return errors.New("component availability must be between 0 and 100")
			}
		}
	}

	if totalComponents < 2 {
		return errors.New("at least two components are required for calculation")
	}

	if p.TargetSLO != nil {
		if err := p.TargetSLO.Validate(); err != nil {
			return err
		}
	}
	if p.TimeWindow < 0 {
		return errors.New("time window cannot be negative")
	}

	return nil
}

type Service interface {
	CalculateCompositeAvailability(params CalculationParams) (Result, error)
}
