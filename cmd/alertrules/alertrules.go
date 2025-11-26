package alertrules

import (
	"fmt"
	"time"

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

func NewAlertRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alert-thresholds",
		Short: "Generates multi-window burn rate alert thresholds.",
		Long:  `Calculates the required Burn Rate and Observation Window to trigger alerts based on Time-To-Exhaustion targets.`,
		Run:   runAlertRulesCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")
	cmd.Flags().String("page-tte", "24h", "Target TTE for Critical/Page alerts")
	cmd.Flags().String("ticket-tte", "3d", "Target TTE for Warning/Ticket alerts")

	return cmd
}

func runAlertRulesCmd(cmd *cobra.Command, args []string) {
	sloFlag, _ := cmd.Flags().GetFloat64("slo")
	windowStr, _ := cmd.Flags().GetString("window")
	pageTTEStr, _ := cmd.Flags().GetString("page-tte")
	ticketTTEStr, _ := cmd.Flags().GetString("ticket-tte")

	totalWindow, _ := util.ParseTimeWindow(windowStr)
	pageTTE, _ := util.ParseTimeWindow(pageTTEStr)
	ticketTTE, _ := util.ParseTimeWindow(ticketTTEStr)

	sloTarget, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	params := domain.GenerateParams{
		TargetSLO:   sloTarget,
		TotalWindow: totalWindow,
		PageTTE:     pageTTE,
		TicketTTE:   ticketTTE,
	}

	res, err := service.GenerateThresholds(params)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	printRule := func(r domain.AlertRule, title string) {
		derivedTTE := time.Duration(float64(params.TotalWindow) / r.BurnRate)

		fmt.Printf("\n--- %s ---\n", title)
		fmt.Printf("Target TTE:          %s\n", util.FormatDuration(derivedTTE))
		fmt.Printf("Burn Rate Threshold: %.2fx\n", r.BurnRate)
		fmt.Printf("Observation Window:  %s\n", util.FormatDuration(r.RecallWindow))
		fmt.Printf("Budget Consumption:  %.1f%%\n", r.BudgetConsumed)
		fmt.Printf("Error Rate Thresh:   %.3f%% (of total traffic)\n", r.ErrorRateThreshold)
		fmt.Println("--------------------------------")
		fmt.Printf("Logic: Fire if error rate > %.2fx over %s\n", r.BurnRate, util.FormatDuration(r.RecallWindow))
	}

	fmt.Printf("\n--- Multi-Window Alert Rules ---\n")
	fmt.Printf("SLO Target:  %.3f%%\n", sloTarget.Value)
	fmt.Printf("Time Window: %s\n", util.FormatDuration(totalWindow))

	printRule(res.PageRule, "CRITICAL / Paging")
	printRule(res.TicketRule, "WARNING / Ticket")
}
