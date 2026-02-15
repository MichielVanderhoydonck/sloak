package feasibility

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/feasibility"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type Service interface {
	CalculateFeasibility(params domain.FeasibilityParams) (domain.FeasibilityResult, error)
}

var service Service

func SetService(s Service) {
	service = s
}

func NewFeasibilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feasibility",
		Short: "Calculates if an SLO is realistic given your MTTR.",
		Long:  `Analyzes whether your target SLO is mathematically feasible based on your team's average Mean Time To Recover (MTTR).`,
		Run:   runFeasibilityCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("mttr", "m", "30m", "Mean Time To Recover (average incident duration)")

	return cmd
}

func runFeasibilityCmd(cmd *cobra.Command, args []string) {
	sloFlag, _ := cmd.Flags().GetFloat64("slo")
	mttrStr, _ := cmd.Flags().GetString("mttr")

	mttr, _ := util.ParseTimeWindow(mttrStr)
	slo, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	params := domain.FeasibilityParams{
		TargetSLO: slo,
		MTTR:      util.Duration(mttr),
	}

	res, err := service.CalculateFeasibility(params)
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
}
