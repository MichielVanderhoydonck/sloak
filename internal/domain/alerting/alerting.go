package alerting

import (
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
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
	LongWindow         time.Duration    `json:"long_window"`
	LongWindowSeconds  float64          `json:"long_window_seconds"`
	ShortWindow        time.Duration    `json:"short_window"`
	ShortWindowSeconds float64          `json:"short_window_seconds"`
	BurnRate           float64          `json:"burn_rate"`
	NotificationType   NotificationType `json:"notification_type"`
}

type TableResult struct {
	TargetSLO          common.SLOTarget  `json:"target_slo"`
	TotalWindow        time.Duration     `json:"total_window"`
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
	TargetSLO          float64                `json:"slo_target"`
	ErrorBudgetRatio   float64                `json:"error_budget_ratio"`
	TotalWindowSeconds float64                `json:"total_window_seconds"`
	Windows            []AlertingWindow       `json:"windows"`
	Config             map[string]interface{} `json:"config,omitempty"`
}
