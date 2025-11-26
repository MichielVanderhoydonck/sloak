package alerting

import (
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/alerting"
)

const (
	PageBudgetSpend   = 0.02
	TicketBudgetSpend = 0.05
	ShortWindowRatio  = 12.0
)

var _ port.AlertGeneratorService = (*AlertGeneratorServiceImpl)(nil)

type AlertGeneratorServiceImpl struct{}

func NewAlertGeneratorService() port.AlertGeneratorService {
	return &AlertGeneratorServiceImpl{}
}

func (s *AlertGeneratorServiceImpl) GenerateThresholds(params domain.GenerateParams) (domain.Result, error) {
	calc := func(severity domain.AlertSeverity, tte time.Duration, spendPct float64) domain.AlertRule {
		burnRate := float64(params.TotalWindow) / float64(tte)
		recallNanos := float64(tte) * spendPct
		recallWindow := time.Duration(math.Round(recallNanos))
		precisionWindow := time.Duration(math.Round(float64(recallWindow) / ShortWindowRatio))

		return domain.AlertRule{
			Severity:        severity,
			BurnRate:        burnRate,
			RecallWindow:    recallWindow,
			PrecisionWindow: precisionWindow,
			BudgetConsumed:  spendPct * 100,
		}
	}

	return domain.Result{
		PageRule:   calc(domain.SeverityPage, params.PageTTE, PageBudgetSpend),
		TicketRule: calc(domain.SeverityTicket, params.TicketTTE, TicketBudgetSpend),
	}, nil
}
