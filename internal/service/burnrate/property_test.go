package burnrate_test

import (
	"math"
	"math/rand"
	"testing"
	"testing/quick"
	"time"

	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/burnrate"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/burnrate"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func TestProperty_BurnRateInvariants(t *testing.T) {
	svc := service.NewBurnRateService()

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Random SLO in (90.0, 99.99)
		sloVal := r.Float64()*9.99 + 90.0
		sloVO, err := common.NewSLOTarget(sloVal)
		if err != nil {
			return false
		}

		// Random window between 1 day and 90 days
		totalWindowSecs := float64(r.Int63n(89*86400) + 86400)
		totalWindow := time.Duration(totalWindowSecs * float64(time.Second))

		// Total error budget duration
		budgetSecs := totalWindowSecs * (1.0 - (sloVal / 100.0))
		totalBudgetDur := time.Duration(budgetSecs * float64(time.Second))
		if totalBudgetDur <= 0 {
			return true
		}

		// Pick an elapsed fraction in (0.01, 1.0)
		elapsedFraction := r.Float64()*0.99 + 0.01
		elapsedSecs := totalWindowSecs * elapsedFraction
		timeElapsed := time.Duration(elapsedSecs * float64(time.Second))

		// Test Case 1: When consumed fraction == elapsed fraction, burn rate must be ~1.0
		// consumed = totalBudget * elapsedFraction
		consumedSecs := budgetSecs * elapsedFraction
		errorConsumed := time.Duration(consumedSecs * float64(time.Second))

		res, err := svc.CalculateBurnRate(burnrateDomain.CalculationParams{
			TargetSLO:     sloVO,
			TotalWindow:   util.Duration(totalWindow),
			TimeElapsed:   util.Duration(timeElapsed),
			ErrorConsumed: util.Duration(errorConsumed),
		})
		if err != nil {
			t.Logf("unexpected error for 1.0x burnrate: %v", err)
			return false
		}

		if res.BurnRate == nil {
			t.Log("expected non-nil burn rate")
			return false
		}
		if math.Abs(*res.BurnRate-1.0) > 0.05 {
			t.Logf("expected burn rate ~1.0, got %v (consumedPct=%v, elapsedPct=%v)", *res.BurnRate, res.BudgetConsumed, res.TimeElapsedPercent)
			return false
		}

		// Test Case 2: Invariant remaining + consumed == totalBudget (within 1 second)
		remainingDur := time.Duration(res.BudgetRemaining)
		sum := remainingDur + errorConsumed
		if math.Abs(float64(sum-totalBudgetDur)) > float64(time.Second) {
			t.Logf("budget sum mismatch: remaining=%v, consumed=%v, total=%v", remainingDur, errorConsumed, totalBudgetDur)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("quick check failed: %v", err)
	}
}

func TestProperty_BurnRateDirectionality(t *testing.T) {
	svc := service.NewBurnRateService()

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		sloVal := 99.0
		sloVO, _ := common.NewSLOTarget(sloVal)

		totalWindow := 30 * 24 * time.Hour
		elapsedPct := r.Float64()*0.4 + 0.3 // 30% to 70% elapsed
		timeElapsed := time.Duration(float64(totalWindow) * elapsedPct)
		totalBudget := time.Duration(float64(totalWindow) * (1.0 - (sloVal / 100.0)))

		// If consumed fraction > elapsed fraction, burn rate must be > 1.0
		consumedHighPct := elapsedPct + 0.15
		consumedHigh := time.Duration(float64(totalBudget) * consumedHighPct)
		resHigh, err := svc.CalculateBurnRate(burnrateDomain.CalculationParams{
			TargetSLO:     sloVO,
			TotalWindow:   util.Duration(totalWindow),
			TimeElapsed:   util.Duration(timeElapsed),
			ErrorConsumed: util.Duration(consumedHigh),
		})
		if err != nil || resHigh.BurnRate == nil || *resHigh.BurnRate <= 1.0 {
			t.Logf("expected burnRate > 1.0 for high consumption, got %v", resHigh.BurnRate)
			return false
		}

		// If consumed fraction < elapsed fraction, burn rate must be < 1.0
		consumedLowPct := elapsedPct - 0.15
		consumedLow := time.Duration(float64(totalBudget) * consumedLowPct)
		resLow, err := svc.CalculateBurnRate(burnrateDomain.CalculationParams{
			TargetSLO:     sloVO,
			TotalWindow:   util.Duration(totalWindow),
			TimeElapsed:   util.Duration(timeElapsed),
			ErrorConsumed: util.Duration(consumedLow),
		})
		if err != nil || resLow.BurnRate == nil || *resLow.BurnRate >= 1.0 {
			t.Logf("expected burnRate < 1.0 for low consumption, got %v", resLow.BurnRate)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 500}); err != nil {
		t.Errorf("quick check failed: %v", err)
	}
}
