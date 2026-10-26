// test/e2e/config_test.go

//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestConfigOverrideE2E(t *testing.T) {
	// Create a temporary .sloak.yaml config file
	configContent := `
slo: 95.0
window: 7d
`
	err := os.WriteFile(".sloak.yaml", []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	// Make sure to clean up the config file after the test
	defer os.Remove(".sloak.yaml")

	// Run errorbudget calculation without explicit flags, so it uses the config
	cmd := exec.Command(sloakBinaryPath, "calculate", "errorbudget")
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Command failed with error: %v\nOutput: %s", err, outStr)
	}

	// 95% of 7d = 8.4 hours = 8h24m0s
	if !strings.Contains(outStr, "SLO Target: 95.000%") {
		t.Errorf("Expected 'SLO Target: 95.000%%' in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Time Window: 7.0d") {
		t.Errorf("Expected 'Time Window: 7.0d' in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Allowed Downtime: 8h24m0s") {
		t.Errorf("Expected 'Allowed Downtime: 8h24m0s' in output, got: %s", outStr)
	}
}

func TestEnvOverrideE2E(t *testing.T) {
	// Run burnrate calculation using ENV vars
	cmd := exec.Command(sloakBinaryPath, "calculate", "burnrate", "--consumed", "1h")
	cmd.Env = append(os.Environ(), "SLOAK_SLO=96.0", "SLOAK_WINDOW=7d")

	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Command failed with error: %v\nOutput: %s", err, outStr)
	}

	if !strings.Contains(outStr, "SLO Target: 96.000%") {
		t.Errorf("Expected 'SLO Target: 96.000%%' in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Time Window: 7.0d") {
		t.Errorf("Expected 'Time Window: 7.0d' in output, got: %s", outStr)
	}
}

func TestMissingExplicitConfigFileE2E(t *testing.T) {
	cmd := exec.Command(sloakBinaryPath, "calculate", "errorbudget", "--config", "nonexistent_config_file.yaml")
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err == nil {
		t.Fatal("Expected command to fail on missing explicit config file, but it succeeded")
	}

	if !strings.Contains(outStr, "reading config file") {
		t.Errorf("Expected error to mention reading config file, got:\n%s", outStr)
	}
}

func TestMalformedConfigFileE2E(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "malformed-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("slo: [invalid yaml\n  unbalanced"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	cmd := exec.Command(sloakBinaryPath, "calculate", "errorbudget", "--config", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err == nil {
		t.Fatal("Expected command to fail on malformed config file, but it succeeded")
	}

	if !strings.Contains(outStr, "reading config file") {
		t.Errorf("Expected error to mention reading config file, got:\n%s", outStr)
	}
}
