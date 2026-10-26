package burnrate

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichielVanderhoydonck/sloak/cmd/cliutil"
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
	v, err := cliutil.ScopedConfig(cmd)
	if err != nil {
		return err
	}
	sloTarget := v.GetFloat64("slo")
	windowStr := v.GetString("window")
	elapsedStr := v.GetString("elapsed")
	consumedStr := v.GetString("consumed")

	totalWindow, err := util.ParseTimeWindow(windowStr)
	if err != nil {
		return fmt.Errorf("parsing window: %w", err)
	}

	var timeElapsed time.Duration
	if cmd.Flags().Changed("elapsed") || v.IsSet("elapsed") {
		timeElapsed, err = util.ParseTimeWindow(elapsedStr)
		if err != nil {
			return fmt.Errorf("parsing elapsed: %w", err)
		}
	} else {
		timeElapsed = totalWindow / 2
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

	return cliutil.PrintResult(cmd, v.GetString("output"), result, func() error {

		fmt.Printf("\n--- Burn Rate Analysis ---\n")
		fmt.Printf("SLO Target: %.3f%%\n", params.TargetSLO.Value)

		fmt.Printf("Time Window: %s\n", params.TotalWindow)

		fmt.Printf("Total Error Budget: %s\n", result.TotalErrorBudget)

		fmt.Printf("--------------------------\n")
		fmt.Printf("Budget Consumed: %.2f%% (Time Elapsed: %.2f%%)\n",
			result.BudgetConsumed,
			result.TimeElapsedPercent)
		if result.BurnRate != nil {
			fmt.Printf("Burn Rate: %.2fx\n", *result.BurnRate)
		} else if result.BurnRateUnbounded {
			fmt.Println("Burn Rate: Unbounded (instant consumption)")
		} else {
			fmt.Println("Burn Rate: 0.00x")
		}
		fmt.Printf("Budget Remaining: %s\n", result.BudgetRemaining)

		switch {
		case result.NoBudgetConsumed || result.IsInfinite:
			fmt.Println("\nStatus: Excellent! No error budget consumed.")
		case result.BurnRateUnbounded || (result.BurnRate != nil && *result.BurnRate >= 14.4):
			fmt.Println("\nStatus: CRITICAL (Page)! Budget is burning at an emergency rate.")
			fmt.Printf("Forecast: Budget will be empty in %s\n", result.TimeToExhaustion)
			exhaustionDate := time.Now().Add(time.Duration(result.TimeToExhaustion))
			fmt.Printf("Predicted Exhaustion: %s\n", exhaustionDate.Format(time.RFC1123))
		case result.BurnRate != nil && *result.BurnRate >= 6.0:
			fmt.Println("\nStatus: HIGH (Message). Budget is burning significantly faster than expected.")
			fmt.Printf("Forecast: Budget will be empty in %s\n", result.TimeToExhaustion)
			exhaustionDate := time.Now().Add(time.Duration(result.TimeToExhaustion))
			fmt.Printf("Predicted Exhaustion: %s\n", exhaustionDate.Format(time.RFC1123))
		case result.BurnRate != nil && *result.BurnRate > 1.0:
			fmt.Println("\nStatus: WARNING (Ticket). Budget is burning faster than sustainable.")
			fmt.Printf("Forecast: Budget will be empty in %s\n", result.TimeToExhaustion)
			exhaustionDate := time.Now().Add(time.Duration(result.TimeToExhaustion))
			fmt.Printf("Predicted Exhaustion: %s\n", exhaustionDate.Format(time.RFC1123))
		default:
			fmt.Println("\nStatus: Healthy. Budget is being consumed at an acceptable rate.")
		}
		return nil
	})
}
