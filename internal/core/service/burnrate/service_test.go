package burnrate_test

import (
	"errors"
	"math"
	"testing"
	"time"

	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/burnrate"
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	burnrateService "github.com/MichielVanderhoydonck/sloak/internal/core/service/burnrate"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func mustNewSLO(t *testing.T, val float64) common.SLOTarget {
	slo, err := common.NewSLOTarget(val)
	if err != nil {
		t.Fatalf("failed to create valid SLO for test: %v", err)
	}
	return slo
}

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestBurnRateService(t *testing.T) {
	svc := burnrateService.NewBurnRateService()

	testCases := []struct {
		name                 string
		params               burnrateDomain.CalculationParams
		expectedError        error
		expectedBurnRate     float64
		expectedConsumedPct  float64
		expectedRemainingDur util.Duration
		expectedTTE          util.Duration
		expectedInfinite     bool
	}{
		{
			name: "Critical Burn (2.98x)",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour), // 720h
				TimeElapsed:   util.Duration(7 * 24 * time.Hour),  // 168h
				ErrorConsumed: util.Duration(30 * time.Minute),
			},
			expectedError:        nil,
			expectedBurnRate:     2.97619,
			expectedConsumedPct:  69.444,
			expectedRemainingDur: util.Duration((13 * time.Minute) + (12 * time.Second)),
			expectedTTE:          util.Duration(73*time.Hour + 55*time.Minute + 12*time.Second),
			expectedInfinite:     false,
		},
		{
			name: "Ideal Burn (1x)",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour),
				TimeElapsed:   util.Duration(15 * 24 * time.Hour),
				ErrorConsumed: util.Duration((21 * time.Minute) + (36 * time.Second)),
			},
			expectedError:        nil,
			expectedBurnRate:     1.0,
			expectedConsumedPct:  50.0,
			expectedRemainingDur: util.Duration((21 * time.Minute) + (36 * time.Second)),
			expectedTTE:          util.Duration(15 * 24 * time.Hour),
			expectedInfinite:     false,
		},
		{
			name: "No Burn (0x)",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour),
				TimeElapsed:   util.Duration(7 * 24 * time.Hour),
				ErrorConsumed: 0,
			},
			expectedError:        nil,
			expectedBurnRate:     0.0,
			expectedConsumedPct:  0.0,
			expectedRemainingDur: util.Duration((43 * time.Minute) + (12 * time.Second)),
			expectedTTE:          0,
			expectedInfinite:     true,
		},
		{
			name: "Instant Burn (Infinite)",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour),
				TimeElapsed:   0,
				ErrorConsumed: util.Duration(1 * time.Minute),
			},
			expectedError:        nil,
			expectedBurnRate:     math.Inf(1),
			expectedConsumedPct:  2.31,
			expectedRemainingDur: util.Duration((42 * time.Minute) + (12 * time.Second)),
			expectedTTE:          0,
			expectedInfinite:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.CalculateBurnRate(tc.params)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected error '%v', but got '%v'", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error, but got: %v", err)
				}

				if math.IsInf(tc.expectedBurnRate, 1) {
					if !math.IsInf(result.BurnRate, 1) {
						t.Errorf("expected burn rate +Inf, but got %.2f", result.BurnRate)
					}
				} else if !approxEqual(result.BurnRate, tc.expectedBurnRate, 0.01) {
					t.Errorf("expected burn rate %.2f, but got %.2f", tc.expectedBurnRate, result.BurnRate)
				}

				if !approxEqual(result.BudgetConsumed, tc.expectedConsumedPct, 0.01) {
					t.Errorf("expected consumed pct %.2f, but got %.2f", tc.expectedConsumedPct, result.BudgetConsumed)
				}
				if result.BudgetRemaining.String() != tc.expectedRemainingDur.String() {
					t.Errorf("expected remaining duration %v, but got %v", tc.expectedRemainingDur, result.BudgetRemaining)
				}

				if result.IsInfinite != tc.expectedInfinite {
					t.Errorf("expected IsInfinite %v, got %v", tc.expectedInfinite, result.IsInfinite)
				}

				if !tc.expectedInfinite && tc.expectedTTE > 0 {
					if result.TimeToExhaustion.String() != tc.expectedTTE.String() {
						t.Errorf("expected TTE %v, got %v", tc.expectedTTE, result.TimeToExhaustion)
					}
				}
			}
		})
	}
}
