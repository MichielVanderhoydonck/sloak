package alerting

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/MichielVanderhoydonck/sloak/cmd/cliutil"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewAlertTableCmd(service domain.TableGenerator) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "alert-table",
		Short:         "Generates the standard SRE Alerting Table.",
		Long:          `Generates the Google SRE standard alerting table (Fast/Slow/Sustained burn rates).`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAlertRulesCmd(cmd, service)
		},
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")

	return cmd
}

func runAlertRulesCmd(cmd *cobra.Command, service domain.TableGenerator) error {
	v, err := cliutil.ScopedConfig(cmd)
	if err != nil {
		return err
	}
	sloFlag := v.GetFloat64("slo")
	windowStr := v.GetString("window")

	totalWindow, err := util.ParseTimeWindow(windowStr)
	if err != nil {
		return fmt.Errorf("parsing window: %w", err)
	}

	sloTarget, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		return err
	}

	params := domain.GenerateParams{
		TargetSLO:   sloTarget,
		TotalWindow: totalWindow,
	}

	res, err := service.GenerateTable(params)
	if err != nil {
		return fmt.Errorf("generating table: %w", err)
	}

	return cliutil.PrintResult(cmd, v.GetString("output"), res, func(out io.Writer) error {
		fmt.Fprintf(out, "\n--- SRE Alerting Table ---\n")
		fmt.Fprintf(out, "SLO: %.3f%% | Window: %s\n\n", res.TargetSLO.Value, res.TotalWindow.String())

		w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

		fmt.Fprintln(w, "THRESHOLD\tSHORT WINDOW\tLONG WINDOW\tBURN RATE\tNOTIFICATION\t")
		fmt.Fprintln(w, "---------\t------------\t-----------\t---------\t------------\t")

		for _, rule := range res.Alerts {
			fmt.Fprintf(w, "%.0f%%\t%s\t%s\t%.2fx\t%s\t\n",
				rule.ConsumptionTarget,
				rule.ShortWindow.String(),
				rule.LongWindow.String(),
				rule.BurnRate,
				rule.NotificationType,
			)
		}

		w.Flush()
		fmt.Fprintln(out)
		return nil
	})
}
