package alerting

import (
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

type NotificationType string

const (
	Page    NotificationType = "Page"
	Message NotificationType = "Message"
	Ticket  NotificationType = "Ticket"
)

type GenerateParams struct {
	TargetSLO   common.SLOTarget
	TotalWindow time.Duration
}

type AlertDefinition struct {
	ConsumptionTarget  float64          `json:"consumption_target"`
	LongWindow         util.Duration    `json:"long_window"`
	LongWindowSeconds  float64          `json:"long_window_seconds"`
	ShortWindow        util.Duration    `json:"short_window"`
	ShortWindowSeconds float64          `json:"short_window_seconds"`
	BurnRate           float64          `json:"burn_rate"`
	NotificationType   NotificationType `json:"notification_type"`
}

type TableResult struct {
	TargetSLO          common.SLOTarget  `json:"target_slo"`
	TotalWindow        util.Duration     `json:"total_window"`
	TotalWindowSeconds float64           `json:"total_window_seconds"`
	Alerts             []AlertDefinition `json:"alerts"`
}

type AlertingWindow struct {
	LongWindow       string  `json:"long"`
	ShortWindow      string  `json:"short"`
	BurnRate         float64 `json:"burn"`
	NotificationType string  `json:"severity"`
}

type AlertingContext struct {
	TargetSLO          float64          `json:"slo_target"`
	ErrorBudgetRatio   float64          `json:"error_budget_ratio"`
	TotalWindowSeconds float64          `json:"total_window_seconds"`
	Windows            []AlertingWindow `json:"windows"`
	Config             map[string]any   `json:"config,omitempty"`
}

type TableGenerator interface {
	GenerateTable(params GenerateParams) (TableResult, error)
}

type TemplateRenderer interface {
	RenderTemplate(params GenerateParams, config map[string]any, templateContent string) (string, error)
}

type Service interface {
	TableGenerator
	TemplateRenderer
}

// SLOSpec represents a parsed SLO specification from an external definition (such as OpenSLO).
type SLOSpec struct {
	TargetSLO   float64
	TotalWindow time.Duration
	Config      map[string]any
}

// SanitizePrometheusMetricName sanitizes an arbitrary string (such as an OpenSLO DNS-1123 name)
// so that it is a valid Prometheus metric name matching ^[a-zA-Z_:][a-zA-Z0-9_:]*$.
func SanitizePrometheusMetricName(name string) string {
	if name == "" {
		return "slo_errors"
	}
	var b []byte
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == ':' {
			b = append(b, c)
		} else if c >= '0' && c <= '9' {
			if len(b) == 0 {
				b = append(b, '_')
			}
			b = append(b, c)
		} else {
			b = append(b, '_')
		}
	}
	res := string(b)
	if res == "" || res == "_" {
		return "slo_errors"
	}
	return res
}
