package alerting_test

import (
	"math"
	"strings"
	"testing"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/alerting"
	templates "github.com/MichielVanderhoydonck/sloak/templates"
)

func TestGenerateMatrix(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo, _ := common.NewSLOTarget(99.9)

	params := domain.GenerateParams{
		TargetSLO:   slo,
		TotalWindow: 30 * 24 * time.Hour,
	}

	res, err := svc.GenerateTable(params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(res.Alerts) != 3 {
		t.Fatalf("Expected 3 alert rules, got %d", len(res.Alerts))
	}

	assertFloat := func(name string, expected, actual float64) {
		if math.Abs(expected-actual) > 0.01 {
			t.Errorf("[%s] Expected %.2f, got %.2f", name, expected, actual)
		}
	}

	r1 := res.Alerts[0]
	assertFloat("R1 BurnRate", 14.4, r1.BurnRate)
	if r1.NotificationType != domain.Page {
		t.Errorf("R1 expected Page, got %s", r1.NotificationType)
	}
	if time.Duration(r1.LongWindow) != 1*time.Hour || time.Duration(r1.ShortWindow) != 5*time.Minute {
		t.Errorf("R1 expected 1h/5m, got %v/%v", r1.LongWindow, r1.ShortWindow)
	}

	r2 := res.Alerts[1]
	assertFloat("R2 BurnRate", 6.0, r2.BurnRate)
	if r2.NotificationType != domain.Message {
		t.Errorf("R2 expected Message, got %s", r2.NotificationType)
	}
	if time.Duration(r2.LongWindow) != 6*time.Hour || time.Duration(r2.ShortWindow) != 30*time.Minute {
		t.Errorf("R2 expected 6h/30m, got %v/%v", r2.LongWindow, r2.ShortWindow)
	}

	r3 := res.Alerts[2]
	assertFloat("R3 BurnRate", 1.0, r3.BurnRate)
	if r3.NotificationType != domain.Ticket {
		t.Errorf("R3 expected Ticket, got %s", r3.NotificationType)
	}
	if time.Duration(r3.LongWindow) != 72*time.Hour || time.Duration(r3.ShortWindow) != 6*time.Hour {
		t.Errorf("R3 expected 72h/6h, got %v/%v", r3.LongWindow, r3.ShortWindow)
	}
}

func TestGenerateMatrix_ScalesWithWindow(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo, _ := common.NewSLOTarget(99.9)

	// 7-day window
	params7d := domain.GenerateParams{
		TargetSLO:   slo,
		TotalWindow: 7 * 24 * time.Hour,
	}

	res7d, err := svc.GenerateTable(params7d)
	if err != nil {
		t.Fatalf("Unexpected error for 7d window: %v", err)
	}

	assertFloat := func(name string, expected, actual float64) {
		if math.Abs(expected-actual) > 0.01 {
			t.Errorf("[%s] Expected %.2f, got %.2f", name, expected, actual)
		}
	}

	// Page: 14m / 70s, 14.4x
	assertFloat("7d R1 BurnRate", 14.4, res7d.Alerts[0].BurnRate)
	if time.Duration(res7d.Alerts[0].LongWindow) != 14*time.Minute {
		t.Errorf("7d R1 expected LongWindow 14m, got %v", res7d.Alerts[0].LongWindow)
	}

	// Message: 84m / 7m, 6.0x
	assertFloat("7d R2 BurnRate", 6.0, res7d.Alerts[1].BurnRate)
	if time.Duration(res7d.Alerts[1].LongWindow) != 84*time.Minute {
		t.Errorf("7d R2 expected LongWindow 84m, got %v", res7d.Alerts[1].LongWindow)
	}

	// Ticket: 16h48m / 84m, 1.0x
	assertFloat("7d R3 BurnRate", 1.0, res7d.Alerts[2].BurnRate)
	expectedTicketLong := 16*time.Hour + 48*time.Minute
	if time.Duration(res7d.Alerts[2].LongWindow) != expectedTicketLong {
		t.Errorf("7d R3 expected LongWindow %v, got %v", expectedTicketLong, res7d.Alerts[2].LongWindow)
	}
}

func TestGenerateMatrix_Validation(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo99, _ := common.NewSLOTarget(99.9)

	tests := []struct {
		name        string
		slo         common.SLOTarget
		window      time.Duration
		errContains string
	}{
		{
			name:        "zero window",
			slo:         slo99,
			window:      0,
			errContains: "greater than zero",
		},
		{
			name:        "negative window",
			slo:         slo99,
			window:      -1 * time.Hour,
			errContains: "greater than zero",
		},
		{
			name:        "window under 7d",
			slo:         slo99,
			window:      3 * 24 * time.Hour,
			errContains: "at least 7d",
		},
		{
			name:        "SLO 100 percent",
			slo:         common.SLOTarget{Value: 100.0},
			window:      30 * 24 * time.Hour,
			errContains: "strictly between 0 and 100",
		},
		{
			name:        "SLO 0 percent",
			slo:         common.SLOTarget{Value: 0.0},
			window:      30 * 24 * time.Hour,
			errContains: "strictly between 0 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GenerateTable(domain.GenerateParams{
				TargetSLO:   tt.slo,
				TotalWindow: tt.window,
			})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errContains)
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("expected error containing %q, got %v", tt.errContains, err)
			}
		})
	}
}

func TestRenderRecordingRules(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, _ := common.NewSLOTarget(99.9)

	promTpl, err := templates.GetTemplate("prometheus-operator")
	if err != nil {
		t.Fatalf("failed to get prometheus-operator template: %v", err)
	}

	t.Run("prometheus with errors_query and total_query", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":  "api_errors",
			"errors_query": "http_requests_total{job=\"api\",code=~\"5..\"}",
			"total_query":  "http_requests_total{job=\"api\"}",
			"rule_labels": map[string]any{
				"service": "api",
			},
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		if !strings.Contains(out, "name: slo-recording-rules") {
			t.Errorf("expected recording-rules group, got:\n%s", out)
		}
		for _, w := range []string{"5m", "30m", "1h", "6h", "3d"} {
			expectedRecord := "record: job:api_errors:rate" + w
			if !strings.Contains(out, expectedRecord) {
				t.Errorf("missing recording rule for %s: expected %q in output", w, expectedRecord)
			}
			expectedExpr := "expr: sum(rate(http_requests_total{job=\"api\",code=~\"5..\"}[" + w + "])) / sum(rate(http_requests_total{job=\"api\"}[" + w + "]))"
			if !strings.Contains(out, expectedExpr) {
				t.Errorf("missing expected recording expr for %s: expected %q in output", w, expectedExpr)
			}
		}
		if !strings.Contains(out, "name: slo-alerts") {
			t.Errorf("expected slo-alerts group alongside recording rules, got:\n%s", out)
		}
	})

	t.Run("prometheus with vector simulation query", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":  "api_errors",
			"errors_query": "vector(0.2)",
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		if !strings.Contains(out, "record: job:api_errors:rate5m") {
			t.Errorf("expected recording rule for 5m, got:\n%s", out)
		}
		if !strings.Contains(out, "expr: vector(0.2)") {
			t.Errorf("expected expr: vector(0.2), got:\n%s", out)
		}
	})

	t.Run("grafana cloud with sli_query", func(t *testing.T) {
		grafanaTpl, err := templates.GetTemplate("grafana-cloud")
		if err != nil {
			t.Fatalf("failed to get grafana-cloud template: %v", err)
		}

		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name": "checkout_errors",
			"sli_query":   "sum(rate(errors[$window])) / sum(rate(total[$window]))",
		}

		out, err := svc.RenderTemplate(params, config, grafanaTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		if !strings.Contains(out, "name: slo-recording-rules") {
			t.Errorf("expected slo-recording-rules, got:\n%s", out)
		}
		if !strings.Contains(out, "record: job:checkout_errors:rate5m") {
			t.Errorf("expected recording rule for rate5m, got:\n%s", out)
		}
		if !strings.Contains(out, "expr: sum(rate(errors[5m])) / sum(rate(total[5m]))") {
			t.Errorf("expected interpolated expr for 5m, got:\n%s", out)
		}
	})
}
