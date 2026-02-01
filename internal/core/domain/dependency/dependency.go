package dependency

import "errors"

type CalculationType string

const (
	Serial   CalculationType = "serial"
	Parallel CalculationType = "parallel"
)

type CalculationParams struct {
	Components []float64

	Type CalculationType
}

type Result struct {
	TotalAvailability float64         `json:"total_availability"`
	CalculationType   CalculationType `json:"calculation_type"`
	ComponentCount    int             `json:"component_count"`
}

func (p CalculationParams) Validate() error {
	if len(p.Components) < 2 {
		return errors.New("at least two components are required for calculation")
	}
	if p.Type != Serial && p.Type != Parallel {
		return errors.New("type must be 'serial' or 'parallel'")
	}
	for _, c := range p.Components {
		if c < 0 || c > 100 {
			return errors.New("component availability must be between 0 and 100")
		}
	}
	return nil
}
