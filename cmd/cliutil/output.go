package cliutil

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// PrintResult prints the result either as indented JSON or by executing humanFn with cmd.OutOrStdout().
func PrintResult(cmd *cobra.Command, format string, v any, humanFn func(w io.Writer) error) error {
	out := cmd.OutOrStdout()
	if format == "json" {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(v); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		return nil
	}
	return humanFn(out)
}
