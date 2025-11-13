package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	burnrateCmd "github.com/MichielVanderhoydonck/sloak/cmd/burnrate"
	errorbudgetCmd "github.com/MichielVanderhoydonck/sloak/cmd/errorbudget"

	burnrateService "github.com/MichielVanderhoydonck/sloak/internal/core/service/burnrate"
	errorbudgetService "github.com/MichielVanderhoydonck/sloak/internal/core/service/errorbudget"
)

var rootCmd = &cobra.Command{
	Use:   "sloak",
	Short: "SLOAK is a Service Level Objective Army Knife for SRE calculations.",
	Long: `SLOAK provides a suite of tools for calculating error budgets, 
           SLI attainment, burn rates, and more, following strict SRE principles.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(calculateCmd)

	calculatorSvc := errorbudgetService.NewCalculatorService()
	errorbudgetCmd.SetService(calculatorSvc)
	calculateCmd.AddCommand(errorbudgetCmd.NewErrorBudgetCmd())

	burnRateSvc := burnrateService.NewBurnRateService()
	burnrateCmd.SetService(burnRateSvc)
	calculateCmd.AddCommand(burnrateCmd.NewBurnRateCmd())
}
