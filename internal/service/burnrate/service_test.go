package burnrate_test

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"

	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/burnrate"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	burnrateService "github.com/MichielVanderhoydonck/sloak/internal/service/burnrate"
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

func floatPtr(v float64) *float64 {
	return &v
}

func TestBurnRateService(t *testing.T) {
	svc := burnrateService.NewBurnRateService()

	testCases := []struct {
		name                 string
		params               burnrateDomain.CalculationParams
		expectedError        error
		expectedBurnRate     *float64
		expectedUnbounded    bool
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
			expectedBurnRate:     floatPtr(2.97619),
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
			expectedBurnRate:     floatPtr(1.0),
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
				TimeElapsed:   util.Duration(10 * 24 * time.Hour),
				ErrorConsumed: 0,
			},
			expectedError:        nil,
			expectedBurnRate:     floatPtr(0.0),
			expectedConsumedPct:  0.0,
			expectedRemainingDur: util.Duration((43 * time.Minute) + (12 * time.Second)),
			expectedTTE:          0,
			expectedInfinite:     true,
		},
		{
			name: "Instant Burn (Infinite/Unbounded)",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour),
				TimeElapsed:   0,
				ErrorConsumed: util.Duration(1 * time.Minute),
			},
			expectedError:        nil,
			expectedBurnRate:     nil,
			expectedUnbounded:    true,
			expectedConsumedPct:  2.31,
			expectedRemainingDur: util.Duration((42 * time.Minute) + (12 * time.Second)),
			expectedTTE:          0,
			expectedInfinite:     false,
		},
		{
			name: "Overspent Budget (Consumed > Budget, Negative Remaining)",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour), // budget: 43m12s
				TimeElapsed:   util.Duration(15 * 24 * time.Hour), // 50% elapsed
				ErrorConsumed: util.Duration(60 * time.Minute),    // 60m consumed
			},
			expectedError:        nil,
			expectedBurnRate:     floatPtr(2.78),
			expectedUnbounded:    false,
			expectedConsumedPct:  138.89,
			expectedRemainingDur: util.Duration(-(16*time.Minute + 48*time.Second)),
			expectedTTE:          0,
			expectedInfinite:     false,
		},
		{
			name: "Tiny Elapsed Time (1m elapsed, 1m consumed out of 30d)",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour),
				TimeElapsed:   util.Duration(1 * time.Minute),
				ErrorConsumed: util.Duration(1 * time.Minute),
			},
			expectedError:        nil,
			expectedBurnRate:     floatPtr(1000.0),
			expectedUnbounded:    false,
			expectedConsumedPct:  2.3148,
			expectedRemainingDur: util.Duration((42 * time.Minute) + (12 * time.Second)),
			expectedTTE:          util.Duration(42*time.Minute + 12*time.Second),
			expectedInfinite:     false,
		},
		{
			name: "Tiny Elapsed Time Zero Consumed (1m elapsed, 0s consumed out of 30d)",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour),
				TimeElapsed:   util.Duration(1 * time.Minute),
				ErrorConsumed: 0,
			},
			expectedError:        nil,
			expectedBurnRate:     floatPtr(0.0),
			expectedUnbounded:    false,
			expectedConsumedPct:  0.0,
			expectedRemainingDur: util.Duration((43 * time.Minute) + (12 * time.Second)),
			expectedTTE:          0,
			expectedInfinite:     true,
		},
		{
			name: "Zero Total Window Error",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   0,
				TimeElapsed:   util.Duration(1 * time.Hour),
				ErrorConsumed: util.Duration(1 * time.Minute),
			},
			expectedError: errors.New("total window must be greater than zero"),
		},
		{
			name: "Negative Elapsed Error",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour),
				TimeElapsed:   util.Duration(-1 * time.Hour),
				ErrorConsumed: util.Duration(1 * time.Minute),
			},
			expectedError: errors.New("time elapsed cannot be negative"),
		},
		{
			name: "Negative Consumed Error",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(30 * 24 * time.Hour),
				TimeElapsed:   util.Duration(1 * time.Hour),
				ErrorConsumed: util.Duration(-1 * time.Minute),
			},
			expectedError: errors.New("error consumed cannot be negative"),
		},
		{
			name: "Elapsed Exceeds Window Error",
			params: burnrateDomain.CalculationParams{
				TargetSLO:     mustNewSLO(t, 99.9),
				TotalWindow:   util.Duration(7 * 24 * time.Hour),
				TimeElapsed:   util.Duration(8 * 24 * time.Hour),
				ErrorConsumed: util.Duration(1 * time.Minute),
			},
			expectedError: errors.New("time elapsed cannot exceed total window"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.CalculateBurnRate(tc.params)

			if tc.expectedError != nil {
				if err == nil || err.Error() != tc.expectedError.Error() {
					t.Fatalf("expected error '%v', but got '%v'", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error, but got: %v", err)
				}

				if tc.expectedBurnRate == nil {
					if result.BurnRate != nil {
						t.Errorf("expected burn rate nil (unbounded), but got %.2f", *result.BurnRate)
					}
					if !result.BurnRateUnbounded {
						t.Errorf("expected BurnRateUnbounded true, got false")
					}
				} else {
					if result.BurnRate == nil {
						t.Fatalf("expected burn rate %.2f, got nil", *tc.expectedBurnRate)
					}
					if !approxEqual(*result.BurnRate, *tc.expectedBurnRate, 0.01) {
						t.Errorf("expected burn rate %.2f, but got %.2f", *tc.expectedBurnRate, *result.BurnRate)
					}
				}

				if !approxEqual(result.BudgetConsumed, tc.expectedConsumedPct, 0.01) {
					t.Errorf("expected consumed pct %.2f, but got %.2f", tc.expectedConsumedPct, result.BudgetConsumed)
				}
				if result.BudgetRemaining.String() != tc.expectedRemainingDur.String() {
					t.Errorf("expected remaining duration %v, but got %v", tc.expectedRemainingDur, result.BudgetRemaining)
				}

				if result.NoBudgetConsumed != tc.expectedInfinite {
					t.Errorf("expected NoBudgetConsumed %v, got %v", tc.expectedInfinite, result.NoBudgetConsumed)
				}

				if !tc.expectedInfinite && tc.expectedTTE > 0 {
					if result.TimeToExhaustion.String() != tc.expectedTTE.String() {
						t.Errorf("expected TTE %v, got %v", tc.expectedTTE, result.TimeToExhaustion)
					}
				}

				// Verify JSON encoding always succeeds
				jsonBytes, err := json.Marshal(result)
				if err != nil {
					t.Fatalf("JSON encoding failed: %v", err)
				}
				if len(jsonBytes) == 0 {
					t.Fatal("JSON output is empty")
				}
			}
		})
	}
}
