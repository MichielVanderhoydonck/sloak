// test/e2e/cli_test.go

//go:build e2e
// Build tag tells Go to ONLY run this test when explicitly asked.

package e2e

import (
	"fmt"
	"os"
	"os/exec"
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
	if !strings.Contains(outStr, "Status: CRITICAL!") {
		t.Errorf("Expected 'CRITICAL!' in output, got: %s", outStr)
	}
    if !strings.Contains(outStr, "Forecast: Budget will be empty in") {
        t.Errorf("Expected Forecast message in output, got: %s", outStr)
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

	if !strings.Contains(outStr, "Total Availability: 99.999900%") {
		t.Errorf("Expected '99.999900%%', got: %s", outStr)
	}
}

func TestTranslatorE2E(t *testing.T) {
	// Test translating 99.0% -> Should allow ~14m per day
	cmd := exec.Command(sloakBinaryPath, "calculate", "translator", "--nines=99.0")
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
	output, _ := cmd.CombinedOutput()
	outStr := string(output)

	if !strings.Contains(outStr, "14.40x") {
		t.Error("Missing Burn Rate")
	}
	if !strings.Contains(outStr, "Page") {
		t.Error("Missing 'Page' notification type")
	}
}

func TestDisruptionE2E(t *testing.T) {
	// 99.9% of 30d = ~43m. Cost = 1m. Expect 43 events.
	cmd := exec.Command(sloakBinaryPath, "calculate", "max-disruption", 
		"--slo=99.9", "--window=30d", "--cost=1m",
	)
	output, _ := cmd.CombinedOutput()
	outStr := string(output)

	if !strings.Contains(outStr, "Max Events Total: 43") {
		t.Errorf("Expected 43 events, got output:\n%s", outStr)
	}
}