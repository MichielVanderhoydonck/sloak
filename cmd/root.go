package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

var cfgFile string

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
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sloak.yaml)")

	rootCmd.AddCommand(calculateCmd)
	rootCmd.AddCommand(generateCmd)

	calculateCmd.AddCommand(errorbudgetCmd.NewErrorBudgetCmd(errorbudgetService.NewCalculatorService()))
	calculateCmd.AddCommand(burnrateCmd.NewBurnRateCmd(burnrateService.NewBurnRateService()))
	calculateCmd.AddCommand(dependencyCmd.NewDependencyCmd(dependencyService.NewAvailabilityService()))
	calculateCmd.AddCommand(disruptionCmd.NewDisruptionCmd(disruptionService.NewDisruptionService()))
	calculateCmd.AddCommand(feasibilityCmd.NewFeasibilityCmd(feasibilityService.NewFeasibilityService()))

	rootCmd.AddCommand(convertCmd.NewConvertCmd(convertService.NewConvertService()))

	alertSvc := alertingService.NewAlertGeneratorService()
	generateCmd.AddCommand(alertingTableCmd.NewAlertTableCmd(alertSvc))
	rootCmd.AddCommand(alertingTableCmd.NewRenderCmd(alertSvc))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
			viper.AddConfigPath(".")
			viper.SetConfigType("yaml")
			viper.SetConfigName(".sloak")
		}
	}

	viper.SetEnvPrefix("SLOAK")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
}
