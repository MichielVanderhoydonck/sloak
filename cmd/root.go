package cmd

import (
	"errors"
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

var (
	cfgFile   string
	configErr error
)

var rootCmd = &cobra.Command{
	Use:   "sloak",
	Short: "SLOAK is a Service Level Objective Army Knife for SRE calculations.",
	Long: `SLOAK provides a suite of tools for calculating error budgets, 
           SLI attainment, burn rates, and more, following strict SRE principles.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		if configErr != nil {
			return configErr
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
	configErr = nil
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

	if err := viper.ReadInConfig(); err != nil {
		if cfgFile != "" {
			configErr = fmt.Errorf("reading config file %q: %w", cfgFile, err)
		} else {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if !errors.As(err, &configFileNotFoundError) {
				configErr = fmt.Errorf("reading config file: %w", err)
			}
		}
	}
}
