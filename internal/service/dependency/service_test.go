package dependency_test

import (
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/dependency"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/dependency"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func TestCalculateCompositeAvailability(t *testing.T) {
	svc := service.NewAvailabilityService()

	testCases := []struct {
		name     string
		params   domain.CalculationParams
		expected float64
		wantErr  bool
	}{
		{
			name: "Serial: Two 99.9s",
			params: domain.CalculationParams{
				Type:       domain.Serial,
				Components: []float64{99.9, 99.9},
			},
			expected: 99.8001, // 0.999 * 0.999
			wantErr:  false,
		},
		{
			name: "Parallel: Two 99.9s",
			params: domain.CalculationParams{
				Type:       domain.Parallel,
				Components: []float64{99.9, 99.9},
			},
			expected: 99.9999, // 1 - (0.001 * 0.001)
			wantErr:  false,
		},
		{
			name: "Mixed Serial with Inline Parallel Redundant Stage",
			params: domain.CalculationParams{
				Type: domain.Serial,
				Stages: []domain.Stage{
					{Name: "api-gateway", Availability: 99.99},
					{
						RedundantMembers: []domain.Component{
							{Name: "db-primary", Availability: 99.9},
							{Name: "db-replica", Availability: 99.9},
						},
					},
				},
			},
			expected: 99.9899, // 0.9999 * 0.999999 * 100
			wantErr:  false,
		},
		{
			name: "Validation Error",
			params: domain.CalculationParams{
				Type:       domain.Serial,
				Components: []float64{99.9}, // Too few
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := svc.CalculateCompositeAvailability(tc.params)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Simple float check for this example (epsilon 0.000001)
				diff := res.TotalAvailability - tc.expected
				if diff < -0.000001 || diff > 0.000001 {
					t.Errorf("expected %.6f, got %.6f", tc.expected, res.TotalAvailability)
				}
			}
		})
	}

	t.Run("Bottleneck, AllowedDowntime, and TargetSLO evaluation", func(t *testing.T) {
		sloTarget, _ := common.NewSLOTarget(99.9)
		res, err := svc.CalculateCompositeAvailability(domain.CalculationParams{
			Type: domain.Serial,
			Stages: []domain.Stage{
				{Name: "api", Availability: 99.99},
				{Name: "auth", Availability: 99.9},
				{Name: "db", Availability: 99.95},
			},
			TargetSLO:  &sloTarget,
			TimeWindow: util.Duration(30 * 24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Bottleneck == nil || res.Bottleneck.Name != "auth" {
			t.Errorf("expected bottleneck 'auth', got %+v", res.Bottleneck)
		}
		if res.Status != domain.StatusBelowTarget || res.MeetsTarget == nil || *res.MeetsTarget {
			t.Errorf("expected BELOW_TARGET for 99.8401%% vs 99.9%% target, got status=%s", res.Status)
		}
		if res.AllowedDowntime == nil || *res.AllowedDowntime <= 0 {
			t.Errorf("expected positive AllowedDowntime over 30d, got %v", res.AllowedDowntime)
		}
	})
}
