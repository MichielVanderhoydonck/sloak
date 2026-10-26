package alerting_test

import (
	"testing"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
)

func TestSanitizePrometheusMetricName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"billing-success", "billing_success"},
		{"test-openslo-metric", "test_openslo_metric"},
		{"api.latency/errors", "api_latency_errors"},
		{"valid_metric:name", "valid_metric:name"},
		{"123start_with_digit", "_123start_with_digit"},
		{"", "slo_errors"},
		{"___", "___"},
		{"-", "slo_errors"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := domain.SanitizePrometheusMetricName(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizePrometheusMetricName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
