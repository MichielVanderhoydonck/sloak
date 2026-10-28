package disruption_test

import (
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/disruption"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/disruption"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func TestDisruptionService(t *testing.T) {
	svc := service.NewDisruptionService()
	slo999, _ := common.NewSLOTarget(99.9)

	// Scenario: 99.9% SLO over 30 days. Budget ~43m.
	// Cost per deploy = 1 minute.
	// Expect ~43 deploys total.
	params := domain.CalculationParams{
		TargetSLO:    slo999,
		TotalWindow:  util.Duration(30 * 24 * time.Hour),
		CostPerEvent: util.Duration(1 * time.Minute),
	}

	res, err := svc.CalculateCapacity(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.MaxDisruptions != 43 {
		t.Errorf("Expected 43 disruptions, got %d", res.MaxDisruptions)
	}

	// 43 / 30 days = 1.43 per day
	if res.DailyDisruptions < 1.4 || res.DailyDisruptions > 1.5 {
		t.Errorf("Expected ~1.43 daily, got %f", res.DailyDisruptions)
	}
	if res.Status != domain.StatusSafe {
		t.Errorf("Expected status SAFE, got %s", res.Status)
	}
}

func TestDisruptionService_Validation(t *testing.T) {
	svc := service.NewDisruptionService()
	slo999, _ := common.NewSLOTarget(99.9)

	tests := []struct {
		name        string
		params      domain.CalculationParams
		errContains string
	}{
		{
			name: "Zero window",
			params: domain.CalculationParams{
				TargetSLO:    slo999,
				TotalWindow:  0,
				CostPerEvent: util.Duration(1 * time.Minute),
			},
			errContains: "window must be greater than zero",
		},
		{
			name: "Zero cost per disruption",
			params: domain.CalculationParams{
				TargetSLO:    slo999,
				TotalWindow:  util.Duration(30 * 24 * time.Hour),
				CostPerEvent: 0,
			},
			errContains: "cost per disruption must be greater than zero",
		},
		{
			name: "Cost exceeds window",
			params: domain.CalculationParams{
				TargetSLO:    slo999,
				TotalWindow:  util.Duration(1 * time.Hour),
				CostPerEvent: util.Duration(2 * time.Hour),
			},
			errContains: "cost per disruption cannot exceed total window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CalculateCapacity(tt.params)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errContains)
			}
		})
	}
}
