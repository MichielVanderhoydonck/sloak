package disruption_test

import (
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/disruption"
	service "github.com/MichielVanderhoydonck/sloak/internal/core/service/disruption"
)

func TestDisruptionService(t *testing.T) {
	svc := service.NewDisruptionService()
	slo999, _ := common.NewSLOTarget(99.9)

	// Scenario: 99.9% SLO over 30 days. Budget ~43m.
	// Cost per deploy = 1 minute.
	// Expect ~43 deploys total.
	params := domain.CalculationParams{
		TargetSLO:    slo999,
		TotalWindow:  30 * 24 * time.Hour,
		CostPerEvent: 1 * time.Minute,
	}

	res, err := svc.CalculateCapacity(params)
	if err != nil { t.Fatalf("unexpected error: %v", err) }

	if res.MaxDisruptions != 43 {
		t.Errorf("Expected 43 disruptions, got %d", res.MaxDisruptions)
	}
	
	// 43 / 30 days = 1.43 per day
	if res.DailyDisruptions < 1.4 || res.DailyDisruptions > 1.5 {
		t.Errorf("Expected ~1.43 daily, got %f", res.DailyDisruptions)
	}
}