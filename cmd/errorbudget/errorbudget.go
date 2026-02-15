package errorbudget

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	errorbudgetDomain "github.com/MichielVanderhoydonck/sloak/internal/domain/errorbudget"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type Service interface {
	CalculateBudget(params errorbudgetDomain.CalculationParams) (errorbudgetDomain.BudgetResult, error)
}

var service Service

func SetService(svc Service) {
	service = svc
}

func NewErrorBudgetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "errorbudget",
		Short: "Calculates the total error budget (time) for a given SLO.",
		Long:  `Calculates the allowed failure time based on an SLO percentage and a time window.`,
		Run:   runErrorBudgetCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "The SLO target percentage (e.g., 99.9)")
	cmd.Flags().StringP("window", "w", "30d", "The time window (e.g., 7d, 30d, 1y)")

	return cmd
}

func runErrorBudgetCmd(cmd *cobra.Command, args []string) {
	sloTarget, _ := cmd.Flags().GetFloat64("slo")
	windowStr, _ := cmd.Flags().GetString("window")

	timeWindow, err := util.ParseTimeWindow(windowStr)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	sloTargetVO, err := common.NewSLOTarget(sloTarget)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	params := errorbudgetDomain.CalculationParams{
		TargetSLO:  sloTargetVO,
		TimeWindow: util.Duration(timeWindow),
	}

	result, err := service.CalculateBudget(params)

	if err != nil {
		fmt.Printf("Calculation Error: %v\n", err)
		return
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		}
		return
	}

	fmt.Printf("\n--- Error Budget Calculation ---\n")
	fmt.Printf("SLO Target: %.3f%%\n", result.TargetSLO.Value)
	fmt.Printf("Time Window: %s\n", result.TotalDuration)
	fmt.Printf("--------------------------------\n")
	fmt.Printf("Error Budget: %.4f%% of time\n", result.ErrorBudget)
	fmt.Printf("Allowed Downtime: %s\n", result.AllowedError)
}
