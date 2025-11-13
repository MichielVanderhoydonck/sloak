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
	// Arrange
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
}