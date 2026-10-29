package burnrate

import (
	"errors"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type Status string

const (
	StatusExcellent Status = "EXCELLENT"
	StatusHealthy   Status = "HEALTHY"
	StatusWarning   Status = "WARNING"
	StatusHigh      Status = "HIGH"
	StatusCritical  Status = "CRITICAL"
)

const (
	PageBurnRateThreshold    = 14.4
	MessageBurnRateThreshold = 6.0
	TicketBurnRateThreshold  = 1.0
)

type CalculationParams struct {
	TargetSLO      common.SLOTarget
	TotalWindow    util.Duration
	TimeElapsed    util.Duration
	ErrorConsumed  util.Duration
	RPS            float64
	FailedRequests *int64
}

func (p CalculationParams) Validate() error {
	if err := p.TargetSLO.Validate(); err != nil {
		return err
	}
	if p.TotalWindow <= 0 {
		return errors.New("total window must be greater than zero")
	}
	if p.TimeElapsed < 0 {
		return errors.New("time elapsed cannot be negative")
	}
	if p.TimeElapsed > p.TotalWindow {
		return errors.New("time elapsed cannot exceed total window")
	}
	if p.ErrorConsumed < 0 {
		return errors.New("error consumed cannot be negative")
	}
	if p.RPS < 0 {
		return errors.New("rps cannot be negative")
	}
	if p.FailedRequests != nil {
		if *p.FailedRequests < 0 {
			return errors.New("failed requests cannot be negative")
		}
		if p.RPS <= 0 {
			return errors.New("rps must be greater than zero when failed requests are specified")
		}
	}
	return nil
}

type BurnRateResult struct {
	TotalErrorBudget        util.Duration `json:"total_error_budget"`
	TotalErrorBudgetSeconds float64       `json:"total_error_budget_seconds"`
	BudgetRemaining         util.Duration `json:"budget_remaining"`
	BudgetRemainingSeconds  float64       `json:"budget_remaining_seconds"`
	BudgetConsumed          float64       `json:"budget_consumed_percentage"`
	TimeElapsedPercent      float64       `json:"time_elapsed_percentage"`
	BurnRate                *float64      `json:"burn_rate"`
	BurnRateUnbounded       bool          `json:"burn_rate_unbounded,omitempty"`
	TimeToExhaustion        util.Duration `json:"time_to_exhaustion"`
	TimeToExhaustionSeconds float64       `json:"time_to_exhaustion_seconds"`
	NoBudgetConsumed        bool          `json:"no_budget_consumed"`
	Status                  Status        `json:"status"`
	RPS                     float64       `json:"rps,omitempty"`
	TotalBudgetRequests     *int64        `json:"total_error_budget_requests,omitempty"`
	ConsumedRequests        *int64        `json:"consumed_requests,omitempty"`
	RemainingRequests       *int64        `json:"remaining_requests,omitempty"`
}

// ResolveStatus returns the effective Status, classifying from BurnRate/flags if Status wasn't explicitly set.
func (r BurnRateResult) ResolveStatus() Status {
	if r.Status != "" {
		return r.Status
	}
	return ClassifyBurnRate(r.BurnRate, r.BurnRateUnbounded, r.NoBudgetConsumed)
}

func ClassifyBurnRate(burnRate *float64, unbounded, noBudgetConsumed bool) Status {
	switch {
	case noBudgetConsumed:
		return StatusExcellent
	case unbounded:
		return StatusCritical
	case burnRate != nil && *burnRate >= PageBurnRateThreshold:
		return StatusCritical
	case burnRate != nil && *burnRate >= MessageBurnRateThreshold:
		return StatusHigh
	case burnRate != nil && *burnRate > TicketBurnRateThreshold:
		return StatusWarning
	default:
		return StatusHealthy
	}
}

type Service interface {
	CalculateBurnRate(params CalculationParams) (BurnRateResult, error)
}
