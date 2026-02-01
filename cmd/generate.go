package cmd

import "github.com/spf13/cobra"

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates SRE configurations and rules.",
	Long:  `Generates actionable configurations, such as Prometheus alert rules, based on SLO targets.`,
}

func init() {
	generateCmd.PersistentFlags().StringP("output", "o", "", "Output format (e.g., json)")
}
