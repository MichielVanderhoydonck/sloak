package alerting

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
	common "github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewPrometheusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prometheus",
		Short: "Generates Prometheus MWMBR alert rules.",
		Run:   runPrometheusCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")

	cmd.Flags().String("metric-name", "slo_errors", "The metric name to use in expressions (e.g. slo_errors)")
	cmd.Flags().String("namespace", "monitoring", "The Kubernetes namespace for the PrometheusRule")
	cmd.Flags().String("rule-labels", "", "Comma-separated labels to add to each rule (e.g. team=sre,service=api)")
	cmd.Flags().String("meta-labels", "role=alert-rules", "Comma-separated labels for the PrometheusRule metadata")
	cmd.Flags().String("runbook-url", "", "Template URL for runbooks")

	return cmd
}

func runPrometheusCmd(cmd *cobra.Command, args []string) {
	sloFlag, _ := cmd.Flags().GetFloat64("slo")
	windowStr, _ := cmd.Flags().GetString("window")

	metricName, _ := cmd.Flags().GetString("metric-name")
	namespace, _ := cmd.Flags().GetString("namespace")
	ruleLabelsStr, _ := cmd.Flags().GetString("rule-labels")
	metaLabelsStr, _ := cmd.Flags().GetString("meta-labels")
	runbookURL, _ := cmd.Flags().GetString("runbook-url")

	totalWindow, err := util.ParseTimeWindow(windowStr)
	if err != nil {
		fmt.Printf("Error parsing window: %v\n", err)
		return
	}

	sloTarget, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	params := domain.GeneratePrometheusParams{
		TargetSLO:   sloTarget,
		TotalWindow: totalWindow,
		Config: domain.PrometheusConfig{
			MetricName: metricName,
			Namespace:  namespace,
			RuleLabels: parseLabels(ruleLabelsStr),
			MetaLabels: parseLabels(metaLabelsStr),
			RunbookURL: runbookURL,
		},
	}

	res, err := service.GeneratePrometheus(params)
	if err != nil {
		fmt.Printf("Error generating prometheus rules: %v\n", err)
		return
	}

	fmt.Print(res)
}

func parseLabels(s string) map[string]string {
	labels := make(map[string]string)
	if s == "" {
		return labels
	}
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return labels
}
