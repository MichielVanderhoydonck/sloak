package cliutil_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/cmd/cliutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestScopedConfigIsolation(t *testing.T) {
	cmd1 := &cobra.Command{Use: "cmd1"}
	cmd1.Flags().String("foo", "default1", "foo flag")
	if err := cmd1.ParseFlags([]string{"--foo=custom1"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	v1, err := cliutil.ScopedConfig(cmd1)
	if err != nil {
		t.Fatalf("ScopedConfig failed: %v", err)
	}
	if got := v1.GetString("foo"); got != "custom1" {
		t.Errorf("v1 expected 'custom1', got %q", got)
	}

	// Ensure global viper was NOT polluted
	if viper.IsSet("foo") {
		t.Errorf("expected global viper not to have 'foo' set, but it was")
	}

	// Create second command with different value
	cmd2 := &cobra.Command{Use: "cmd2"}
	cmd2.Flags().String("foo", "default2", "foo flag")

	v2, err := cliutil.ScopedConfig(cmd2)
	if err != nil {
		t.Fatalf("ScopedConfig failed: %v", err)
	}
	if got := v2.GetString("foo"); got != "default2" {
		t.Errorf("v2 expected 'default2', got %q", got)
	}
}

func TestScopedConfigWithBaseContext(t *testing.T) {
	base := viper.New()
	base.Set("slo", 95.0)
	base.Set("window", "14d")

	cmd := &cobra.Command{Use: "sub"}
	cmd.Flags().Float64("slo", 99.9, "slo flag")
	cmd.Flags().String("window", "30d", "window flag")
	cliutil.WithBaseConfig(cmd, base)

	// Without explicit flag override, base config from context wins over flag default
	v, err := cliutil.ScopedConfig(cmd)
	if err != nil {
		t.Fatalf("ScopedConfig failed: %v", err)
	}
	if got := v.GetFloat64("slo"); got != 95.0 {
		t.Errorf("expected slo 95.0 from base context, got %v", got)
	}
	if got := v.GetString("window"); got != "14d" {
		t.Errorf("expected window 14d from base context, got %q", got)
	}
}

func TestPrintResult(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	data := map[string]string{"key": "value"}

	// Test JSON format
	if err := cliutil.PrintResult(cmd, "json", data, func(w io.Writer) error {
		t.Fatal("humanFn should not be called when format is json")
		return nil
	}); err != nil {
		t.Fatalf("PrintResult(json) failed: %v", err)
	}

	if !strings.Contains(buf.String(), `"key": "value"`) {
		t.Errorf("expected JSON output containing key: value, got %q", buf.String())
	}

	// Test Human format
	humanCalled := false
	if err := cliutil.PrintResult(cmd, "", data, func(w io.Writer) error {
		humanCalled = true
		fmt.Fprint(w, "human output")
		return nil
	}); err != nil {
		t.Fatalf("PrintResult(human) failed: %v", err)
	}

	if !humanCalled {
		t.Errorf("expected humanFn to be called when format is not json")
	}
}
