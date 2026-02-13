package alerting_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/alerting"
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
)

type mockService struct {
	res domain.TableResult
	err error
}

func (m *mockService) GenerateTable(params domain.GenerateParams) (domain.TableResult, error) {
	return m.res, m.err
}

func (m *mockService) GeneratePrometheus(params domain.GeneratePrometheusParams) (string, error) {
	return "mocked-prometheus-output", nil
}

func TestAlertRulesCommand(t *testing.T) {
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

	svc := &mockService{res: mockRes}
	alerting.SetService(svc)

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewAlertTableCmd()
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
	})

	cmd.Execute()
	restore()

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
