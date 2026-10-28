package burnrate_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/burnrate"
	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/burnrate"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type mockBurnRateService struct {
	MockResult burnrateDomain.BurnRateResult
	MockError  error
}

func (m *mockBurnRateService) CalculateBurnRate(_ burnrateDomain.CalculationParams) (burnrateDomain.BurnRateResult, error) {
	return m.MockResult, m.MockError
}

func TestBurnRateCommand(t *testing.T) {
	t.Parallel()
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

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=30m",
	})

	err := cmd.Execute()
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
	if !strings.Contains(outStr, "Status: WARNING (Ticket)") {
		t.Errorf("Output string did not contain the expected WARNING (Ticket) status, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Forecast: Budget will be empty in") {
		t.Error("Output string did not contain the Forecast prediction")
	}
}

func TestBurnRateCommand_ConsumedWithDayUnit(t *testing.T) {
	t.Parallel()
	br := 1.0
	mockResult := burnrateDomain.BurnRateResult{
		TotalErrorBudget: util.Duration(43 * time.Minute),
		BurnRate:         &br,
	}
	mockSvc := &mockBurnRateService{MockResult: mockResult}

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=1d",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected --consumed=1d to parse successfully, got error: %v", err)
	}
	_ = output.String()
}

func TestBurnRateCommand_Unbounded(t *testing.T) {
	t.Parallel()
	mockResult := burnrateDomain.BurnRateResult{
		TotalErrorBudget:  util.Duration(43 * time.Minute),
		BurnRate:          nil,
		BurnRateUnbounded: true,
	}
	mockSvc := &mockBurnRateService{MockResult: mockResult}

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=0s",
		"--consumed=5m",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Burn Rate: Unbounded") {
		t.Errorf("expected unbounded burn rate in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Status: CRITICAL (Page)!") {
		t.Errorf("expected CRITICAL (Page)! status, got: %s", outStr)
	}
}

func TestBurnRateCommand_HighMessageSeverity(t *testing.T) {
	t.Parallel()
	br := 7.5
	mockResult := burnrateDomain.BurnRateResult{
		TotalErrorBudget: util.Duration(43 * time.Minute),
		BurnRate:         &br,
	}
	mockSvc := &mockBurnRateService{MockResult: mockResult}

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=2h",
		"--consumed=15m",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Status: HIGH (Message)") {
		t.Errorf("expected HIGH (Message) status, got: %s", outStr)
	}
}

func TestBurnRateCommand_InvalidWindow(t *testing.T) {
	t.Parallel()
	mockSvc := &mockBurnRateService{}

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=invalid",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid window, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestBurnRateCommand_InvalidElapsed(t *testing.T) {
	t.Parallel()
	mockSvc := &mockBurnRateService{}

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=invalid",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid elapsed, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestBurnRateCommand_InvalidConsumed(t *testing.T) {
	t.Parallel()
	mockSvc := &mockBurnRateService{}

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=invalid",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid consumed, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestBurnRateCommand_InvalidSLO(t *testing.T) {
	t.Parallel()
	mockSvc := &mockBurnRateService{}

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=120.0",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=30m",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid slo, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestBurnRateCommand_ServiceError(t *testing.T) {
	t.Parallel()
	mockSvc := &mockBurnRateService{
		MockError: errors.New("burn rate calculation failed"),
	}

	var output bytes.Buffer
	cmd := burnrate.NewBurnRateCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=30m",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from service error, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}
