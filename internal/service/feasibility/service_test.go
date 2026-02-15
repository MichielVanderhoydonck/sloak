package feasibility_test

import (
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/feasibility"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/feasibility"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func TestCalculateFeasibility(t *testing.T) {
	svc := service.NewFeasibilityService()
	slo999, _ := common.NewSLOTarget(99.9)

	params := domain.FeasibilityParams{
		TargetSLO: slo999,
		MTTR:      util.Duration(1 * time.Hour),
	}

	res, err := svc.CalculateFeasibility(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IncidentsPerYear < 8.7 || res.IncidentsPerYear > 8.8 {
		t.Errorf("Expected ~8.76 incidents/year, got %f", res.IncidentsPerYear)
	}

	expectedMTBF := "41.7d"
	if res.RequiredMTBF.String() != expectedMTBF {
		t.Errorf("Expected MTBF %s, got %s", expectedMTBF, res.RequiredMTBF.String())
	}
}
