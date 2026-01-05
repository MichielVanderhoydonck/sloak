package feasibility_test

import (
	"math"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/feasibility"
	service "github.com/MichielVanderhoydonck/sloak/internal/core/service/feasibility"
)

func TestCalculateFeasibility(t *testing.T) {
	svc := service.NewFeasibilityService()
	slo999, _ := common.NewSLOTarget(99.9)

	params := domain.FeasibilityParams{
		TargetSLO: slo999,
		MTTR:      1 * time.Hour,
	}

	res, err := svc.CalculateFeasibility(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IncidentsPerYear < 8.7 || res.IncidentsPerYear > 8.8 {
		t.Errorf("Expected ~8.76 incidents/year, got %f", res.IncidentsPerYear)
	}

	expectedMTBF := 999 * time.Hour
	diff := float64(res.RequiredMTBF - expectedMTBF)
	
	if math.Abs(diff) > float64(time.Second) {
		t.Errorf("Expected MTBF %v, got %v", expectedMTBF, res.RequiredMTBF)
	}
}