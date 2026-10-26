package disruption_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/disruption"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/disruption"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
	"github.com/spf13/viper"
)

type mockDisruptionService struct {
	res domain.Result
	err error
}

func (m *mockDisruptionService) CalculateCapacity(_ domain.CalculationParams) (domain.Result, error) {
	return m.res, m.err
}

func TestDisruptionCommand_Success(t *testing.T) {
	t.Parallel()
	viper.Reset()
	mockRes := domain.Result{
		TotalErrorBudget: util.Duration(43 * time.Minute),
		MaxDisruptions:   43,
		DailyDisruptions: 1.4,
	}
	svc := &mockDisruptionService{res: mockRes}

	var output bytes.Buffer
	cmd := disruption.NewDisruptionCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--cost=1m",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Disruption Budget Analysis") {
		t.Errorf("output missing header, got: %q", outStr)
	}
	if !strings.Contains(outStr, "Max Events Total: 43") {
		t.Errorf("output missing max events, got: %q", outStr)
	}
}

func TestDisruptionCommand_InvalidWindow(t *testing.T) {
	t.Parallel()
	viper.Reset()
	svc := &mockDisruptionService{}

	var output bytes.Buffer
	cmd := disruption.NewDisruptionCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=invalid_window",
		"--cost=1m",
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

func TestDisruptionCommand_InvalidCost(t *testing.T) {
	t.Parallel()
	viper.Reset()
	svc := &mockDisruptionService{}

	var output bytes.Buffer
	cmd := disruption.NewDisruptionCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--cost=invalid_cost",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid cost, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestDisruptionCommand_InvalidSLO(t *testing.T) {
	t.Parallel()
	viper.Reset()
	svc := &mockDisruptionService{}

	var output bytes.Buffer
	cmd := disruption.NewDisruptionCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=150.0",
		"--window=30d",
		"--cost=1m",
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

func TestDisruptionCommand_ServiceError(t *testing.T) {
	t.Parallel()
	viper.Reset()
	svc := &mockDisruptionService{err: errors.New("capacity calc failed")}

	var output bytes.Buffer
	cmd := disruption.NewDisruptionCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--cost=1m",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for service error, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}
