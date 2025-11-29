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
	ConsumptionTarget float64
	LongWindow        time.Duration
	ShortWindow       time.Duration
	BurnRate          float64
	NotificationType  NotificationType
}

type TableResult struct {
	TargetSLO   common.SLOTarget
	TotalWindow time.Duration
	Alerts      []AlertDefinition
}
