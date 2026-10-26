package disruption

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/MichielVanderhoydonck/sloak/cmd/cliutil"
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
	v, err := cliutil.ScopedConfig(cmd)
	if err != nil {
		return err
	}
	sloFlag := v.GetFloat64("slo")
	windowStr := v.GetString("window")
	costStr := v.GetString("cost")

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

	return cliutil.PrintResult(cmd, v.GetString("output"), res, func(w io.Writer) error {
		fmt.Fprintf(w, "\n--- Disruption Budget Analysis ---\n")
		fmt.Fprintf(w, "SLO Target:       %.3f%%\n", slo.Value)
		fmt.Fprintf(w, "Total Budget:     %s\n", res.TotalErrorBudget)
		fmt.Fprintf(w, "Cost per Event:   %s\n", params.CostPerEvent)
		fmt.Fprintln(w, "----------------------------------")
		fmt.Fprintf(w, "Max Events Total: %d\n", res.MaxDisruptions)
		fmt.Fprintf(w, "Daily Frequency:  %.1f events/day\n", res.DailyDisruptions)

		if res.MaxDisruptions < 1 {
			fmt.Fprintln(w, "\nStatus: BLOCKED. Disruption cost exceeds total budget.")
		} else if res.DailyDisruptions < 1.0 {
			fmt.Fprintln(w, "\nStatus: RISKY. Budget allows less than 1 event per day.")
		} else {
			fmt.Fprintln(w, "\nStatus: SAFE. Budget supports frequent changes.")
		}
		return nil
	})
}
