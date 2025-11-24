package dependency_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/cmd/dependency"
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/dependency"
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

	output, restore := captureOutput(t)
	// No defer here, manual restore
	
	cmd := dependency.NewDependencyCmd()
	cmd.SetArgs([]string{"--components=99.9,99.9", "--type=serial"})
	
	cmd.Execute()
	
	restore() // Restore before assertions

	outStr := output.String()
	if !strings.Contains(outStr, "Total Availability: 99.800100%") {
		t.Errorf("Unexpected output: %s", outStr)
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