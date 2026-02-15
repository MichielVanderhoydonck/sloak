package feasibility_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/feasibility"
	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/feasibility"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type mockFeasibilityService struct {
	res domain.FeasibilityResult
	err error
}

func (m *mockFeasibilityService) CalculateFeasibility(params domain.FeasibilityParams) (domain.FeasibilityResult, error) {
	return m.res, m.err
}

func TestFeasibilityCommand(t *testing.T) {
	targetSLO, _ := common.NewSLOTarget(99.9)
	mockRes := domain.FeasibilityResult{
		TargetSLO:           targetSLO,
		MTTR:                util.Duration(30 * time.Minute),
		IncidentsPerYear:    17.5,
		IncidentsPerQuarter: 4.4,
		IncidentsPerMonth:   1.5,
		RequiredMTBF:        util.Duration(500 * time.Hour),
	}

	svc := &mockFeasibilityService{res: mockRes}
	feasibility.SetService(svc)

	output, restore := testutil.CaptureOutput(t)

	cmd := feasibility.NewFeasibilityCmd()
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--mttr=30m",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	restore()

	outStr := output.String()

	if !strings.Contains(outStr, "SLO Feasibility Analysis") {
		t.Error("Output missing header")
	}
	if !strings.Contains(outStr, "Target SLO:   99.900%") {
		t.Error("Output missing parsed SLO")
	}

	if !strings.Contains(outStr, "17.5 incidents") {
		t.Error("Output missing Incidents Per Year")
	}
	if !strings.Contains(outStr, "20.8d") {
		t.Error("Output missing MTBF duration")
	}

	if !strings.Contains(outStr, "REALISTIC") {
		t.Errorf("Expected status REALISTIC, got output:\n%s", outStr)
	}
}
