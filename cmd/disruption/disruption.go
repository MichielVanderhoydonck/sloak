package disruption

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/disruption"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type Service interface {
	CalculateCapacity(params domain.CalculationParams) (domain.Result, error)
}

var service Service

func SetService(s Service) {
	service = s
}

func NewDisruptionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "max-disruption",
		Short: "Calculates allowed deployment frequency based on disruption cost.",
		Long:  `Determines how many disruptive events (e.g. deployments, restarts) fit into the error budget.`,
		Run:   runDisruptionCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")
	cmd.Flags().String("cost", "10s", "Estimated downtime cost per event")

	return cmd
}

func runDisruptionCmd(cmd *cobra.Command, args []string) {
	sloFlag, _ := cmd.Flags().GetFloat64("slo")
	windowStr, _ := cmd.Flags().GetString("window")
	costStr, _ := cmd.Flags().GetString("cost")

	window, _ := util.ParseTimeWindow(windowStr)
	cost, _ := util.ParseTimeWindow(costStr)
	slo, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	params := domain.CalculationParams{
		TargetSLO:    slo,
		TotalWindow:  util.Duration(window),
		CostPerEvent: util.Duration(cost),
	}

	res, err := service.CalculateCapacity(params)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(res); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		}
		return
	}

	fmt.Printf("\n--- Disruption Budget Analysis ---\n")
	fmt.Printf("SLO Target:       %.3f%%\n", slo.Value)
	fmt.Printf("Total Budget:     %s\n", res.TotalErrorBudget)
	fmt.Printf("Cost per Event:   %s\n", params.CostPerEvent)
	fmt.Println("----------------------------------")
	fmt.Printf("Max Events Total: %d\n", res.MaxDisruptions)
	fmt.Printf("Daily Frequency:  %.1f events/day\n", res.DailyDisruptions)

	if res.MaxDisruptions < 1 {
		fmt.Println("\nStatus: BLOCKED. Disruption cost exceeds total budget.")
	} else if res.DailyDisruptions < 1.0 {
		fmt.Println("\nStatus: RISKY. Budget allows less than 1 event per day.")
	} else {
		fmt.Println("\nStatus: SAFE. Budget supports frequent changes.")
	}
}
