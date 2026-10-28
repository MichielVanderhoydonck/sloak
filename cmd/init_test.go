package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestInitCmd(t *testing.T) {
	var buf bytes.Buffer
	initCmd := NewInitCmd()
	initCmd.SetOut(&buf)

	// Run command
	initCmd.Run(initCmd, []string{})

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
