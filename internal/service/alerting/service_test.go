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

func TestGenerateMatrix_CanonicalMonthlyWindows(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo, err := common.NewSLOTarget(99.9)
	if err != nil {
		t.Fatalf("unexpected error creating SLO target: %v", err)
	}

	windows := []struct {
		name string
		dur  time.Duration
	}{
		{"28d (4w)", 28 * 24 * time.Hour},
		{"29d", 29 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"31d", 31 * 24 * time.Hour},
	}

	for _, w := range windows {
		t.Run(w.name, func(t *testing.T) {
			res, err := svc.GenerateTable(domain.GenerateParams{
				TargetSLO:   slo,
				TotalWindow: w.dur,
			})
			if err != nil {
				t.Fatalf("GenerateTable failed for canonical window %s: %v", w.name, err)
			}
			if len(res.Alerts) != 3 {
				t.Fatalf("expected 3 alerts, got %d", len(res.Alerts))
			}

			// Verify canonical windows and burn rates
			// Page
			if res.Alerts[0].BurnRate != 14.4 || time.Duration(res.Alerts[0].LongWindow) != time.Hour || time.Duration(res.Alerts[0].ShortWindow) != 5*time.Minute {
				t.Errorf("unexpected Page alert for %s: %+v", w.name, res.Alerts[0])
			}
			// Message
			if res.Alerts[1].BurnRate != 6.0 || time.Duration(res.Alerts[1].LongWindow) != 6*time.Hour || time.Duration(res.Alerts[1].ShortWindow) != 30*time.Minute {
				t.Errorf("unexpected Message alert for %s: %+v", w.name, res.Alerts[1])
			}
			// Ticket
			if res.Alerts[2].BurnRate != 1.0 || time.Duration(res.Alerts[2].LongWindow) != 72*time.Hour || time.Duration(res.Alerts[2].ShortWindow) != 6*time.Hour {
				t.Errorf("unexpected Ticket alert for %s: %+v", w.name, res.Alerts[2])
			}
		})
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
			errContains: "strictly between 0.0 and 100.0",
		},
		{
			name:        "SLO 0 percent",
			slo:         common.SLOTarget{Value: 0.0},
			window:      30 * 24 * time.Hour,
			errContains: "strictly between 0.0 and 100.0",
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
			expectedExpr := "expr: sum by (job) (rate(http_requests_total{job=\"api\",code=~\"5..\"}[" + w + "])) / sum by (job) (rate(http_requests_total{job=\"api\"}[" + w + "]))"
			if !strings.Contains(out, expectedExpr) {
				t.Errorf("missing expected recording expr for %s: expected %q in output", w, expectedExpr)
			}
		}
		if !strings.Contains(out, "name: slo-alerts") {
			t.Errorf("expected slo-alerts group alongside recording rules, got:\n%s", out)
		}
	})

	t.Run("prometheus with regex label matcher containing brackets", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":  "api_errors",
			"errors_query": `http_requests_total{code=~"[45].."}`,
			"total_query":  `http_requests_total{status=~"5[0-9]{2}"}`,
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		expectedExpr := `expr: sum by (job) (rate(http_requests_total{code=~"[45].."}[5m])) / sum by (job) (rate(http_requests_total{status=~"5[0-9]{2}"}[5m]))`
		if !strings.Contains(out, expectedExpr) {
			t.Errorf("expected regex character classes to be wrapped with rate, got:\n%s", out)
		}
	})

	t.Run("prometheus with custom aggregation_labels list", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":        "api_errors",
			"errors_query":       `http_requests_total{code=~"5.."}`,
			"total_query":        `http_requests_total`,
			"aggregation_labels": []string{"job", "service"},
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		expectedExpr := `expr: sum by (job, service) (rate(http_requests_total{code=~"5.."}[5m])) / sum by (job, service) (rate(http_requests_total[5m]))`
		if !strings.Contains(out, expectedExpr) {
			t.Errorf("expected sum by (job, service), got:\n%s", out)
		}
	})

	t.Run("prometheus with empty aggregation_labels disables by clause", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":        "api_errors",
			"errors_query":       `http_requests_total{code=~"5.."}`,
			"total_query":        `http_requests_total`,
			"aggregation_labels": "",
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		expectedExpr := `expr: sum(rate(http_requests_total{code=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`
		if !strings.Contains(out, expectedExpr) {
			t.Errorf("expected bare sum(rate(...)), got:\n%s", out)
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

	t.Run("reject query with hardcoded range without placeholder", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":  "api_errors",
			"errors_query": "sum(rate(http_requests_total[5m]))",
			"total_query":  "sum(rate(http_requests_total[5m]))",
		}

		_, err := svc.RenderTemplate(params, config, promTpl)
		if err == nil {
			t.Fatal("expected error for hardcoded range selector without placeholder, got nil")
		}
		if !strings.Contains(err.Error(), "contains a fixed range selector without a window placeholder") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("reject errors_query without total_query", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":  "api_errors",
			"errors_query": "http_requests_total{code=~\"5..\"}",
		}

		_, err := svc.RenderTemplate(params, config, promTpl)
		if err == nil {
			t.Fatal("expected error for errors_query without total_query, got nil")
		}
		if !strings.Contains(err.Error(), "errors_query requires total_query to compute the error ratio") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("reject total_query without errors_query", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name": "api_errors",
			"total_query": "http_requests_total",
		}

		_, err := svc.RenderTemplate(params, config, promTpl)
		if err == nil {
			t.Fatal("expected error for total_query without errors_query, got nil")
		}
		if !strings.Contains(err.Error(), "total_query provided without errors_query") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("format param overrides config format", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"format": "json", // should be ignored in favor of params.Format
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}
		if strings.HasPrefix(strings.TrimSpace(out), "{") {
			t.Errorf("expected YAML output because params.Format is yaml, but got JSON:\n%s", out)
		}
	})
}

func TestRenderErrorRatioQuery(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, _ := common.NewSLOTarget(99.9)

	promTpl, err := templates.GetTemplate("prometheus-operator")
	if err != nil {
		t.Fatalf("failed to get prometheus-operator template: %v", err)
	}

	t.Run("auto-wraps bare ratio gauge with avg_over_time and default job grouping", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":       "checkout_errors",
			"error_ratio_query": "slo:error_ratio",
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		expected := "expr: avg by (job) (avg_over_time(slo:error_ratio[5m]))"
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in output, got:\n%s", expected, out)
		}
	})

	t.Run("auto-wraps bare ratio gauge with custom aggregation labels", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":        "checkout_errors",
			"error_ratio_query":  "slo:error_ratio",
			"aggregation_labels": "service, env",
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		expected := "expr: avg by (service, env) (avg_over_time(slo:error_ratio[5m]))"
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in output, got:\n%s", expected, out)
		}
	})

	t.Run("auto-wraps bare ratio gauge with empty aggregation labels", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":        "checkout_errors",
			"error_ratio_query":  "slo:error_ratio",
			"aggregation_labels": "",
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		expected := "expr: avg_over_time(slo:error_ratio[5m])"
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in output, got:\n%s", expected, out)
		}
	})

	t.Run("preserves placeholder in error_ratio_query", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":       "checkout_errors",
			"error_ratio_query": "rate(errors[$window]) / rate(total[$window])",
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		expected := "expr: rate(errors[5m]) / rate(total[5m])"
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in output, got:\n%s", expected, out)
		}
	})

	t.Run("sli_query alias works identically to error_ratio_query", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name": "checkout_errors",
			"sli_query":   "slo:error_ratio",
		}

		out, err := svc.RenderTemplate(params, config, promTpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		expected := "expr: avg by (job) (avg_over_time(slo:error_ratio[5m]))"
		if !strings.Contains(out, expected) {
			t.Errorf("expected %q in output, got:\n%s", expected, out)
		}
	})

	t.Run("rejects combining error_ratio_query with counter queries", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
		}
		config := map[string]any{
			"metric_name":       "checkout_errors",
			"error_ratio_query": "slo:error_ratio",
			"errors_query":      "http_requests_total{code=~\"5..\"}",
			"total_query":       "http_requests_total",
		}

		_, err := svc.RenderTemplate(params, config, promTpl)
		if err == nil {
			t.Fatal("expected error combining error_ratio_query with errors_query, got nil")
		}
		if !strings.Contains(err.Error(), "error_ratio_query cannot be combined with errors_query or total_query") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestPromQLQueryValidation(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, _ := common.NewSLOTarget(99.9)

	promTpl, err := templates.GetTemplate("prometheus-operator")
	if err != nil {
		t.Fatalf("failed to get prometheus-operator template: %v", err)
	}

	testCases := []struct {
		name        string
		config      map[string]any
		errContains string
	}{
		{
			name: "rejects complex aggregate expression without placeholder",
			config: map[string]any{
				"metric_name":  "api_errors",
				"errors_query": "sum(http_errors_total)",
				"total_query":  "http_requests_total",
			},
			errContains: "complex PromQL expression (*parser.AggregateExpr)",
		},
		{
			name: "rejects binary operator expression without placeholder",
			config: map[string]any{
				"metric_name":  "api_errors",
				"errors_query": "a_total or b_total",
				"total_query":  "http_requests_total",
			},
			errContains: "complex PromQL expression (*parser.BinaryExpr)",
		},
		{
			name: "rejects compound fixed range selector x[1h30m]",
			config: map[string]any{
				"metric_name":  "api_errors",
				"errors_query": "x[1h30m]",
				"total_query":  "http_requests_total",
			},
			errContains: "contains a fixed range selector without a window placeholder",
		},
		{
			name: "rejects subquery selector x[1h:5m]",
			config: map[string]any{
				"metric_name":  "api_errors",
				"errors_query": "x[1h:5m]",
				"total_query":  "http_requests_total",
			},
			errContains: "contains a fixed range selector without a window placeholder",
		},
		{
			name: "rejects invalid PromQL syntax",
			config: map[string]any{
				"metric_name":  "api_errors",
				"errors_query": "http_requests_total{{broken",
				"total_query":  "http_requests_total",
			},
			errContains: "is not valid PromQL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := domain.GenerateParams{
				TargetSLO:   slo999,
				TotalWindow: 30 * 24 * time.Hour,
				Format:      "yaml",
			}
			_, err := svc.RenderTemplate(params, tc.config, promTpl)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.errContains)
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Fatalf("expected error containing %q, got %v", tc.errContains, err)
			}
		})
	}
}

func TestAggregationLabelsValidation(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, _ := common.NewSLOTarget(99.9)

	promTpl, err := templates.GetTemplate("prometheus-operator")
	if err != nil {
		t.Fatalf("failed to get prometheus-operator template: %v", err)
	}

	invalidLabels := []string{
		"job;service",
		"job service",
		"job-name",
		"123abc",
		"service.name",
	}

	for _, label := range invalidLabels {
		t.Run("rejects invalid label "+label, func(t *testing.T) {
			params := domain.GenerateParams{
				TargetSLO:   slo999,
				TotalWindow: 30 * 24 * time.Hour,
				Format:      "yaml",
			}
			config := map[string]any{
				"metric_name":        "api_errors",
				"errors_query":       "http_requests_total{code=~\"5..\"}",
				"total_query":        "http_requests_total",
				"aggregation_labels": label,
			}
			_, err := svc.RenderTemplate(params, config, promTpl)
			if err == nil {
				t.Fatalf("expected error for invalid label %q, got nil", label)
			}
			if !strings.Contains(err.Error(), "invalid aggregation label") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})
	}
}

func TestValidateDatadogConfig(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, _ := common.NewSLOTarget(99.9)

	ddTpl, err := templates.GetTemplate("datadog")
	if err != nil {
		t.Fatalf("failed to get datadog template: %v", err)
	}

	t.Run("rejects invalid timeframe", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "json",
			Template:    "datadog",
		}
		config := map[string]any{
			"timeframe": "14d",
		}
		_, err := svc.RenderTemplate(params, config, ddTpl)
		if err == nil {
			t.Fatal("expected error for invalid datadog timeframe 14d, got nil")
		}
		if !strings.Contains(err.Error(), `datadog SLO timeframe "14d" is invalid`) {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("rejects unsupported window without timeframe override", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 14 * 24 * time.Hour,
			Format:      "json",
			Template:    "datadog",
		}
		config := map[string]any{}
		_, err := svc.RenderTemplate(params, config, ddTpl)
		if err == nil {
			t.Fatal("expected error for unsupported window 14d, got nil")
		}
		if !strings.Contains(err.Error(), "datadog SLOs only support 7d, 30d, or 90d timeframes") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("rejects unsupported window when template path is ./templates/datadog.cue", func(t *testing.T) {
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 14 * 24 * time.Hour,
			Format:      "json",
			Template:    "./templates/datadog.cue",
		}
		config := map[string]any{}
		_, err := svc.RenderTemplate(params, config, ddTpl)
		if err == nil {
			t.Fatal("expected error for unsupported window 14d via file path, got nil")
		}
		if !strings.Contains(err.Error(), "datadog SLOs only support 7d, 30d, or 90d timeframes") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("allows custom template mentioning datadog with 28d window without false positive", func(t *testing.T) {
		customTpl := `package templates
// Migrated from datadog monitoring system
data: #Input
rule: {
	labels: {
		source: "datadog-migrated"
	}
	expr: "sum(rate(http_requests_total[5m]))"
}
`
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 28 * 24 * time.Hour,
			Format:      "yaml",
			Template:    "./my-custom.cue",
		}
		config := map[string]any{}
		out, err := svc.RenderTemplate(params, config, customTpl)
		if err != nil {
			t.Fatalf("expected custom template mentioning datadog to succeed with 28d window, got error: %v", err)
		}
		if !strings.Contains(out, "datadog-migrated") {
			t.Fatalf("expected output to contain datadog-migrated label, got: %s", out)
		}
	})
}

func TestRenderedPromQLValidation_JSONAndNested(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, _ := common.NewSLOTarget(99.9)

	t.Run("rejects invalid PromQL expression in JSON output format", func(t *testing.T) {
		tpl := `package templates
data: #Input
rule: {
	expr: "sum(rate(http_requests_total[5m]" // missing closing parenthesis
}
`
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "json",
			Template:    "custom.cue",
		}
		_, err := svc.RenderTemplate(params, map[string]any{}, tpl)
		if err == nil {
			t.Fatal("expected error for invalid PromQL in JSON format, got nil")
		}
		if !strings.Contains(err.Error(), "rendered invalid PromQL expression") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts valid PromQL expression in JSON output format", func(t *testing.T) {
		tpl := `package templates
data: #Input
rule: {
	expr: "sum(rate(http_requests_total[5m]))"
}
`
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "json",
			Template:    "custom.cue",
		}
		out, err := svc.RenderTemplate(params, map[string]any{}, tpl)
		if err != nil {
			t.Fatalf("expected valid PromQL in JSON format to succeed, got error: %v", err)
		}
		if !strings.Contains(out, "http_requests_total") {
			t.Fatalf("unexpected output: %s", out)
		}
	})
}

func TestValidateQueryConfig_NoMapMutation(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, _ := common.NewSLOTarget(99.9)
	tpl, err := templates.GetTemplate("prometheus-operator")
	if err != nil {
		t.Fatalf("failed to get template: %v", err)
	}

	t.Run("sli_query does not mutate caller map with error_ratio_query", func(t *testing.T) {
		cfg := map[string]any{
			"sli_query": "my_ratio_metric",
		}
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
			Template:    "prometheus-operator",
		}
		_, err := svc.RenderTemplate(params, cfg, tpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}
		if _, ok := cfg["error_ratio_query"]; ok {
			t.Errorf("caller's config map was unexpectedly mutated with error_ratio_query")
		}
		if len(cfg) != 1 {
			t.Errorf("expected caller map to have 1 key, got %d", len(cfg))
		}
	})

	t.Run("error_ratio_query does not mutate caller map with sli_query", func(t *testing.T) {
		cfg := map[string]any{
			"error_ratio_query": "my_ratio_metric",
		}
		params := domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
			Format:      "yaml",
			Template:    "prometheus-operator",
		}
		_, err := svc.RenderTemplate(params, cfg, tpl)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}
		if _, ok := cfg["sli_query"]; ok {
			t.Errorf("caller's config map was unexpectedly mutated with sli_query")
		}
		if len(cfg) != 1 {
			t.Errorf("expected caller map to have 1 key, got %d", len(cfg))
		}
	})
}

func TestConsumptionTargetFractions(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, _ := common.NewSLOTarget(99.9)

	t.Run("28d exact consumption targets", func(t *testing.T) {
		res, err := svc.GenerateTable(domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 28 * 24 * time.Hour,
		})
		if err != nil {
			t.Fatalf("GenerateTable failed: %v", err)
		}
		if len(res.Alerts) != 3 {
			t.Fatalf("expected 3 alerts, got %d", len(res.Alerts))
		}
		// Page: 14.4 * 1h / 672h * 100 = 2.14%
		if res.Alerts[0].ConsumptionTarget != 2.14 {
			t.Errorf("expected 2.14%% for 28d Page, got %v", res.Alerts[0].ConsumptionTarget)
		}
		// Message: 6.0 * 6h / 672h * 100 = 5.36%
		if res.Alerts[1].ConsumptionTarget != 5.36 {
			t.Errorf("expected 5.36%% for 28d Message, got %v", res.Alerts[1].ConsumptionTarget)
		}
		// Ticket: 1.0 * 72h / 672h * 100 = 10.71%
		if res.Alerts[2].ConsumptionTarget != 10.71 {
			t.Errorf("expected 10.71%% for 28d Ticket, got %v", res.Alerts[2].ConsumptionTarget)
		}
	})

	t.Run("30d exact consumption targets", func(t *testing.T) {
		res, err := svc.GenerateTable(domain.GenerateParams{
			TargetSLO:   slo999,
			TotalWindow: 30 * 24 * time.Hour,
		})
		if err != nil {
			t.Fatalf("GenerateTable failed: %v", err)
		}
		if res.Alerts[0].ConsumptionTarget != 2.0 {
			t.Errorf("expected 2.0%% for 30d Page, got %v", res.Alerts[0].ConsumptionTarget)
		}
		if res.Alerts[1].ConsumptionTarget != 5.0 {
			t.Errorf("expected 5.0%% for 30d Message, got %v", res.Alerts[1].ConsumptionTarget)
		}
		if res.Alerts[2].ConsumptionTarget != 10.0 {
			t.Errorf("expected 10.0%% for 30d Ticket, got %v", res.Alerts[2].ConsumptionTarget)
		}
	})
}
