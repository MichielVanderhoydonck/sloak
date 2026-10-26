package feasibility

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/feasibility"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewFeasibilityCmd(service domain.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "feasibility",
		Short:         "Calculates if an SLO is realistic given your MTTR.",
		Long:          `Analyzes whether your target SLO is mathematically feasible based on your team's average Mean Time To Recover (MTTR).`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFeasibilityCmd(cmd, service)
		},
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("mttr", "m", "30m", "Mean Time To Recover (average incident duration)")

	return cmd
}

func runFeasibilityCmd(cmd *cobra.Command, service domain.Service) error {
	viper.BindPFlags(cmd.Flags())
	sloFlag := viper.GetFloat64("slo")
	mttrStr := viper.GetString("mttr")

	mttr, err := util.ParseTimeWindow(mttrStr)
	if err != nil {
		return fmt.Errorf("parsing mttr: %w", err)
	}
	slo, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		return err
	}

	params := domain.FeasibilityParams{
		TargetSLO: slo,
		MTTR:      util.Duration(mttr),
	}

	res, err := service.CalculateFeasibility(params)
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

	fmt.Printf("\n--- SLO Feasibility Analysis ---\n")
	fmt.Printf("Target SLO:   %.3f%%\n", res.TargetSLO.Value)
	fmt.Printf("Average MTTR: %s\n\n", res.MTTR)

	fmt.Println("--- Operational Reality ---")
	fmt.Printf("To maintain %.3f%% with a %s response time:\n\n", res.TargetSLO.Value, res.MTTR)

	fmt.Println("Max Incident Frequency:")
	fmt.Printf("  • Per Year:    %.1f incidents\n", res.IncidentsPerYear)
	fmt.Printf("  • Per Quarter: %.1f incidents\n", res.IncidentsPerQuarter)
	fmt.Printf("  • Per Month:   %.1f incidents\n\n", res.IncidentsPerMonth)

	fmt.Println("Required Reliability (MTBF):")
	fmt.Printf("  • Systems must run for %s without failure.\n\n", res.RequiredMTBF)

	// Simple Status Logic
	fmt.Print("Status: ")
	if res.IncidentsPerQuarter < 1.0 {
		fmt.Println("EXTREMELY HARD")
		fmt.Println("You cannot afford even one major incident per quarter.")
	} else if res.IncidentsPerQuarter < 3.0 {
		fmt.Println("CHALLENGING")
		fmt.Println("Allows ~1 incident per month. Tight but possible.")
	} else {
		fmt.Println("REALISTIC")
		fmt.Println("Allows multiple incidents per quarter.")
	}
	fmt.Println()
	return nil
}
