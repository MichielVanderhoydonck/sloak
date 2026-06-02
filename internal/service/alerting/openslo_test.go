package alerting

import (
	"os"
	"testing"
	"time"
)

func TestParseOpenSLO(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		wantTarget  float64
		wantWindow  time.Duration
		wantSLOName string
		wantConfig  map[string]any
	}{
		{
			name: "Valid OpenSLO with target ratio and duration",
			yamlContent: `
apiVersion: openslo/v1
kind: SLO
metadata:
  name: billing-success
  displayName: Billing Success Rate
spec:
  service: billing-service
  budgetingMethod: Occurrences
  indicatorRef: success-rate-sli
  objectives:
    - target: 0.999
  timeWindow:
    - duration: 30d
      isRolling: true
`,
			wantErr:    false,
			wantTarget: 99.9,
			wantWindow: 30 * 24 * time.Hour,
			wantSLOName: "billing-success",
			wantConfig: map[string]any{
				"metric_name": "billing-success",
				"name":        "Billing Success Rate",
				"service":     "billing-service",
			},
		},
		{
			name: "Valid OpenSLO with targetPercent and duration",
			yamlContent: `
apiVersion: openslo/v1
kind: SLO
metadata:
  name: latency-slo
spec:
  service: api-gateway
  budgetingMethod: Occurrences
  indicatorRef: latency-sli
  objectives:
    - targetPercent: 99.5
  timeWindow:
    - duration: 7d
      isRolling: true
`,
			wantErr:    false,
			wantTarget: 99.5,
			wantWindow: 7 * 24 * time.Hour,
			wantSLOName: "latency-slo",
			wantConfig: map[string]any{
				"metric_name": "latency-slo",
				"name":        "latency-slo",
				"service":     "api-gateway",
			},
		},
		{
			name: "Invalid kind",
			yamlContent: `
apiVersion: openslo/v1
kind: SLI
metadata:
  name: invalid-kind
`,
			wantErr: true,
		},
		{
			name: "Invalid apiVersion",
			yamlContent: `
apiVersion: invalid/v1
kind: SLO
metadata:
  name: invalid-api
`,
			wantErr: true,
		},
		{
			name: "Missing name",
			yamlContent: `
apiVersion: openslo/v1
kind: SLO
metadata:
  displayName: Missing Name
`,
			wantErr: true,
		},
		{
			name: "Missing objectives",
			yamlContent: `
apiVersion: openslo/v1
kind: SLO
metadata:
  name: test-slo
spec:
  service: billing-service
  budgetingMethod: Occurrences
  indicatorRef: success-rate-sli
  timeWindow:
    - duration: 30d
      isRolling: true
`,
			wantErr: true,
		},
		{
			name: "Missing timeWindow",
			yamlContent: `
apiVersion: openslo/v1
kind: SLO
metadata:
  name: test-slo
spec:
  service: billing-service
  budgetingMethod: Occurrences
  indicatorRef: success-rate-sli
  objectives:
    - target: 0.99
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "openslo-*.yaml")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.yamlContent); err != nil {
				t.Fatalf("failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			target, window, name, config, err := ParseOpenSLO(tmpFile.Name())
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseOpenSLO() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if target != tt.wantTarget {
					t.Errorf("expected target %f, got %f", tt.wantTarget, target)
				}
				if window != tt.wantWindow {
					t.Errorf("expected window %s, got %s", tt.wantWindow, window)
				}
				if name != tt.wantSLOName {
					t.Errorf("expected name %q, got %q", tt.wantSLOName, name)
				}
				if tt.wantConfig != nil {
					for k, v := range tt.wantConfig {
						if gotVal, ok := config[k]; !ok || gotVal != v {
							t.Errorf("expected config[%q] = %v, got %v", k, v, gotVal)
						}
					}
				}
			}
		})
	}
}
