package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Prints a default .sloak.yaml configuration template",
	Long:  `Prints a default .sloak.yaml template to standard output to help you get started with file-based configuration. You can redirect this to a file (e.g. sloak init > .sloak.yaml).`,
	Run: func(cmd *cobra.Command, args []string) {
		configContent := `# Default SLOAK Configuration File
# This file can be placed in your home directory (~/.sloak.yaml) or your current working directory.

# The default Service Level Objective target percentage
slo: 99.9

# The default time window used for calculations
window: 30d

# Mean Time To Recover (average incident duration)
# Used for feasibility calculations
mttr: 30m

# Estimated downtime cost per event (e.g. deployment, restart)
# Used for disruption budget calculations
cost: 10s
`
		fmt.Print(configContent)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
