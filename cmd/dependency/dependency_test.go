package dependency_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/cmd/dependency"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/dependency"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
	"github.com/spf13/viper"
)

type mockService struct {
	res domain.Result
	err error
}

func (m *mockService) CalculateCompositeAvailability(_ domain.CalculationParams) (domain.Result, error) {
	return m.res, m.err
}

func TestDependencyCommand(t *testing.T) {
	viper.Reset()
	mockRes := domain.Result{
		TotalAvailability: 99.8001,
		CalculationType:   domain.Serial,
		ComponentCount:    2,
	}

	svc := &mockService{res: mockRes, err: nil}

	output, restore := testutil.CaptureOutput(t)

	cmd := dependency.NewDependencyCmd(svc)
	cmd.SetArgs([]string{"--components=99.9,99.9", "--type=serial"})

	err := cmd.Execute()
	restore()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Total Availability: 99.8001%") {
		t.Errorf("Unexpected output: %s", outStr)
	}
}

func TestDependencyCommand_MissingComponents(t *testing.T) {
	viper.Reset()
	svc := &mockService{}

	output, restore := testutil.CaptureOutput(t)

	cmd := dependency.NewDependencyCmd(svc)
	cmd.SetArgs([]string{"--type=serial"})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for missing components, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestDependencyCommand_InvalidComponent(t *testing.T) {
	viper.Reset()
	svc := &mockService{}

	output, restore := testutil.CaptureOutput(t)

	cmd := dependency.NewDependencyCmd(svc)
	cmd.SetArgs([]string{"--components=99.9,invalid", "--type=serial"})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for invalid component, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestDependencyCommand_ServiceError(t *testing.T) {
	viper.Reset()
	svc := &mockService{err: errors.New("composite calculation failed")}

	output, restore := testutil.CaptureOutput(t)

	cmd := dependency.NewDependencyCmd(svc)
	cmd.SetArgs([]string{"--components=99.9,99.5", "--type=serial"})

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
