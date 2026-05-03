// test/e2e/init_flags_test.go

//go:build e2e

package e2e

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// List of flags that we explicitly DO NOT want in the init config
var ignoredFlags = map[string]bool{
	"config":     true,
	"help":       true,
	"output":     true,
	"elapsed":    true,
	"consumed":   true,
	"components": true,
	"type":       true,
	"nines":      true,
	"downtime":   true,
}

func getSubCommands(cmdArgs []string) ([]string, error) {
	cmd := exec.Command(sloakBinaryPath, append(cmdArgs, "--help")...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var subCmds []string
	inCmdsSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, "Available Commands:") {
			inCmdsSection = true
			continue
		}
		if inCmdsSection {
			// Stop if we hit a new section, like Flags or Global Flags
			if line != "" && !strings.HasPrefix(line, "  ") {
				inCmdsSection = false
				continue
			}
			parts := strings.Fields(line)
			if len(parts) > 0 {
				subCmds = append(subCmds, parts[0])
			}
		}
	}
	return subCmds, nil
}

func getFlags(cmdArgs []string) ([]string, error) {
	cmd := exec.Command(sloakBinaryPath, append(cmdArgs, "--help")...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var flags []string
	// Matches lines like:
	//   -s, --slo float
	//       --window string
	flagRegex := regexp.MustCompile(`(?m)^\s+(?:-[a-zA-Z0-9],\s+)?--([a-zA-Z0-9-]+)`)

	matches := flagRegex.FindAllStringSubmatch(string(out), -1)
	for _, match := range matches {
		flags = append(flags, match[1])
	}

	return flags, nil
}

func collectAllFlags(t *testing.T, baseCmd []string) []string {
	flags, err := getFlags(baseCmd)
	if err != nil {
		t.Fatalf("Failed to get flags for %v: %v", baseCmd, err)
	}

	subCmds, err := getSubCommands(baseCmd)
	if err != nil {
		t.Fatalf("Failed to get subcommands for %v: %v", baseCmd, err)
	}

	for _, sc := range subCmds {
		// Ignore 'generate', 'help', 'completion' commands from the root
		if len(baseCmd) == 0 && (sc == "generate" || sc == "help" || sc == "completion" || sc == "init") {
			continue
		}

		subFlags := collectAllFlags(t, append(baseCmd, sc))
		flags = append(flags, subFlags...)
	}

	return flags
}

func TestInitTemplateContainsAllExpectedFlags(t *testing.T) {
	// 1. Dynamically discover all flags
	allFlags := collectAllFlags(t, []string{})

	// 2. Filter out ignored flags and deduplicate
	expectedFlags := make(map[string]bool)
	for _, f := range allFlags {
		if !ignoredFlags[f] {
			expectedFlags[f] = true
		}
	}

	// 3. Get the output of sloak init
	cmd := exec.Command(sloakBinaryPath, "init")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command sloak init failed: %v", err)
	}
	initOutput := string(out)

	// 4. Check that every expected flag is in the init output
	missingFlags := []string{}
	for f := range expectedFlags {
		// Assuming the init template outputs flags as yaml keys: `flag:`
		expectedKey := f + ":"
		if !strings.Contains(initOutput, expectedKey) {
			missingFlags = append(missingFlags, f)
		}
	}

	if len(missingFlags) > 0 {
		t.Errorf("WARNING: The 'sloak init' template is missing the following dynamically discovered flags:\n%v\n\n"+
			"If these flags should be global defaults, add them to cmd/init.go.\n"+
			"If these flags are context-specific, add them to 'ignoredFlags' in test/e2e/init_flags_test.go.", missingFlags)
	}
}
