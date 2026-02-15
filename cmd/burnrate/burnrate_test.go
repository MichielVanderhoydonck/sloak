package burnrate_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/burnrate"
	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/burnrate"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type mockBurnRateService struct {
	MockResult burnrateDomain.BurnRateResult
	MockError  error
}

func (m *mockBurnRateService) CalculateBurnRate(params burnrateDomain.CalculationParams) (burnrateDomain.BurnRateResult, error) {
	return m.MockResult, m.MockError
}

func TestBurnRateCommand(t *testing.T) {
	mockResult := burnrateDomain.BurnRateResult{
		TotalErrorBudget: util.Duration((43 * time.Minute) + (12 * time.Second)),
		BudgetConsumed:   69.44,
		BurnRate:         2.98,
		BudgetRemaining:  util.Duration((13 * time.Minute) + (12 * time.Second)),
		TimeToExhaustion: util.Duration(73*time.Hour + 55*time.Minute + 12*time.Second),
	}

	mockSvc := &mockBurnRateService{
		MockResult: mockResult,
		MockError:  nil,
	}

	burnrate.SetService(mockSvc)

	output, restoreStdout := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd()
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=30m",
	})

	cmd.Execute()

	restoreStdout()

	outStr := output.String()
	t.Log(outStr)

	if !strings.Contains(outStr, "Burn Rate: 2.98x") {
		t.Error("Output string did not contain the expected burn rate")
	}
	if !strings.Contains(outStr, "Budget Remaining: 13m12s") {
		t.Error("Output string did not contain the expected remaining budget")
	}
	if !strings.Contains(outStr, "Status: CRITICAL!") {
		t.Error("Output string did not contain the expected CRITICAL status")
	}
	if !strings.Contains(outStr, "Forecast: Budget will be empty in") {
		t.Error("Output string did not contain the Forecast prediction")
	}
}
