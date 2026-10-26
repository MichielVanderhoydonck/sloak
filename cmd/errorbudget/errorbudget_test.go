package errorbudget_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/errorbudget"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	errorbudgetDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/errorbudget"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
	"github.com/spf13/viper"
)

type mockCalculatorService struct {
	MockResult errorbudgetDomain.BudgetResult
	MockError  error
}

func (m *mockCalculatorService) CalculateBudget(_ errorbudgetDomain.CalculationParams) (errorbudgetDomain.BudgetResult, error) {
	return m.MockResult, m.MockError
}

func TestErrorBudgetCommand(t *testing.T) {
	t.Parallel()
	viper.Reset()
	slo99_95, _ := common.NewSLOTarget(99.95)
	mockResult := errorbudgetDomain.BudgetResult{
		TargetSLO:     slo99_95,
		TotalDuration: util.Duration(30 * 24 * time.Hour),
		AllowedError:  util.Duration((21 * time.Minute) + (36 * time.Second)),
		ErrorBudget:   0.05,
	}

	mockSvc := &mockCalculatorService{
		MockResult: mockResult,
		MockError:  nil,
	}

	var output bytes.Buffer
	cmd := errorbudget.NewErrorBudgetCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.95",
		"--window=30d",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := output.String()
	t.Log(outStr)

	if !strings.Contains(outStr, "Allowed Downtime: 21m36s") {
		t.Error("Output string did not contain the expected allowed downtime")
	}
	if !strings.Contains(outStr, "Error Budget: 0.0500% of time") {
		t.Error("Output string did not contain the expected error budget percentage")
	}
}

func TestErrorBudgetCommand_InvalidWindow(t *testing.T) {
	t.Parallel()
	viper.Reset()
	mockSvc := &mockCalculatorService{}

	var output bytes.Buffer
	cmd := errorbudget.NewErrorBudgetCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.95",
		"--window=bad_window",
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

func TestErrorBudgetCommand_InvalidSLO(t *testing.T) {
	t.Parallel()
	viper.Reset()
	mockSvc := &mockCalculatorService{}

	var output bytes.Buffer
	cmd := errorbudget.NewErrorBudgetCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=-1.0",
		"--window=30d",
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

func TestErrorBudgetCommand_ServiceError(t *testing.T) {
	t.Parallel()
	viper.Reset()
	mockSvc := &mockCalculatorService{
		MockError: errors.New("budget calculation failure"),
	}

	var output bytes.Buffer
	cmd := errorbudget.NewErrorBudgetCmd(mockSvc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.95",
		"--window=30d",
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
