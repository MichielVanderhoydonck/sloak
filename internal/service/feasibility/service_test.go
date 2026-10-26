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

	if res.IncidentsPerYear != 8.76 {
		t.Errorf("Expected 8.76 incidents/year, got %f", res.IncidentsPerYear)
	}

	expectedMTBF := "41.6d"
	if res.RequiredMTBF.String() != expectedMTBF {
		t.Errorf("Expected MTBF %s, got %s", expectedMTBF, res.RequiredMTBF.String())
	}
}

func TestCalculateFeasibility_NinetyPercent(t *testing.T) {
	svc := service.NewFeasibilityService()
	slo90, _ := common.NewSLOTarget(90.0)

	params := domain.FeasibilityParams{
		TargetSLO: slo90,
		MTTR:      util.Duration(30 * time.Minute),
	}

	res, err := svc.CalculateFeasibility(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IncidentsPerYear != 1752.0 {
		t.Errorf("Expected exactly 1752.0 incidents/year, got %f", res.IncidentsPerYear)
	}

	if res.RequiredMTBF.String() != "4h30m0s" {
		t.Errorf("Expected MTBF 4h30m0s, got %s", res.RequiredMTBF.String())
	}
}

func TestCalculateFeasibility_InvalidSLO(t *testing.T) {
	svc := service.NewFeasibilityService()

	// 100% SLO
	_, err := svc.CalculateFeasibility(domain.FeasibilityParams{
		TargetSLO: common.SLOTarget{Value: 100.0},
		MTTR:      util.Duration(30 * time.Minute),
	})
	if err == nil {
		t.Fatal("expected error for 100% SLO, got nil")
	}

	// 0% SLO
	_, err = svc.CalculateFeasibility(domain.FeasibilityParams{
		TargetSLO: common.SLOTarget{Value: 0.0},
		MTTR:      util.Duration(30 * time.Minute),
	})
	if err == nil {
		t.Fatal("expected error for 0% SLO, got nil")
	}
}

func TestCalculateFeasibility_ZeroMTTR(t *testing.T) {
	svc := service.NewFeasibilityService()
	slo999, _ := common.NewSLOTarget(99.9)

	params := domain.FeasibilityParams{
		TargetSLO: slo999,
		MTTR:      0,
	}

	_, err := svc.CalculateFeasibility(params)
	if err == nil {
		t.Fatal("expected error for zero MTTR, got nil")
	}
}
