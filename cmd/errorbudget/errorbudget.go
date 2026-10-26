package errorbudget

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	errorbudgetDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/errorbudget"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewErrorBudgetCmd(service errorbudgetDomain.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "errorbudget",
		Short:         "Calculates the total error budget (time) for a given SLO.",
		Long:          `Calculates the allowed failure time based on an SLO percentage and a time window.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runErrorBudgetCmd(cmd, service)
		},
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "The SLO target percentage (e.g., 99.9)")
	cmd.Flags().StringP("window", "w", "30d", "The time window (e.g., 7d, 30d, 1y)")

	return cmd
}

func runErrorBudgetCmd(cmd *cobra.Command, service errorbudgetDomain.Service) error {
	viper.BindPFlags(cmd.Flags())
	sloTarget := viper.GetFloat64("slo")
	windowStr := viper.GetString("window")

	timeWindow, err := util.ParseTimeWindow(windowStr)
	if err != nil {
		return fmt.Errorf("parsing window: %w", err)
	}

	sloTargetVO, err := common.NewSLOTarget(sloTarget)
	if err != nil {
		return err
	}

	params := errorbudgetDomain.CalculationParams{
		TargetSLO:  sloTargetVO,
		TimeWindow: util.Duration(timeWindow),
	}

	result, err := service.CalculateBudget(params)
	if err != nil {
		return fmt.Errorf("calculation error: %w", err)
	}

	output := viper.GetString("output")
	if output == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		return nil
	}

	fmt.Printf("\n--- Error Budget Calculation ---\n")
	fmt.Printf("SLO Target: %.3f%%\n", result.TargetSLO.Value)
	fmt.Printf("Time Window: %s\n", result.TotalDuration)
	fmt.Printf("--------------------------------\n")
	fmt.Printf("Error Budget: %.4f%% of time\n", result.ErrorBudget)
	fmt.Printf("Allowed Downtime: %s\n", result.AllowedError)
	return nil
}
