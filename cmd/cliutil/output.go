package cliutil

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// PrintResult prints the result either as indented JSON or by executing humanFn.
func PrintResult(cmd *cobra.Command, format string, v any, humanFn func() error) error {
	if format == "json" {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(v); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		return nil
	}
	return humanFn()
}
