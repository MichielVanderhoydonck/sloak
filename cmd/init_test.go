package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestInitCmd(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	initCmd.Run(initCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
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
