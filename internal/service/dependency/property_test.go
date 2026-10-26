package dependency_test

import (
	"math/rand"
	"testing"
	"testing/quick"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/dependency"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/dependency"
)

func TestProperty_DependencyBounds(t *testing.T) {
	svc := service.NewAvailabilityService()

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		n := r.Intn(7) + 2 // 2 to 8 components
		components := make([]float64, n)
		minC := 100.0
		maxC := 0.0

		for i := 0; i < n; i++ {
			// Random value between 1.0 and 99.99
			val := r.Float64()*98.99 + 1.0
			components[i] = val
			if val < minC {
				minC = val
			}
			if val > maxC {
				maxC = val
			}
		}

		resSerial, err := svc.CalculateCompositeAvailability(domain.CalculationParams{
			Components: components,
			Type:       domain.Serial,
		})
		if err != nil {
			t.Logf("unexpected serial error: %v", err)
			return false
		}

		resParallel, err := svc.CalculateCompositeAvailability(domain.CalculationParams{
			Components: components,
			Type:       domain.Parallel,
		})
		if err != nil {
			t.Logf("unexpected parallel error: %v", err)
			return false
		}

		// Core Invariant: Serial <= Min(Components) <= Max(Components) <= Parallel
		// Allow small floating-point epsilon
		eps := 1e-4
		if resSerial.TotalAvailability > minC+eps {
			t.Logf("serial violation: serial=%v > minC=%v", resSerial.TotalAvailability, minC)
			return false
		}
		if resParallel.TotalAvailability < maxC-eps {
			t.Logf("parallel violation: parallel=%v < maxC=%v", resParallel.TotalAvailability, maxC)
			return false
		}
		if resSerial.TotalAvailability > resParallel.TotalAvailability+eps {
			t.Logf("ordering violation: serial=%v > parallel=%v", resSerial.TotalAvailability, resParallel.TotalAvailability)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("quick check failed: %v", err)
	}
}
