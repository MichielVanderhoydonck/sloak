package burnrate_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/burnrate"
	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/burnrate"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
	"github.com/spf13/viper"
)

type mockBurnRateService struct {
	MockResult burnrateDomain.BurnRateResult
	MockError  error
}

func (m *mockBurnRateService) CalculateBurnRate(_ burnrateDomain.CalculationParams) (burnrateDomain.BurnRateResult, error) {
	return m.MockResult, m.MockError
}

func TestBurnRateCommand(t *testing.T) {
	viper.Reset()
	br := 2.98
	mockResult := burnrateDomain.BurnRateResult{
		TotalErrorBudget: util.Duration((43 * time.Minute) + (12 * time.Second)),
		BudgetConsumed:   69.44,
		BurnRate:         &br,
		BudgetRemaining:  util.Duration((13 * time.Minute) + (12 * time.Second)),
		TimeToExhaustion: util.Duration(73*time.Hour + 55*time.Minute + 12*time.Second),
	}

	mockSvc := &mockBurnRateService{
		MockResult: mockResult,
		MockError:  nil,
	}

	output, restoreStdout := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=30m",
	})

	err := cmd.Execute()
	restoreStdout()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func TestBurnRateCommand_ConsumedWithDayUnit(t *testing.T) {
	viper.Reset()
	br := 1.0
	mockResult := burnrateDomain.BurnRateResult{
		TotalErrorBudget: util.Duration(43 * time.Minute),
		BurnRate:         &br,
	}
	mockSvc := &mockBurnRateService{MockResult: mockResult}

	output, restore := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=1d",
	})

	err := cmd.Execute()
	restore()
	if err != nil {
		t.Fatalf("expected --consumed=1d to parse successfully, got error: %v", err)
	}
	_ = output.String()
}

func TestBurnRateCommand_Unbounded(t *testing.T) {
	viper.Reset()
	mockResult := burnrateDomain.BurnRateResult{
		TotalErrorBudget:  util.Duration(43 * time.Minute),
		BurnRate:          nil,
		BurnRateUnbounded: true,
	}
	mockSvc := &mockBurnRateService{MockResult: mockResult}

	output, restore := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=0s",
		"--consumed=5m",
	})

	err := cmd.Execute()
	restore()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Burn Rate: Unbounded") {
		t.Errorf("expected unbounded burn rate in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Status: CRITICAL!") {
		t.Errorf("expected CRITICAL status, got: %s", outStr)
	}
}

func TestBurnRateCommand_InvalidWindow(t *testing.T) {
	viper.Reset()
	mockSvc := &mockBurnRateService{}

	output, restore := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=invalid",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for invalid window, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestBurnRateCommand_InvalidElapsed(t *testing.T) {
	viper.Reset()
	mockSvc := &mockBurnRateService{}

	output, restore := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=invalid",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for invalid elapsed, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestBurnRateCommand_InvalidConsumed(t *testing.T) {
	viper.Reset()
	mockSvc := &mockBurnRateService{}

	output, restore := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=invalid",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for invalid consumed, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestBurnRateCommand_InvalidSLO(t *testing.T) {
	viper.Reset()
	mockSvc := &mockBurnRateService{}

	output, restore := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetArgs([]string{
		"--slo=120.0",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=30m",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for invalid slo, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestBurnRateCommand_ServiceError(t *testing.T) {
	viper.Reset()
	mockSvc := &mockBurnRateService{
		MockError: errors.New("burn rate calculation failed"),
	}

	output, restore := testutil.CaptureOutput(t)

	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=30m",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error from service error, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}
