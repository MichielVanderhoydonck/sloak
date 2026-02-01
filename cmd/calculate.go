package cmd

import "github.com/spf13/cobra"

var calculateCmd = &cobra.Command{
	Use:   "calculate",
	Short: "Performs core SRE calculations (budgets, burn rates, etc.)",
	Long: `The 'calculate' command provides a suite of tools for running
core SRE calculations against SLOs, error budgets, and burn rates.`,
}

func init() {
	calculateCmd.PersistentFlags().StringP("output", "o", "", "Output format (e.g., json)")
}
