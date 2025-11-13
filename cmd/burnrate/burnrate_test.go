package burnrate_test 

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/burnrate"
	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/burnrate"
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
		TotalErrorBudget: (43 * time.Minute) + (12 * time.Second),
		BudgetConsumed:   69.44,
		BurnRate:         2.98,
		BudgetRemaining:  (13 * time.Minute) + (12 * time.Second),
	}

	mockSvc := &mockBurnRateService{
		MockResult: mockResult,
		MockError:  nil,
	}

	burnrate.SetService(mockSvc)

	output, restoreStdout := captureOutput(t)

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
}

func captureOutput(t *testing.T) (output *bytes.Buffer, restore func()) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	output = new(bytes.Buffer)
	done := make(chan struct{})
	go func() {
		io.Copy(output, r)
		close(done) 
	}()
	restore = func() {
		w.Close()
		<-done 
		os.Stdout = old
	}
	return output, restore
}