package alerting

import (
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	"time"
)

type AlertSeverity string

const (
	SeverityPage   AlertSeverity = "critical"
	SeverityTicket AlertSeverity = "warning"
)

type GenerateParams struct {
	TargetSLO   common.SLOTarget
	TotalWindow time.Duration

	// Target Time-to-Exhaustion: "Page me if budget will be gone in X"
	PageTTE   time.Duration
	TicketTTE time.Duration
}

type AlertRule struct {
	Severity        AlertSeverity
	BurnRate        float64
	RecallWindow    time.Duration // The "Long" window (e.g. 1h)
	PrecisionWindow time.Duration // The "Short" window (e.g. 5m) to prevent flapping
	BudgetConsumed  float64       // % of budget consumed in the recall window
}

type Result struct {
	PageRule   AlertRule
	TicketRule AlertRule
}
