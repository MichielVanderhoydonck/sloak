package translator_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/translator"
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/translator"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
)

type mockService struct {
	res domain.TranslationResult
	err error
}

func (m *mockService) Translate(params domain.TranslationParams) (domain.TranslationResult, error) {
	return m.res, m.err
}

func TestTranslatorCommand(t *testing.T) {
	mockRes := domain.TranslationResult{
		AvailabilityPercent: 99.9,
		DailyDowntime:       1*time.Minute + 26*time.Second,
		YearlyDowntime:      8*time.Hour + 45*time.Minute,
	}

	svc := &mockService{res: mockRes, err: nil}
	translator.SetService(svc)

	output, restore := testutil.CaptureOutput(t)
	
	cmd := translator.NewTranslatorCmd()
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