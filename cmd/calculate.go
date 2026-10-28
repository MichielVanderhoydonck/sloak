package cmd

import "github.com/spf13/cobra"

func NewCalculateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calculate",
		Short: "Performs core SRE calculations (budgets, burn rates, etc.)",
		Long: `The 'calculate' command provides a suite of tools for running
core SRE calculations against SLOs, error budgets, and burn rates.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().StringP("output", "o", "", "Output format (e.g., json)")
	return cmd
}
