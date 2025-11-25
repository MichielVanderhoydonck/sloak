package dependency_test

import (
	"strings"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/cmd/dependency"
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/dependency"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
)

type mockService struct {
	res domain.Result
	err error
}

func (m *mockService) CalculateCompositeAvailability(params domain.CalculationParams) (domain.Result, error) {
	return m.res, m.err
}

func TestDependencyCommand(t *testing.T) {
	mockRes := domain.Result{
		TotalAvailability: 99.8001,
		CalculationType:   domain.Serial,
		ComponentCount:    2,
	}

	svc := &mockService{res: mockRes, err: nil}
	dependency.SetService(svc)

	output, restore := testutil.CaptureOutput(t)

	cmd := dependency.NewDependencyCmd()
	cmd.SetArgs([]string{"--components=99.9,99.9", "--type=serial"})

	cmd.Execute()

	restore()

	outStr := output.String()
	if !strings.Contains(outStr, "Total Availability: 99.800100%") {
		t.Errorf("Unexpected output: %s", outStr)
	}
}
