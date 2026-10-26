package cmd

import (
	"strings"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
)

func TestInitCmd(t *testing.T) {
	buf, restoreStdout := testutil.CaptureOutput(t)

	// Run command
	initCmd.Run(initCmd, []string{})
	restoreStdout()

	outStr := buf.String()

	// Verify output
	expectedSubstrings := []string{
		"slo: 99.9",
		"window: 30d",
		"mttr: 30m",
		"cost: 10s",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(outStr, expected) {
			t.Errorf("Expected '%s' in output, but it was missing.\nOutput:\n%s", expected, outStr)
		}
	}
}
