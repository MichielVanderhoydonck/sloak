package convert

import (
	"fmt"

	"github.com/spf13/cobra"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/convert"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/convert"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

var service port.ConvertService

func SetService(s port.ConvertService) {
	service = s
}

func NewConvertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Converts between Availability % and Downtime Duration.",
		Long:  `Converts "Nines" (e.g. 99.9%) to allowed downtime per day/month/year, or converts a specific downtime duration back into a percentage.`,
		Run:   runConvertCmd,
	}

	cmd.Flags().Float64("nines", -1, "Availability Percentage (e.g. 99.95)")
	cmd.Flags().String("downtime", "", "Downtime duration (e.g. 15m)")
	cmd.Flags().String("window", "30d", "Window size (only used with --downtime)")

	return cmd
}

func runConvertCmd(cmd *cobra.Command, args []string) {
	nines, _ := cmd.Flags().GetFloat64("nines")
	downtimeStr, _ := cmd.Flags().GetString("downtime")
	windowStr, _ := cmd.Flags().GetString("window")

	var params domain.ConversionParams

	if nines != -1 && downtimeStr != "" {
		fmt.Println("Error: Please provide EITHER --nines OR --downtime, not both.")
		return
	}

	if nines != -1 {
		params = domain.ConversionParams{
			Mode:  domain.ModeFromNines,
			Nines: nines,
		}
	} else if downtimeStr != "" {
		dt, err := util.ParseTimeWindow(downtimeStr)
		if err != nil {
			fmt.Printf("Error parsing downtime: %v\n", err)
			return
		}
		win, err := util.ParseTimeWindow(windowStr)
		if err != nil {
			fmt.Printf("Error parsing window: %v\n", err)
			return
		}
		params = domain.ConversionParams{
			Mode:         domain.ModeFromDowntime,
			Downtime:     dt,
			CustomWindow: win,
		}
	} else {
		fmt.Println("Error: Must provide either --nines or --downtime")
		return
	}

	res, err := service.Convert(params)
	if err != nil {
		fmt.Printf("Calculation Error: %v\n", err)
		return
	}

	fmt.Printf("\n--- Availability Conversion ---\n")
	fmt.Printf("Availability: %.5f%%\n", res.AvailabilityPercent)
	fmt.Printf("--------------------------------\n")
	fmt.Printf("Daily Allowed:     %s\n", util.FormatDuration(res.DailyDowntime))
	fmt.Printf("Weekly Allowed:    %s\n", util.FormatDuration(res.WeeklyDowntime))
	fmt.Printf("Monthly (30d):     %s\n", util.FormatDuration(res.MonthlyDowntime))
	fmt.Printf("Quarterly (90d):   %s\n", util.FormatDuration(res.QuarterlyDowntime))
	fmt.Printf("Yearly (365d):     %s\n", util.FormatDuration(res.YearlyDowntime))
}
