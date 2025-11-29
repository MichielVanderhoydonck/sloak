package alerting

import (
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/alerting"
)

const (
	ShortWindowRatio = 12.0
	WinFast          = 1 * time.Hour
	WinSlow          = 6 * time.Hour
	WinSustained     = 72 * time.Hour
)

var _ port.AlertGeneratorService = (*AlertGeneratorServiceImpl)(nil)

type AlertGeneratorServiceImpl struct{}

func NewAlertGeneratorService() port.AlertGeneratorService {
	return &AlertGeneratorServiceImpl{}
}

func (s *AlertGeneratorServiceImpl) GenerateTable(params domain.GenerateParams) (domain.TableResult, error) {

	createRule := func(consumptionThreshold float64, longWindow time.Duration, notif domain.NotificationType) domain.AlertDefinition {
		consumptionWindows := float64(params.TotalWindow) / float64(longWindow)
		burnFactor := consumptionWindows * consumptionThreshold
		shortWindow := time.Duration(math.Round(float64(longWindow) / ShortWindowRatio))

		return domain.AlertDefinition{
			ConsumptionTarget: consumptionThreshold * 100,
			LongWindow:        longWindow,
			ShortWindow:       shortWindow,
			BurnRate:          burnFactor,
			NotificationType:  notif,
		}
	}

	rules := []domain.AlertDefinition{
		createRule(0.02, WinFast, domain.Page),
		createRule(0.05, WinSlow, domain.Message),
		createRule(0.10, WinSustained, domain.Ticket),
	}

	return domain.TableResult{
		TargetSLO:   params.TargetSLO,
		TotalWindow: params.TotalWindow,
		Alerts:      rules,
	}, nil
}
