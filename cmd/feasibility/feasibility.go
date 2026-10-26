package feasibility

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/MichielVanderhoydonck/sloak/cmd/cliutil"
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
	v, err := cliutil.ScopedConfig(cmd)
	if err != nil {
		return err
	}
	sloFlag := v.GetFloat64("slo")
	mttrStr := v.GetString("mttr")

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

	return cliutil.PrintResult(cmd, v.GetString("output"), res, func(w io.Writer) error {
		fmt.Fprintf(w, "\n--- SLO Feasibility Analysis ---\n")
		fmt.Fprintf(w, "Target SLO:   %.3f%%\n", res.TargetSLO.Value)
		fmt.Fprintf(w, "Average MTTR: %s\n\n", res.MTTR)

		fmt.Fprintln(w, "--- Operational Reality ---")
		fmt.Fprintf(w, "To maintain %.3f%% with a %s response time:\n\n", res.TargetSLO.Value, res.MTTR)

		fmt.Fprintln(w, "Max Incident Frequency:")
		fmt.Fprintf(w, "  • Per Year:    %.1f incidents\n", res.IncidentsPerYear)
		fmt.Fprintf(w, "  • Per Quarter: %.1f incidents\n", res.IncidentsPerQuarter)
		fmt.Fprintf(w, "  • Per Month:   %.1f incidents\n\n", res.IncidentsPerMonth)

		fmt.Fprintln(w, "Required Reliability (MTBF):")
		fmt.Fprintf(w, "  • Systems must run for %s without failure.\n\n", res.RequiredMTBF)

		// Simple Status Logic
		fmt.Fprint(w, "Status: ")
		if res.IncidentsPerQuarter < 1.0 {
			fmt.Fprintln(w, "EXTREMELY HARD")
			fmt.Fprintln(w, "You cannot afford even one major incident per quarter.")
		} else if res.IncidentsPerQuarter < 3.0 {
			fmt.Fprintln(w, "CHALLENGING")
			fmt.Fprintln(w, "Allows ~1 incident per month. Tight but possible.")
		} else {
			fmt.Fprintln(w, "REALISTIC")
			fmt.Fprintln(w, "Allows multiple incidents per quarter.")
		}
		fmt.Fprintln(w)
		return nil
	})
}
