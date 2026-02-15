package convert_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/convert"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/convert"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
)

type mockService struct {
	res domain.ConversionResult
	err error
}

func (m *mockService) Convert(params domain.ConversionParams) (domain.ConversionResult, error) {
	return m.res, m.err
}

func TestConvertCommand(t *testing.T) {
	mockRes := domain.ConversionResult{
		AvailabilityPercent: 99.9,
		DailyDowntime:       1*time.Minute + 26*time.Second,
		YearlyDowntime:      8*time.Hour + 45*time.Minute,
	}

	svc := &mockService{res: mockRes, err: nil}
	convert.SetService(svc)

	output, restore := testutil.CaptureOutput(t)

	cmd := convert.NewConvertCmd()
	cmd.SetArgs([]string{"--nines=99.9"})

	cmd.Execute()

	restore()

	outStr := output.String()
	if !strings.Contains(outStr, "Availability: 99.90000%") {
		t.Errorf("Missing availability in output: %s", outStr)
	}
	if !strings.Contains(outStr, "Daily Allowed:     1m26s") {
		t.Errorf("Missing daily downtime in output: %s", outStr)
	}
}
