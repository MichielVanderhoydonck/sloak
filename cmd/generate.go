package cmd

import "github.com/spf13/cobra"

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "generate",
		Short:         "Generates SRE configurations and rules.",
		Long:          `Generates actionable configurations, such as Prometheus Operator alert rules, based on SLO targets.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().StringP("output", "o", "", "Output format (e.g., json)")
	return cmd
}
