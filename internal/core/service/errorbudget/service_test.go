package errorbudget_test

import (
	"errors"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	errorbudgetDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/errorbudget"
	errorbudgetService "github.com/MichielVanderhoydonck/sloak/internal/core/service/errorbudget"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func TestCalculatorService(t *testing.T) {
	svc := errorbudgetService.NewCalculatorService()

	mustNewSLO := func(val float64) common.SLOTarget {
		slo, err := common.NewSLOTarget(val)
		if err != nil {
			t.Fatalf("failed to create valid SLO for test: %v", err)
		}
		return slo
	}

	testCases := []struct {
		name             string
		params           errorbudgetDomain.CalculationParams
		expectedError    error
		expectedDowntime util.Duration
	}{
		{
			name: "99.9% over 30 days",
			params: errorbudgetDomain.CalculationParams{
				TargetSLO:  mustNewSLO(99.9),
				TimeWindow: util.Duration(30 * 24 * time.Hour), // 720h
			},
			expectedError:    nil,
			expectedDowntime: util.Duration((43 * time.Minute) + (12 * time.Second)), // 0.1% of 720h
		},
		{
			name: "99.95% over 30 days",
			params: errorbudgetDomain.CalculationParams{
				TargetSLO:  mustNewSLO(99.95),
				TimeWindow: util.Duration(30 * 24 * time.Hour), // 720h
			},
			expectedError:    nil,
			expectedDowntime: util.Duration((21 * time.Minute) + (36 * time.Second)), // 0.05% of 720h
		},
		{
			name: "95% over 7 days",
			params: errorbudgetDomain.CalculationParams{
				TargetSLO:  mustNewSLO(95.0),
				TimeWindow: util.Duration(7 * 24 * time.Hour), // 168h
			},
			expectedError:    nil,
			expectedDowntime: util.Duration((8 * time.Hour) + (24 * time.Minute)), // 5% of 168h
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.CalculateBudget(tc.params)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected error '%v', but got '%v'", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error, but got: %v", err)
				}
				if result.AllowedError != tc.expectedDowntime {
					t.Errorf("expected downtime %v, but got %v", tc.expectedDowntime, result.AllowedError)
				}
				if result.TargetSLO.Value != tc.params.TargetSLO.Value {
					t.Error("result did not contain the original SLO target")
				}
			}
		})
	}
}
