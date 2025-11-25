package burnrate

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	burnrateDomain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/burnrate"
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	burnratePort "github.com/MichielVanderhoydonck/sloak/internal/core/port/burnrate"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

var burnRateService burnratePort.BurnRateService

func SetService(svc burnratePort.BurnRateService) {
	burnRateService = svc
}

func NewBurnRateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burnrate",
		Short: "Calculates the current error budget burn rate (consumption speed).",
		Long:  `Calculates the current burn rate against the ideal burn rate (1.0). A value > 1.0 indicates overspending.`,
		Run:   runBurnRateCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "The SLO target percentage (e.g., 99.9)")
	cmd.Flags().StringP("window", "w", "30d", "The total time window (e.g., 30d)")
	cmd.Flags().StringP("elapsed", "t", "15d", "Time elapsed since start of the window.")
	cmd.Flags().StringP("consumed", "c", "1h", "Total error time consumed (e.g., 1h, 10m).")

	return cmd
}

func runBurnRateCmd(cmd *cobra.Command, args []string) {
	sloTarget, _ := cmd.Flags().GetFloat64("slo")
	windowStr, _ := cmd.Flags().GetString("window")
	elapsedStr, _ := cmd.Flags().GetString("elapsed")
	consumedStr, _ := cmd.Flags().GetString("consumed")

	totalWindow, _ := util.ParseTimeWindow(windowStr)
	timeElapsed, _ := util.ParseTimeWindow(elapsedStr)
	errorConsumed, _ := time.ParseDuration(consumedStr)

	sloTargetVO, err := common.NewSLOTarget(sloTarget)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	params := burnrateDomain.CalculationParams{
		TargetSLO:     sloTargetVO,
		TotalWindow:   totalWindow,
		TimeElapsed:   timeElapsed,
		ErrorConsumed: errorConsumed,
	}

	result, err := burnRateService.CalculateBurnRate(params)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Calculation Error: %v\n", err)
		return
	}

	fmt.Printf("\n--- Burn Rate Analysis ---\n")
	fmt.Printf("SLO Target: %.3f%%\n", sloTarget)

	fmt.Printf("Time Window: %s\n", util.FormatDuration(params.TotalWindow))

	fmt.Printf("Total Error Budget: %s\n", util.FormatDuration(result.TotalErrorBudget))

	fmt.Printf("--------------------------\n")
	fmt.Printf("Budget Consumed: %.2f%% (Time Elapsed: %.2f%%)\n",
		result.BudgetConsumed,
		(float64(timeElapsed)/float64(totalWindow))*100.0)
	fmt.Printf("Burn Rate: %.2fx\n", result.BurnRate)
	fmt.Printf("Budget Remaining: %s\n", util.FormatDuration(result.BudgetRemaining))

	if result.BurnRate > 1.0 {
		fmt.Println("\nStatus: CRITICAL! Budget is burning faster than expected.")
		fmt.Printf("Forecast: Budget will be empty in %s\n", util.FormatDuration(result.TimeToExhaustion))
		exhaustionDate := time.Now().Add(result.TimeToExhaustion)
		fmt.Printf("Predicted Exhaustion: %s\n", exhaustionDate.Format(time.RFC1123))
	} else if result.IsInfinite {
		fmt.Println("\nStatus: Excellent! No error budget consumed.")
	} else {
		fmt.Println("\nStatus: Healthy. Budget is being consumed at an acceptable rate.")
	}
}
