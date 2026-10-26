package burnrate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/burnrate"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewBurnRateCmd(service burnrateDomain.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "burnrate",
		Short:         "Calculates the current error budget burn rate (consumption speed).",
		Long:          `Calculates the current burn rate against the ideal burn rate (1.0). A value > 1.0 indicates overspending.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBurnRateCmd(cmd, service)
		},
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "The SLO target percentage (e.g., 99.9)")
	cmd.Flags().StringP("window", "w", "30d", "The total time window (e.g., 30d)")
	cmd.Flags().StringP("elapsed", "t", "15d", "Time elapsed since start of the window.")
	cmd.Flags().StringP("consumed", "c", "1h", "Total error time consumed (e.g., 1h, 10m).")

	return cmd
}

func runBurnRateCmd(cmd *cobra.Command, service burnrateDomain.Service) error {
	viper.BindPFlags(cmd.Flags())
	sloTarget := viper.GetFloat64("slo")
	windowStr := viper.GetString("window")
	elapsedStr := viper.GetString("elapsed")
	consumedStr := viper.GetString("consumed")

	totalWindow, err := util.ParseTimeWindow(windowStr)
	if err != nil {
		return fmt.Errorf("parsing window: %w", err)
	}
	timeElapsed, err := util.ParseTimeWindow(elapsedStr)
	if err != nil {
		return fmt.Errorf("parsing elapsed: %w", err)
	}
	errorConsumed, err := util.ParseTimeWindow(consumedStr)
	if err != nil {
		return fmt.Errorf("parsing consumed duration: %w", err)
	}

	sloTargetVO, err := common.NewSLOTarget(sloTarget)
	if err != nil {
		return err
	}

	params := burnrateDomain.CalculationParams{
		TargetSLO:     sloTargetVO,
		TotalWindow:   util.Duration(totalWindow),
		TimeElapsed:   util.Duration(timeElapsed),
		ErrorConsumed: util.Duration(errorConsumed),
	}

	result, err := service.CalculateBurnRate(params)
	if err != nil {
		return fmt.Errorf("calculation error: %w", err)
	}

	outputFlag := viper.GetString("output")
	if outputFlag == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		return nil
	}

	fmt.Printf("\n--- Burn Rate Analysis ---\n")
	fmt.Printf("SLO Target: %.3f%%\n", sloTarget)

	fmt.Printf("Time Window: %s\n", params.TotalWindow)

	fmt.Printf("Total Error Budget: %s\n", result.TotalErrorBudget)

	fmt.Printf("--------------------------\n")
	fmt.Printf("Budget Consumed: %.2f%% (Time Elapsed: %.2f%%)\n",
		result.BudgetConsumed,
		(float64(timeElapsed)/float64(totalWindow))*100.0)
	if result.BurnRate != nil {
		fmt.Printf("Burn Rate: %.2fx\n", *result.BurnRate)
	} else if result.BurnRateUnbounded {
		fmt.Println("Burn Rate: Unbounded (instant consumption)")
	} else {
		fmt.Println("Burn Rate: 0.00x")
	}
	fmt.Printf("Budget Remaining: %s\n", result.BudgetRemaining)

	if result.BurnRateUnbounded || (result.BurnRate != nil && *result.BurnRate > 1.0) {
		fmt.Println("\nStatus: CRITICAL! Budget is burning faster than expected.")
		fmt.Printf("Forecast: Budget will be empty in %s\n", result.TimeToExhaustion)
		exhaustionDate := time.Now().Add(time.Duration(result.TimeToExhaustion))
		fmt.Printf("Predicted Exhaustion: %s\n", exhaustionDate.Format(time.RFC1123))
	} else if result.IsInfinite {
		fmt.Println("\nStatus: Excellent! No error budget consumed.")
	} else {
		fmt.Println("\nStatus: Healthy. Budget is being consumed at an acceptable rate.")
	}
	return nil
}
