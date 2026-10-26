package convert

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichielVanderhoydonck/sloak/cmd/cliutil"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/convert"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewConvertCmd(service domain.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "convert",
		Short:         "Converts between Availability % and Downtime Duration.",
		Long:          `Converts "Nines" (e.g. 99.9%) to allowed downtime per day/month/year, or converts a specific downtime duration back into a percentage.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConvertCmd(cmd, service)
		},
	}

	cmd.Flags().Float64("nines", -1, "Availability Percentage (e.g. 99.95)")
	cmd.Flags().String("downtime", "", "Downtime duration (e.g. 15m)")
	cmd.Flags().String("window", "30d", "Window size (only used with --downtime)")
	cmd.Flags().StringP("output", "o", "", "Output format (e.g., json)")

	return cmd
}

func runConvertCmd(cmd *cobra.Command, service domain.Service) error {
	v, err := cliutil.ScopedConfig(cmd)
	if err != nil {
		return err
	}
	nines := v.GetFloat64("nines")
	downtimeStr := v.GetString("downtime")
	windowStr := v.GetString("window")

	var params domain.ConversionParams

	if nines != -1 && downtimeStr != "" {
		return fmt.Errorf("please provide EITHER --nines OR --downtime, not both")
	}

	if nines != -1 {
		params = domain.ConversionParams{
			Mode:  domain.ModeFromNines,
			Nines: nines,
		}
	} else if downtimeStr != "" {
		dt, err := util.ParseTimeWindow(downtimeStr)
		if err != nil {
			return fmt.Errorf("parsing downtime: %w", err)
		}
		win, err := util.ParseTimeWindow(windowStr)
		if err != nil {
			return fmt.Errorf("parsing window: %w", err)
		}
		params = domain.ConversionParams{
			Mode:         domain.ModeFromDowntime,
			Downtime:     dt,
			CustomWindow: win,
		}
	} else {
		return fmt.Errorf("must provide either --nines or --downtime")
	}

	res, err := service.Convert(params)
	if err != nil {
		return fmt.Errorf("calculation error: %w", err)
	}

	return cliutil.PrintResult(cmd, v.GetString("output"), res, func() error {
		fmt.Printf("\n--- Availability Conversion ---\n")
		fmt.Printf("Availability: %.5f%%\n", res.AvailabilityPercent)
		fmt.Printf("--------------------------------\n")
		fmt.Printf("Daily Allowed:     %s\n", util.FormatDuration(res.DailyDowntime))
		fmt.Printf("Weekly Allowed:    %s\n", util.FormatDuration(res.WeeklyDowntime))
		fmt.Printf("Monthly (30d):     %s\n", util.FormatDuration(res.MonthlyDowntime))
		fmt.Printf("Quarterly (90d):   %s\n", util.FormatDuration(res.QuarterlyDowntime))
		fmt.Printf("Yearly (365d):     %s\n", util.FormatDuration(res.YearlyDowntime))
		return nil
	})
}
