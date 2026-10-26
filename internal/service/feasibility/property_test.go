package feasibility_test

import (
	"math"
	"math/rand"
	"testing"
	"testing/quick"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/feasibility"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/feasibility"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func TestProperty_FeasibilityInvariants(t *testing.T) {
	svc := service.NewFeasibilityService()

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		sloVal := r.Float64()*99.0 + 0.5 // 0.5% to 99.5%
		sloVO, err := common.NewSLOTarget(sloVal)
		if err != nil {
			return false
		}

		mttrSecs := r.Float64()*7140 + 60 // 1 minute to 2 hours
		mttr := time.Duration(mttrSecs) * time.Second

		res, err := svc.CalculateFeasibility(domain.FeasibilityParams{
			TargetSLO: sloVO,
			MTTR:      util.Duration(mttr),
		})
		if err != nil {
			return false
		}

		// Property 1: RequiredMTBF must be strictly positive
		if time.Duration(res.RequiredMTBF) <= 0 {
			t.Logf("RequiredMTBF <= 0: %v", res.RequiredMTBF)
			return false
		}

		// Property 2: IncidentsPerYear must be strictly positive
		if res.IncidentsPerYear <= 0 {
			t.Logf("IncidentsPerYear <= 0: %v", res.IncidentsPerYear)
			return false
		}

		// Property 3: Implied yearly downtime matches budget
		expectedYearlyDowntimeSecs := (365 * 24 * 3600.0) * (1.0 - (sloVal / 100.0))
		actualYearlyDowntimeSecs := res.IncidentsPerYear * mttrSecs

		// Check difference within reasonable rounding (since IncidentsPerYear is rounded to 2 decimals)
		pctDiff := math.Abs(actualYearlyDowntimeSecs-expectedYearlyDowntimeSecs) / expectedYearlyDowntimeSecs
		if pctDiff > 0.05 {
			t.Logf("yearly downtime mismatch: expected=%v, actual=%v, diff=%v%%", expectedYearlyDowntimeSecs, actualYearlyDowntimeSecs, pctDiff*100)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("quick check failed: %v", err)
	}
}

func TestProperty_FeasibilityMTBFMonotonicity(t *testing.T) {
	svc := service.NewFeasibilityService()

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		slo1 := r.Float64()*45.0 + 50.0      // 50.0 to 95.0
		slo2 := slo1 + r.Float64()*4.0 + 0.1 // slo2 > slo1

		sloVO1, _ := common.NewSLOTarget(slo1)
		sloVO2, _ := common.NewSLOTarget(slo2)

		mttr := util.Duration(30 * time.Minute)

		res1, _ := svc.CalculateFeasibility(domain.FeasibilityParams{TargetSLO: sloVO1, MTTR: mttr})
		res2, _ := svc.CalculateFeasibility(domain.FeasibilityParams{TargetSLO: sloVO2, MTTR: mttr})

		// Higher reliability requirement (higher SLO) strictly requires higher MTBF and allows fewer incidents
		if time.Duration(res2.RequiredMTBF) <= time.Duration(res1.RequiredMTBF) {
			t.Logf("MTBF not monotonic: MTBF(slo2=%v)=%v <= MTBF(slo1=%v)=%v", slo2, res2.RequiredMTBF, slo1, res1.RequiredMTBF)
			return false
		}
		if res2.IncidentsPerYear >= res1.IncidentsPerYear {
			t.Logf("Incidents not monotonic: Incidents(slo2=%v)=%v >= Incidents(slo1=%v)=%v", slo2, res2.IncidentsPerYear, slo1, res1.IncidentsPerYear)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 500}); err != nil {
		t.Errorf("quick check failed: %v", err)
	}
}
