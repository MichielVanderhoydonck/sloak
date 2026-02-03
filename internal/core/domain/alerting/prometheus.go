package alerting

import (
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
)

type PrometheusAlert struct {
	LongWindow       string  `json:"long"`
	ShortWindow      string  `json:"short"`
	BurnRate         float64 `json:"burn"`
	NotificationType string  `json:"severity"`
}

type PrometheusConfig struct {
	MetricName string            `json:"metric_name"`
	RuleLabels map[string]string `json:"rule_labels"`
	MetaLabels map[string]string `json:"meta_labels"`
	Namespace  string            `json:"namespace"`
	RunbookURL string            `json:"runbook_url"`
}

type PrometheusMatrix struct {
	ErrorBudgetRatio float64           `json:"budget"`
	Alerts           []PrometheusAlert `json:"alerts"`
	Config           PrometheusConfig  `json:"config"`
}

type GeneratePrometheusParams struct {
	TargetSLO   common.SLOTarget
	TotalWindow time.Duration
	Config      PrometheusConfig
}
