package alerting_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/alerting"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
	"github.com/spf13/viper"
)

type mockTableGenerator struct {
	res domain.TableResult
	err error
}

var _ domain.TableGenerator = (*mockTableGenerator)(nil)

func (m *mockTableGenerator) GenerateTable(_ domain.GenerateParams) (domain.TableResult, error) {
	return m.res, m.err
}

func TestAlertRulesCommand(t *testing.T) {
	viper.Reset()
	mockRes := domain.TableResult{
		Alerts: []domain.AlertDefinition{
			{
				ConsumptionTarget: 2.0,
				ShortWindow:       5 * time.Minute,
				LongWindow:        1 * time.Hour,
				BurnRate:          14.4,
				NotificationType:  domain.Page,
			},
			{
				ConsumptionTarget: 5.0,
				ShortWindow:       30 * time.Minute,
				LongWindow:        6 * time.Hour,
				BurnRate:          6.0,
				NotificationType:  domain.Page,
			},
			{
				ConsumptionTarget: 10.0,
				ShortWindow:       6 * time.Hour,
				LongWindow:        72 * time.Hour,
				BurnRate:          1.0,
				NotificationType:  domain.Ticket,
			},
		},
	}

	svc := &mockTableGenerator{res: mockRes}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewAlertTableCmd(svc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
	})

	err := cmd.Execute()
	restore()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := output.String()

	if !strings.Contains(outStr, "NOTIFICATION") {
		t.Error("Output missing table header 'NOTIFICATION'")
	}
	if !strings.Contains(outStr, "2%") || !strings.Contains(outStr, "5m") {
		t.Error("Output missing Fast Burn rule values (2%, 5m)")
	}
	if !strings.Contains(outStr, "Page") {
		t.Error("Output missing 'Page' notification type")
	}
	if !strings.Contains(outStr, "10%") || !strings.Contains(outStr, "6h") {
		t.Error("Output missing Ticket rule values (10%, 6h)")
	}
	if !strings.Contains(outStr, "Ticket") {
		t.Error("Output missing 'Ticket' notification type")
	}
}

func TestAlertRulesCommand_InvalidWindow(t *testing.T) {
	viper.Reset()
	svc := &mockTableGenerator{}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewAlertTableCmd(svc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=invalid_window",
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

func TestAlertRulesCommand_InvalidSLO(t *testing.T) {
	viper.Reset()
	svc := &mockTableGenerator{}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewAlertTableCmd(svc)
	cmd.SetArgs([]string{
		"--slo=150.0",
		"--window=30d",
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

func TestAlertRulesCommand_ServiceError(t *testing.T) {
	viper.Reset()
	svc := &mockTableGenerator{err: errors.New("table generation failed")}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewAlertTableCmd(svc)
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for service error, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}
