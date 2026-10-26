// test/e2e/cli_test.go

//go:build e2e

// Build tag tells Go to ONLY run this test when explicitly asked.

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	sloakBinaryPath string
)

func TestMain(m *testing.M) {
	fmt.Println("Building sloak binary for E2E tests...")

	cmd := exec.Command("go", "build", "-o", "sloak_test_binary", "../../cmd/sloak/main.go")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build test binary: %v\n", err)
		os.Exit(1)
	}

	sloakBinaryPath = "./sloak_test_binary"

	exitCode := m.Run()

	fmt.Println("Cleaning up test binary...")
	os.Remove(sloakBinaryPath)

	os.Exit(exitCode)
}

func TestErrorBudgetE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "calculate", "errorbudget", "--slo=99.95", "--window=30d")

	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Command failed with error: %v\nOutput: %s", err, outStr)
	}
	if !strings.Contains(outStr, "Allowed Downtime: 21m36s") {
		t.Errorf("Expected '21m36s' in output, got: %s", outStr)
	}
}

func TestBurnRateE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "calculate", "burnrate",
		"--slo=99.9",
		"--window=30d",
		"--elapsed=7d",
		"--consumed=30m",
	)

	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Command failed with error: %v\nOutput: %s", err, outStr)
	}

	if !strings.Contains(outStr, "Burn Rate: 2.98x") {
		t.Errorf("Expected '2.98x' in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Status: WARNING (Ticket)") {
		t.Errorf("Expected 'Status: WARNING (Ticket)' in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Forecast: Budget will be empty in") {
		t.Errorf("Expected Forecast message in output, got: %s", outStr)
	}

	// Test page-level critical burn rate (>= 14.4x)
	cmdPage := exec.Command(sloakBinaryPath, "calculate", "burnrate",
		"--slo=99.9",
		"--window=30d",
		"--elapsed=1h",
		"--consumed=15m",
	)
	outPage, errPage := cmdPage.CombinedOutput()
	outPageStr := string(outPage)
	if errPage != nil {
		t.Fatalf("Page burnrate command failed: %v\nOutput: %s", errPage, outPageStr)
	}
	if !strings.Contains(outPageStr, "Status: CRITICAL (Page)!") {
		t.Errorf("Expected 'Status: CRITICAL (Page)!' for high burn rate, got: %s", outPageStr)
	}
}

func TestDependencyE2E(t *testing.T) {
	// 99.9 and 99.9 in parallel should be 99.9999
	cmd := exec.Command(sloakBinaryPath, "calculate", "dependency",
		"--components=99.9,99.9",
		"--type=parallel",
	)

	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outStr)
	}

	if !strings.Contains(outStr, "Total Availability: 99.9999%") {
		t.Errorf("Expected '99.9999%%', got: %s", outStr)
	}
}

func TestConvertE2E(t *testing.T) {
	// Test translating 99.0% -> Should allow ~14m per day
	cmd := exec.Command(sloakBinaryPath, "convert", "--nines=99.0")
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outStr)
	}

	// 1% of 24h = 0.24h = 14.4 mins = 14m24s
	if !strings.Contains(outStr, "Daily Allowed:     14m24s") {
		t.Errorf("Expected 'Daily Allowed: 14m24s', got: %s", outStr)
	}
}

func TestAlertTableE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "generate", "alert-table",
		"--slo=99.9",
	)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outStr)
	}

	if !strings.Contains(outStr, "14.40x") {
		t.Error("Missing Burn Rate")
	}
	if !strings.Contains(outStr, "Page") {
		t.Error("Missing 'Page' notification type")
	}

	// Test 7-day window scaling
	cmd7d := exec.Command(sloakBinaryPath, "generate", "alert-table",
		"--slo=99.9",
		"--window=7d",
	)
	output7d, err := cmd7d.CombinedOutput()
	outStr7d := string(output7d)
	if err != nil {
		t.Fatalf("7d alert-table failed: %v\nOutput: %s", err, outStr7d)
	}
	if !strings.Contains(outStr7d, "14.40x") || !strings.Contains(outStr7d, "6.00x") || !strings.Contains(outStr7d, "1.00x") {
		t.Errorf("Expected scaled burn rates 14.40x, 6.00x, 1.00x for 7d window, got:\n%s", outStr7d)
	}

	// Test window under 24h fails
	cmdShort := exec.Command(sloakBinaryPath, "generate", "alert-table",
		"--slo=99.9",
		"--window=1h",
	)
	outShort, errShort := cmdShort.CombinedOutput()
	if errShort == nil {
		t.Errorf("Expected failure for 1h window, but command succeeded: %s", string(outShort))
	}

	// Test SLO = 100 fails
	cmd100 := exec.Command(sloakBinaryPath, "generate", "alert-table",
		"--slo=100",
		"--window=30d",
	)
	out100, err100 := cmd100.CombinedOutput()
	if err100 == nil {
		t.Errorf("Expected failure for SLO 100, but command succeeded: %s", string(out100))
	}

	// Test -o json output formatting
	cmdJSON := exec.Command(sloakBinaryPath, "generate", "alert-table",
		"--slo=99.9",
		"--window=30d",
		"-o=json",
	)
	outJSON, errJSON := cmdJSON.CombinedOutput()
	if errJSON != nil {
		t.Fatalf("Expected alert-table -o json to succeed: %v\nOutput: %s", errJSON, string(outJSON))
	}
	var resMap map[string]any
	if err := json.Unmarshal(outJSON, &resMap); err != nil {
		t.Fatalf("Failed to parse JSON alert-table output: %v", err)
	}
	if totalWin, ok := resMap["total_window"].(string); !ok || !strings.Contains(totalWin, "d") {
		t.Errorf("Expected total_window to be serialized as string with duration unit, got %v", resMap["total_window"])
	}
}

func TestDisruptionE2E(t *testing.T) {
	// 99.9% of 30d = ~43m. Cost = 1m. Expect 43 events.
	cmd := exec.Command(sloakBinaryPath, "calculate", "max-disruption",
		"--slo=99.9", "--window=30d", "--cost=1m",
	)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outStr)
	}

	if !strings.Contains(outStr, "Max Events Total: 43") {
		t.Errorf("Expected 43 events, got output:\n%s", outStr)
	}
}

func TestFeasibilityE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "calculate", "feasibility",
		"--slo=99.9", "--mttr=1h",
	)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outStr)
	}

	if !strings.Contains(outStr, "8.8 incidents") {
		t.Errorf("Expected 8.8 incidents/year, got output:\n%s", outStr)
	}

	if !strings.Contains(outStr, "CHALLENGING") {
		t.Errorf("Expected CHALLENGING status (2.2/qtr), got output:\n%s", outStr)
	}
}

func TestRenderE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "render",
		"--template=../../templates/prometheus_operator_mwmbr.cue", "--slo=99.9",
	)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outStr)
	}

	if !strings.Contains(outStr, "apiVersion: monitoring.coreos.com/v1") {
		t.Errorf("Expected prometheus operator apiVersion in output, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "slo-alerts") {
		t.Errorf("Expected slo-alerts in output, got:\n%s", outStr)
	}

	apiIdx := strings.Index(outStr, "apiVersion:")
	kindIdx := strings.Index(outStr, "kind:")
	metaIdx := strings.Index(outStr, "metadata:")
	specIdx := strings.Index(outStr, "spec:")

	if apiIdx == -1 || kindIdx == -1 || metaIdx == -1 || specIdx == -1 {
		t.Fatalf("Missing expected Kubernetes manifest sections, got:\n%s", outStr)
	}
	if !(apiIdx < kindIdx && kindIdx < metaIdx && metaIdx < specIdx) {
		t.Errorf("Expected manifest field ordering apiVersion < kind < metadata < spec, got indices: apiVersion=%d, kind=%d, metadata=%d, spec=%d",
			apiIdx, kindIdx, metaIdx, specIdx)
	}
	if strings.HasPrefix(outStr, "data:") || strings.Contains(outStr, "\ndata:") {
		t.Errorf("Expected top-level data field to be stripped from rendered output, got:\n%s", outStr)
	}
}

func TestRenderBuiltInTemplatesE2E(t *testing.T) {
	templates := []struct {
		name       string
		wantString string
	}{
		{"prometheus-operator", "apiVersion: monitoring.coreos.com/v1"},
		{"datadog", `"type": "metric"`},
		{"grafana-cloud", "groups:"},
		{"otel-collector", "connectors:"},
		{"incident-io", `"source_type": "custom"`},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(sloakBinaryPath, "render",
				"--template", tt.name, "--slo=99.9",
			)
			output, err := cmd.CombinedOutput()
			outStr := string(output)
			if err != nil {
				t.Fatalf("Template %s failed to render: %v\nOutput: %s", tt.name, err, outStr)
			}
			if !strings.Contains(outStr, tt.wantString) {
				t.Errorf("Template %s output missing %q. Got:\n%s", tt.name, tt.wantString, outStr)
			}
		})
	}
}

func TestRenderFormatAndDatadogTimeframeE2E(t *testing.T) {
	// 1. Datadog with 7d window -> "7d"
	cmd7d := exec.Command(sloakBinaryPath, "render", "--template", "datadog", "--window", "7d", "--slo", "99.9")
	out7d, err := cmd7d.CombinedOutput()
	if err != nil {
		t.Fatalf("datadog 7d failed: %v\nOutput: %s", err, string(out7d))
	}
	if !strings.Contains(string(out7d), `"timeframe": "7d"`) {
		t.Errorf("expected 7d timeframe, got:\n%s", string(out7d))
	}

	// 2. Datadog with 90d window -> "90d"
	cmd90d := exec.Command(sloakBinaryPath, "render", "--template", "datadog", "--window", "90d", "--slo", "99.9")
	out90d, err := cmd90d.CombinedOutput()
	if err != nil {
		t.Fatalf("datadog 90d failed: %v\nOutput: %s", err, string(out90d))
	}
	if !strings.Contains(string(out90d), `"timeframe": "90d"`) {
		t.Errorf("expected 90d timeframe, got:\n%s", string(out90d))
	}

	// 3. Datadog explicit YAML format
	cmdYAML := exec.Command(sloakBinaryPath, "render", "--template", "datadog", "-o", "yaml", "--slo", "99.9")
	outYAML, err := cmdYAML.CombinedOutput()
	if err != nil {
		t.Fatalf("datadog yaml failed: %v\nOutput: %s", err, string(outYAML))
	}
	if !strings.Contains(string(outYAML), "type: metric") {
		t.Errorf("expected YAML output with 'type: metric', got:\n%s", string(outYAML))
	}

	// 4. Prometheus Operator explicit JSON format
	cmdJSON := exec.Command(sloakBinaryPath, "render", "--template", "prometheus-operator", "-o", "json", "--slo", "99.9")
	outJSON, err := cmdJSON.CombinedOutput()
	if err != nil {
		t.Fatalf("prometheus-operator json failed: %v\nOutput: %s", err, string(outJSON))
	}
	if !strings.Contains(string(outJSON), `"apiVersion": "monitoring.coreos.com/v1"`) {
		t.Errorf("expected JSON output with 'apiVersion', got:\n%s", string(outJSON))
	}
}

func TestRenderCustomTemplatePrecedenceE2E(t *testing.T) {
	// Create a custom template file named datadog.cue in a temp dir.
	// It should render custom output and NOT the built-in datadog template.
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "datadog.cue")
	customContent := `package templates

data: _

custom_marker: "my-custom-datadog-override"
name: "Custom Datadog SLO"
`
	if err := os.WriteFile(customPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom template: %v", err)
	}

	cmd := exec.Command(sloakBinaryPath, "render",
		"--template", customPath,
		"--slo=99.9",
	)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err != nil {
		t.Fatalf("Custom template render failed: %v\nOutput: %s", err, outStr)
	}

	if !strings.Contains(outStr, "my-custom-datadog-override") {
		t.Errorf("Expected custom template content in output, got:\n%s", outStr)
	}
	// Verify it does NOT contain the built-in datadog output
	if strings.Contains(outStr, "type: metric") {
		t.Errorf("Output contains built-in datadog content when custom template was provided:\n%s", outStr)
	}
}

func TestRenderSeverityOverridesE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "render",
		"--template", "prometheus-operator",
		"--slo=99.9",
		"--values", "../../templates/values.yaml",
	)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err != nil {
		t.Fatalf("Failed to render prometheus-operator with values.yaml: %v\nOutput: %s", err, outStr)
	}

	expectedSubstrings := []string{
		"urgency: high",
		"urgency: medium",
		"urgency: low",
		"dashboard_url: https://grafana.example.com/d/checkout-slo",
		"action: Escalate to on-call engineer immediately",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(outStr, s) {
			t.Errorf("Output missing expected override %q. Got:\n%s", s, outStr)
		}
	}
}

func TestRenderSetFlagsE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "render",
		"--template", "prometheus-operator",
		"--slo=99.9",
		"--set", "metric_name=api_errors",
		"--set", "rule_labels=service=api,env=prod",
		"--set", "meta_labels.release=prometheus",
	)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err != nil {
		t.Fatalf("Failed to render with --set flags: %v\nOutput: %s", err, outStr)
	}

	expectedSubstrings := []string{
		"job:api_errors:",
		"service: api",
		"env: prod",
		"release: prometheus",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(outStr, s) {
			t.Errorf("Expected output to contain %q, but it did not.\nFull output:\n%s", s, outStr)
		}
	}
}

func TestRenderOpenSLOE2E(t *testing.T) {
	yamlContent := `
apiVersion: openslo/v1
kind: SLO
metadata:
  name: test-openslo-metric
spec:
  service: my-service
  budgetingMethod: Occurrences
  indicatorRef: success-rate-sli
  objectives:
    - target: 0.999
  timeWindow:
    - duration: 30d
      isRolling: true
`
	tmpFile, err := os.CreateTemp("", "openslo-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	cmd := exec.Command(sloakBinaryPath, "render",
		"--spec", tmpFile.Name(),
		"--template", "prometheus-operator",
	)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outStr)
	}

	// Should contain the sanitized metric name in PromQL expressions
	if !strings.Contains(outStr, "job:test_openslo_metric:") {
		t.Errorf("Expected sanitized metric name 'test_openslo_metric' in PromQL expr, got:\n%s", outStr)
	}
	// PromQL must NOT contain invalid hyphenated metric name
	if strings.Contains(outStr, "job:test-openslo-metric:") {
		t.Errorf("Output contains invalid PromQL metric name with hyphens:\n%s", outStr)
	}
	// Should have correct SLO math based on 99.9%
	if !strings.Contains(outStr, "0.001") {
		t.Errorf("Expected correct error budget ratio (0.001) in output, got:\n%s", outStr)
	}
}

func TestRenderMissingTemplateFailureE2E(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sloakBinaryPath, "render")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected failure exit code when --template is missing, got status 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if stdout.Len() != 0 {
		t.Errorf("expected stdout to be empty on error, got: %q", stdout.String())
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "Error: --template is required") {
		t.Errorf("expected stderr to contain 'Error: --template is required', got: %q", stderrStr)
	}
}

func TestRenderInvalidTemplateFailureE2E(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sloakBinaryPath, "render", "--template=nonexistent_file.cue")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected failure exit code for nonexistent template, got status 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if stdout.Len() != 0 {
		t.Errorf("expected stdout to be empty on error, got: %q", stdout.String())
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "Error: reading template file") {
		t.Errorf("expected stderr to contain 'Error: reading template file', got: %q", stderrStr)
	}
}

func TestBurnRateInvalidSLOFailureE2E(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sloakBinaryPath, "calculate", "burnrate", "--slo=120")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected failure exit code for invalid slo, got status 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if stdout.Len() != 0 {
		t.Errorf("expected stdout to be empty on error, got: %q", stdout.String())
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "Error: SLO target must be strictly between 0.0 and 100.0") {
		t.Errorf("expected stderr to contain SLO error, got: %q", stderrStr)
	}
}

func TestConvertConflictingFlagsFailureE2E(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sloakBinaryPath, "convert", "--nines=99.9", "--downtime=10m")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected failure exit code for conflicting flags, got status 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if stdout.Len() != 0 {
		t.Errorf("expected stdout to be empty on error, got: %q", stdout.String())
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "Error: please provide EITHER --nines OR --downtime, not both") {
		t.Errorf("expected stderr to contain conflict error, got: %q", stderrStr)
	}
}

func TestBurnRateUnboundedJSONE2E(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sloakBinaryPath, "calculate", "burnrate", "--elapsed=0s", "--consumed=5m", "-o", "json")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected success exit code for unbounded burnrate json, got error: %v\nStderr: %s", err, stderr.String())
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, `"burn_rate": null`) {
		t.Errorf("expected JSON to contain '\"burn_rate\": null', got: %s", outStr)
	}
	if !strings.Contains(outStr, `"burn_rate_unbounded": true`) {
		t.Errorf("expected JSON to contain '\"burn_rate_unbounded\": true', got: %s", outStr)
	}
}

func TestConvertJSONE2E(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sloakBinaryPath, "convert", "--nines=99.9", "-o", "json")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected success for convert -o json, got error: %v\nStderr: %s", err, stderr.String())
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, `"availability_percent": 99.9`) {
		t.Errorf("expected JSON to contain availability_percent, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"daily_downtime": "1m26s"`) {
		t.Errorf("expected JSON to contain formatted daily_downtime string, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"daily_downtime_seconds": 86`) {
		t.Errorf("expected JSON to contain daily_downtime_seconds, got: %s", outStr)
	}
}

func TestFeasibilityZeroMTTRE2E(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sloakBinaryPath, "calculate", "feasibility", "--mttr=0s", "-o", "json")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected failure for 0s mttr, got success")
	}

	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout on error, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Error: calculation error: MTTR must be greater than zero") {
		t.Errorf("expected MTTR error in stderr, got: %s", stderr.String())
	}
}

func TestConvertZeroWindowE2E(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sloakBinaryPath, "convert", "--downtime=5m", "--window=0s")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected failure for 0s window, got success")
	}

	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout on error, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Error: calculation error: window must be greater than zero") {
		t.Errorf("expected window error in stderr, got: %s", stderr.String())
	}
}

func TestNegativePathCLI_E2E(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "ErrorBudget_SLO_150",
			args:          []string{"calculate", "errorbudget", "--slo=150"},
			expectedError: "SLO target must be strictly between 0.0 and 100.0",
		},
		{
			name:          "ErrorBudget_SLO_0",
			args:          []string{"calculate", "errorbudget", "--slo=0"},
			expectedError: "SLO target must be strictly between 0.0 and 100.0",
		},
		{
			name:          "ErrorBudget_SLO_100",
			args:          []string{"calculate", "errorbudget", "--slo=100"},
			expectedError: "SLO target must be strictly between 0.0 and 100.0",
		},
		{
			name:          "ErrorBudget_InvalidWindowUnit",
			args:          []string{"calculate", "errorbudget", "--window=30x"},
			expectedError: "unknown unit \"x\"",
		},
		{
			name:          "ErrorBudget_NegativeWindow",
			args:          []string{"calculate", "errorbudget", "--window=-30d"},
			expectedError: "duration cannot be negative",
		},
		{
			name:          "ErrorBudget_ZeroWindow",
			args:          []string{"calculate", "errorbudget", "--window=0s"},
			expectedError: "time window must be greater than zero",
		},
		{
			name:          "BurnRate_InvalidConsumed",
			args:          []string{"calculate", "burnrate", "--consumed=foo"},
			expectedError: "invalid duration \"foo\"",
		},
		{
			name:          "BurnRate_ElapsedExceedsWindow",
			args:          []string{"calculate", "burnrate", "--elapsed=40d", "--window=30d"},
			expectedError: "time elapsed cannot exceed total window",
		},
		{
			name:          "Disruption_CostExceedsWindow",
			args:          []string{"calculate", "max-disruption", "--cost=40d", "--window=30d"},
			expectedError: "cost per disruption cannot exceed total window",
		},
		{
			name:          "Dependency_MissingComponents",
			args:          []string{"calculate", "dependency"},
			expectedError: "--components is required",
		},
		{
			name:          "Dependency_InvalidComponent",
			args:          []string{"calculate", "dependency", "--components=99.9,xyz"},
			expectedError: "parsing component 'xyz'",
		},
		{
			name:          "Feasibility_SLO_150",
			args:          []string{"calculate", "feasibility", "--slo=150"},
			expectedError: "SLO target must be strictly between 0.0 and 100.0",
		},
		{
			name:          "ErrorBudget_OverflowWindow",
			args:          []string{"calculate", "errorbudget", "--window=300y"},
			expectedError: "overflows maximum duration limit",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := exec.Command(sloakBinaryPath, tc.args...)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err == nil {
				t.Fatalf("expected non-zero exit code for %v, got 0", tc.args)
			}

			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
			}
			if exitErr.ExitCode() != 1 {
				t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
			}

			if stdout.Len() != 0 {
				t.Errorf("expected stdout to be empty on failure, got: %q", stdout.String())
			}

			stderrStr := stderr.String()
			if !strings.Contains(stderrStr, tc.expectedError) {
				t.Errorf("expected stderr to contain %q, got: %q", tc.expectedError, stderrStr)
			}
		})
	}
}

func TestBurnRateOverspentBudgetE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "calculate", "burnrate",
		"--slo=99.9",
		"--window=30d",
		"--elapsed=15d",
		"--consumed=60m",
		"-o", "json",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected overspent burnrate calculation to succeed: %v\nOutput: %s", err, string(output))
	}

	var res map[string]any
	if err := json.Unmarshal(output, &res); err != nil {
		t.Fatalf("failed to parse json output: %v", err)
	}

	// Budget was 43m12s. Consumed 60m. Budget remaining should be negative: -16m48s.
	remainingStr, ok := res["budget_remaining"].(string)
	if !ok || !strings.HasPrefix(remainingStr, "-") {
		t.Errorf("expected negative budget_remaining string, got: %v", res["budget_remaining"])
	}

	// Budget consumed should be > 100%
	consumedPct, ok := res["budget_consumed_percentage"].(float64)
	if !ok || consumedPct < 100.0 {
		t.Errorf("expected budget_consumed_percentage > 100%%, got: %v", res["budget_consumed_percentage"])
	}
}
