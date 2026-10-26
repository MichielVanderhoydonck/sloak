package errorbudget_test

import (
	"math"
	"testing"
	"testing/quick"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	errorbudgetDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/errorbudget"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/errorbudget"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func TestProperty_ErrorBudgetInvariants(t *testing.T) {
	svc := service.NewCalculatorService()

	// Property 1: AllowedError + ExpectedUptime == TotalDuration (within nanosecond rounding)
	f := func(sloVal float64, windowSeconds int64) bool {
		// Constrain sloVal to (0.01, 99.999)
		slo := math.Mod(math.Abs(sloVal), 99.99) + 0.005
		if slo <= 0 || slo >= 100 {
			return true
		}

		// Constrain window to 1 minute ... 5 years
		secs := (math.Abs(float64(windowSeconds)))
		if secs < 60 {
			secs = 60
		}
		if secs > 5*365*24*3600 {
			secs = 5 * 365 * 24 * 3600
		}
		totalWin := time.Duration(secs) * time.Second

		sloVO, err := common.NewSLOTarget(slo)
		if err != nil {
			return true
		}

		res, err := svc.CalculateBudget(errorbudgetDomain.CalculationParams{
			TargetSLO:  sloVO,
			TimeWindow: util.Duration(totalWin),
		})
		if err != nil {
			t.Logf("unexpected error: %v", err)
			return false
		}

		// Check: AllowedError is strictly positive and strictly less than TotalDuration
		allowedDur := time.Duration(res.AllowedError)
		if allowedDur <= 0 || allowedDur >= totalWin {
			t.Logf("allowedDur %v not in (0, %v)", allowedDur, totalWin)
			return false
		}

		// Check: AllowedError + ExpectedUptime equals totalWin within 1 second
		uptime := time.Duration(math.Round(float64(totalWin) * (slo / 100.0)))
		diff := math.Abs(float64(totalWin - (allowedDur + uptime)))
		if diff > float64(time.Second) {
			t.Logf("sum mismatch: total=%v, allowed=%v, uptime=%v, diff=%v", totalWin, allowedDur, uptime, diff)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("quick check failed: %v", err)
	}
}

func TestProperty_ErrorBudgetMonotonicity(t *testing.T) {
	svc := service.NewCalculatorService()

	f := func(slo1 float64, slo2 float64, windowSecs int64) bool {
		s1 := math.Mod(math.Abs(slo1), 99.9) + 0.05
		s2 := math.Mod(math.Abs(slo2), 99.9) + 0.05
		if s1 == s2 {
			return true
		}
		if s1 > s2 {
			s1, s2 = s2, s1 // s1 < s2
		}

		secs := math.Abs(float64(windowSecs))
		if secs < 3600 {
			secs = 3600
		}
		if secs > 365*24*3600 {
			secs = 365 * 24 * 3600
		}
		totalWin := util.Duration(time.Duration(secs) * time.Second)

		sloVO1, _ := common.NewSLOTarget(s1)
		sloVO2, _ := common.NewSLOTarget(s2)

		res1, err1 := svc.CalculateBudget(errorbudgetDomain.CalculationParams{TargetSLO: sloVO1, TimeWindow: totalWin})
		res2, err2 := svc.CalculateBudget(errorbudgetDomain.CalculationParams{TargetSLO: sloVO2, TimeWindow: totalWin})
		if err1 != nil || err2 != nil {
			return false
		}

		// Higher SLO (s2 > s1) must strictly have less or equal allowed error downtime
		return time.Duration(res2.AllowedError) <= time.Duration(res1.AllowedError)
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("quick check failed: %v", err)
	}
}
