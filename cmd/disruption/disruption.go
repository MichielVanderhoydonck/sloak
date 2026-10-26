package disruption

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/disruption"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewDisruptionCmd(service domain.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "max-disruption",
		Short:         "Calculates allowed deployment frequency based on disruption cost.",
		Long:          `Determines how many disruptive events (e.g. deployments, restarts) fit into the error budget.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDisruptionCmd(cmd, service)
		},
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")
	cmd.Flags().String("cost", "10s", "Estimated downtime cost per event")

	return cmd
}

func runDisruptionCmd(cmd *cobra.Command, service domain.Service) error {
	viper.BindPFlags(cmd.Flags())
	sloFlag := viper.GetFloat64("slo")
	windowStr := viper.GetString("window")
	costStr := viper.GetString("cost")

	window, err := util.ParseTimeWindow(windowStr)
	if err != nil {
		return fmt.Errorf("parsing window: %w", err)
	}
	cost, err := util.ParseTimeWindow(costStr)
	if err != nil {
		return fmt.Errorf("parsing cost: %w", err)
	}
	slo, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		return err
	}

	params := domain.CalculationParams{
		TargetSLO:    slo,
		TotalWindow:  util.Duration(window),
		CostPerEvent: util.Duration(cost),
	}

	res, err := service.CalculateCapacity(params)
	if err != nil {
		return fmt.Errorf("calculation error: %w", err)
	}

	outputFlag := viper.GetString("output")
	if outputFlag == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(res); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		return nil
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
	return nil
}
