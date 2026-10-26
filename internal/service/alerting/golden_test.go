package alerting_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/alerting"
	templates "github.com/MichielVanderhoydonck/sloak/templates"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestRenderGoldenFiles(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, err := common.NewSLOTarget(99.9)
	if err != nil {
		t.Fatalf("NewSLOTarget failed: %v", err)
	}

	params := domain.GenerateParams{
		TargetSLO:   slo999,
		TotalWindow: 30 * 24 * time.Hour,
	}

	config := map[string]any{
		"metric_name":  "api_errors",
		"service_name": "api-service",
		"namespace":    "monitoring",
		"rule_labels": map[string]any{
			"env":     "prod",
			"service": "api",
		},
		"meta_labels": map[string]any{
			"release": "prometheus",
		},
	}

	testCases := []struct {
		templateName string
		format       string
		goldenFile   string
	}{
		{
			templateName: "prometheus-operator",
			format:       "yaml",
			goldenFile:   "testdata/prometheus_operator.golden",
		},
		{
			templateName: "grafana-cloud",
			format:       "yaml",
			goldenFile:   "testdata/grafana_cloud.golden",
		},
		{
			templateName: "datadog",
			format:       "json",
			goldenFile:   "testdata/datadog.golden",
		},
		{
			templateName: "otel-collector",
			format:       "yaml",
			goldenFile:   "testdata/otel_collector.golden",
		},
		{
			templateName: "incident-io",
			format:       "json",
			goldenFile:   "testdata/incident_io.golden",
		},
	}

	testdataDir := "testdata"
	if *updateGolden {
		if err := os.MkdirAll(testdataDir, 0755); err != nil {
			t.Fatalf("failed to create testdata dir: %v", err)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.templateName, func(t *testing.T) {
			tmplContent, err := templates.GetTemplate(tc.templateName)
			if err != nil {
				t.Fatalf("GetTemplate(%q) failed: %v", tc.templateName, err)
			}

			p := params
			p.Format = tc.format
			rendered, err := svc.RenderTemplate(p, config, tmplContent)
			if err != nil {
				t.Fatalf("RenderTemplate failed: %v", err)
			}

			goldenPath := filepath.Clean(tc.goldenFile)

			if *updateGolden {
				if err := os.WriteFile(goldenPath, []byte(rendered), 0644); err != nil {
					t.Fatalf("failed to update golden file %q: %v", goldenPath, err)
				}
				t.Logf("Updated golden file: %s", goldenPath)
				return
			}

			expected, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file %q (run with -update to create): %v", goldenPath, err)
			}

			if rendered != string(expected) {
				t.Errorf("rendered output does not match golden file %q\n--- Expected:\n%s\n--- Got:\n%s", goldenPath, string(expected), rendered)
			}
		})
	}
}

func TestRenderGoldenFiles_7dWindow(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo999, err := common.NewSLOTarget(99.9)
	if err != nil {
		t.Fatalf("NewSLOTarget failed: %v", err)
	}

	params := domain.GenerateParams{
		TargetSLO:   slo999,
		TotalWindow: 7 * 24 * time.Hour,
		Format:      "yaml",
	}

	config := map[string]any{
		"metric_name": "api_errors",
		"namespace":   "monitoring",
		"rule_labels": map[string]any{
			"env":     "prod",
			"service": "api",
		},
		"meta_labels": map[string]any{
			"release": "prometheus",
		},
	}

	tmplContent, err := templates.GetTemplate("prometheus-operator")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}

	rendered, err := svc.RenderTemplate(params, config, tmplContent)
	if err != nil {
		t.Fatalf("RenderTemplate failed: %v", err)
	}

	goldenPath := filepath.Clean("testdata/prometheus_operator_7d.golden")

	if *updateGolden {
		if err := os.WriteFile(goldenPath, []byte(rendered), 0644); err != nil {
			t.Fatalf("failed to update golden file %q: %v", goldenPath, err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %q (run with -update to create): %v", goldenPath, err)
	}

	if rendered != string(expected) {
		t.Errorf("rendered output does not match golden file %q\n--- Expected:\n%s\n--- Got:\n%s", goldenPath, string(expected), rendered)
	}
}
