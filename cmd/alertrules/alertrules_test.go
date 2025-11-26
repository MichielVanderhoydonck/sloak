package alertrules_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/alertrules"
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
)

type mockService struct {
	res domain.Result
	err error
}

func (m *mockService) GenerateThresholds(params domain.GenerateParams) (domain.Result, error) {
	return m.res, m.err
}

func TestAlertRulesCommand(t *testing.T) {
	mockRes := domain.Result{
		PageRule: domain.AlertRule{
			Severity:       domain.SeverityPage,
			BurnRate:       30.0,
			RecallWindow:   28*time.Minute + 48*time.Second,
			BudgetConsumed: 2.0,
		},
		TicketRule: domain.AlertRule{
			Severity:       domain.SeverityTicket,
			BurnRate:       10.0,
			RecallWindow:   3*time.Hour + 36*time.Minute,
			BudgetConsumed: 5.0,
		},
	}

	svc := &mockService{res: mockRes}
	alertrules.SetService(svc)

	output, restore := testutil.CaptureOutput(t)

	cmd := alertrules.NewAlertRulesCmd()
	cmd.SetArgs([]string{
		"--slo=99.9",
		"--window=30d",
		"--page-tte=24h",
	})

	cmd.Execute()
	restore()

	outStr := output.String()

	if !strings.Contains(outStr, "--- CRITICAL / Paging ---") {
		t.Error("Output missing Paging section header")
	}
	if !strings.Contains(outStr, "Burn Rate Threshold: 30.00x") {
		t.Error("Output missing Page burn rate")
	}
	if !strings.Contains(outStr, "Observation Window:  28m48s") {
		t.Error("Output missing Page window")
	}
	if !strings.Contains(outStr, "--- WARNING / Ticket ---") {
		t.Error("Output missing Ticket section header")
	}
}
