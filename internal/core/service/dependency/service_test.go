package dependency_test

import (
	"testing"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/dependency"
	service "github.com/MichielVanderhoydonck/sloak/internal/core/service/dependency"
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
}
