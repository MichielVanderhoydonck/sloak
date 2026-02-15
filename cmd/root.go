package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	burnrateCmd "github.com/MichielVanderhoydonck/sloak/cmd/burnrate"
	errorbudgetCmd "github.com/MichielVanderhoydonck/sloak/cmd/errorbudget"

	burnrateService "github.com/MichielVanderhoydonck/sloak/internal/service/burnrate"
	errorbudgetService "github.com/MichielVanderhoydonck/sloak/internal/service/errorbudget"

	dependencyCmd "github.com/MichielVanderhoydonck/sloak/cmd/dependency"
	dependencyService "github.com/MichielVanderhoydonck/sloak/internal/service/dependency"

	convertCmd "github.com/MichielVanderhoydonck/sloak/cmd/convert"
	convertService "github.com/MichielVanderhoydonck/sloak/internal/service/convert"

	alertingTableCmd "github.com/MichielVanderhoydonck/sloak/cmd/alerting"
	alertingService "github.com/MichielVanderhoydonck/sloak/internal/service/alerting"

	disruptionCmd "github.com/MichielVanderhoydonck/sloak/cmd/disruption"
	disruptionService "github.com/MichielVanderhoydonck/sloak/internal/service/disruption"

	feasibilityCmd "github.com/MichielVanderhoydonck/sloak/cmd/feasibility"
	feasibilityService "github.com/MichielVanderhoydonck/sloak/internal/service/feasibility"
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
	rootCmd.AddCommand(generateCmd)

	calculatorSvc := errorbudgetService.NewCalculatorService()
	errorbudgetCmd.SetService(calculatorSvc)
	calculateCmd.AddCommand(errorbudgetCmd.NewErrorBudgetCmd())

	burnRateSvc := burnrateService.NewBurnRateService()
	burnrateCmd.SetService(burnRateSvc)
	calculateCmd.AddCommand(burnrateCmd.NewBurnRateCmd())

	depSvc := dependencyService.NewAvailabilityService()
	dependencyCmd.SetService(depSvc)
	calculateCmd.AddCommand(dependencyCmd.NewDependencyCmd())

	convSvc := convertService.NewConvertService()
	convertCmd.SetService(convSvc)
	rootCmd.AddCommand(convertCmd.NewConvertCmd())

	alertSvc := alertingService.NewAlertGeneratorService()
	alertingTableCmd.SetService(alertSvc)
	generateCmd.AddCommand(alertingTableCmd.NewAlertTableCmd())
	generateCmd.AddCommand(alertingTableCmd.NewPrometheusCmd())

	disSvc := disruptionService.NewDisruptionService()
	disruptionCmd.SetService(disSvc)
	calculateCmd.AddCommand(disruptionCmd.NewDisruptionCmd())

	feasSvc := feasibilityService.NewFeasibilityService()
	feasibilityCmd.SetService(feasSvc)
	calculateCmd.AddCommand(feasibilityCmd.NewFeasibilityCmd())
}
