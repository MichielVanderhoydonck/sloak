package alerting

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
	common "github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/alerting"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

var service port.AlertGeneratorService

func SetService(s port.AlertGeneratorService) {
	service = s
}

func NewAlertTableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alert-table",
		Short: "Generates the standard SRE Alerting Table.",
		Long:  `Generates the Google SRE standard alerting table (Fast/Slow/Sustained burn rates).`,
		Run:   runAlertRulesCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")

	return cmd
}

func runAlertRulesCmd(cmd *cobra.Command, args []string) {
	sloFlag, _ := cmd.Flags().GetFloat64("slo")
	windowStr, _ := cmd.Flags().GetString("window")

	totalWindow, _ := util.ParseTimeWindow(windowStr)
	sloTarget, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	params := domain.GenerateParams{
		TargetSLO:   sloTarget,
		TotalWindow: totalWindow,
	}

	res, err := service.GenerateTable(params)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\n--- SRE Alerting Table ---\n")
	fmt.Printf("SLO: %.3f%% | Window: %s\n\n", res.TargetSLO.Value, util.FormatDuration(res.TotalWindow))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	fmt.Fprintln(w, "THRESHOLD\tSHORT WINDOW\tLONG WINDOW\tBURN RATE\tNOTIFICATION\t")
	fmt.Fprintln(w, "---------\t------------\t-----------\t---------\t------------\t")

	for _, rule := range res.Alerts {
		fmt.Fprintf(w, "%.0f%%\t%s\t%s\t%.2fx\t%s\t\n",
			rule.ConsumptionTarget,
			util.FormatDuration(rule.ShortWindow),
			util.FormatDuration(rule.LongWindow),
			rule.BurnRate,
			rule.NotificationType,
		)
	}

	w.Flush()
	fmt.Println()
}
