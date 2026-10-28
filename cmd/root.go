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

	openslo "github.com/MichielVanderhoydonck/sloak/internal/adapter/openslo"
)

func NewRootCmd() *cobra.Command {
	var (
		cfgFile   string
		configErr error
	)

	root := &cobra.Command{
		Use:   "sloak",
		Short: "SLOAK is a Service Level Objective Army Knife for SRE calculations.",
		Long: `SLOAK provides a suite of tools for calculating error budgets, 
           SLI attainment, burn rates, and more, following strict SRE principles.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			initConfig(cfgFile, &configErr)
			return configErr
		},
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sloak.yaml)")

	calcCmd := NewCalculateCmd()
	genCmd := NewGenerateCmd()

	root.AddCommand(calcCmd)
	root.AddCommand(genCmd)
	root.AddCommand(NewInitCmd())

	calcCmd.AddCommand(errorbudgetCmd.NewErrorBudgetCmd(errorbudgetService.NewCalculatorService()))
	calcCmd.AddCommand(burnrateCmd.NewBurnRateCmd(burnrateService.NewBurnRateService()))
	calcCmd.AddCommand(dependencyCmd.NewDependencyCmd(dependencyService.NewAvailabilityService()))
	calcCmd.AddCommand(disruptionCmd.NewDisruptionCmd(disruptionService.NewDisruptionService()))
	calcCmd.AddCommand(feasibilityCmd.NewFeasibilityCmd(feasibilityService.NewFeasibilityService()))

	root.AddCommand(convertCmd.NewConvertCmd(convertService.NewConvertService()))

	alertSvc := alertingService.NewAlertGeneratorService()
	specLoader := openslo.NewSpecLoader()
	genCmd.AddCommand(alertingTableCmd.NewAlertTableCmd(alertSvc))
	root.AddCommand(alertingTableCmd.NewRenderCmd(alertSvc, specLoader))

	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initConfig(cfgFile string, configErr *error) {
	*configErr = nil
	viper.Reset()
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
			*configErr = fmt.Errorf("reading config file %q: %w", cfgFile, err)
		} else {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if !errors.As(err, &configFileNotFoundError) {
				*configErr = fmt.Errorf("reading config file: %w", err)
			}
		}
	}
}
