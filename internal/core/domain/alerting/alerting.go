package alerting

import (
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
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

	FastWindow      time.Duration // Default: 1h
	SlowWindow      time.Duration // Default: 6h
	SustainedWindow time.Duration // Default: 3d
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
