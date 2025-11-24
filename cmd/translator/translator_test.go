package translator_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/translator"
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/translator"
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

	output, restore := captureOutput(t)
	
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