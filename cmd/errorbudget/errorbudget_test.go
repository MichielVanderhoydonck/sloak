package errorbudget_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/errorbudget"
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	errorbudgetDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/errorbudget"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
)

type mockCalculatorService struct {
	MockResult errorbudgetDomain.BudgetResult
	MockError  error
}

func (m *mockCalculatorService) CalculateBudget(params errorbudgetDomain.CalculationParams) (errorbudgetDomain.BudgetResult, error) {
	return m.MockResult, m.MockError
}

func TestErrorBudgetCommand(t *testing.T) {
    slo99_95, _ := common.NewSLOTarget(99.95)
    mockResult := errorbudgetDomain.BudgetResult{
        TargetSLO:     slo99_95,
        TotalDuration: 30 * 24 * time.Hour,
        AllowedError:  (21 * time.Minute) + (36 * time.Second),
        ErrorBudget:   0.05,
    }

    mockSvc := &mockCalculatorService{
        MockResult: mockResult,
        MockError:  nil,
    }
    errorbudget.SetService(mockSvc)

    output, restoreStdout := testutil.CaptureOutput(t)

    cmd := errorbudget.NewErrorBudgetCmd() 
    cmd.SetArgs([]string{
        "--slo=99.95",
        "--window=30d",
    })

    cmd.Execute()

    restoreStdout() 

    outStr := output.String()
    t.Log(outStr)

    if !strings.Contains(outStr, "Allowed Downtime: 21m36s") {
        t.Error("Output string did not contain the expected allowed downtime")
    }
    if !strings.Contains(outStr, "Error Budget: 0.05000%") {
        t.Error("Output string did not contain the expected error budget percentage")
    }
}