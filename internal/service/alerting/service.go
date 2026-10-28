package alerting

import (
	"errors"
	"fmt"
	"math"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

const (
	ShortWindowRatio  = 12.0
	FastWindowRatio   = 720.0
	SlowWindowRatio   = 120.0
	SustainedRatio    = 10.0
	MinAlertingWindow = 7 * 24 * time.Hour
	MinShortWindow    = 5 * time.Minute
)

type AlertGeneratorService struct{}

var _ domain.Service = (*AlertGeneratorService)(nil)

func NewAlertGeneratorService() domain.Service {
	return &AlertGeneratorService{}
}

func (s *AlertGeneratorService) GenerateTable(params domain.GenerateParams) (domain.TableResult, error) {
	if params.TotalWindow <= 0 {
		return domain.TableResult{}, errors.New("total window must be greater than zero")
	}
	if params.TotalWindow < MinAlertingWindow {
		return domain.TableResult{}, errors.New("total window must be at least 7d for multi-window multi-burn-rate alerting")
	}
	if err := params.TargetSLO.Validate(); err != nil {
		return domain.TableResult{}, err
	}

	// For canonical monthly windows (28d to 31d), use standard Google SRE Workbook windows and burn rates.
	isCanonicalMonthly := params.TotalWindow >= 28*24*time.Hour && params.TotalWindow <= 31*24*time.Hour

	createRule := func(consumptionThreshold float64, windowRatio float64, notif domain.NotificationType, canonicalLong, canonicalShort time.Duration, canonicalBurn float64) (domain.AlertDefinition, error) {
		var longWindow, shortWindow time.Duration
		var burnFactor float64

		if isCanonicalMonthly {
			longWindow = canonicalLong
			shortWindow = canonicalShort
			burnFactor = canonicalBurn
		} else {
			longWindow = time.Duration(math.Round(float64(params.TotalWindow) / windowRatio)).Round(time.Second)
			if longWindow >= params.TotalWindow {
				return domain.AlertDefinition{}, fmt.Errorf("alert long window (%v) must be smaller than total window (%v)", longWindow, params.TotalWindow)
			}
			burnFactor = (float64(params.TotalWindow) / float64(longWindow)) * consumptionThreshold
			shortWindow = time.Duration(math.Round(float64(longWindow) / ShortWindowRatio)).Round(time.Second)
			// Floor short window to MinShortWindow (5m) to ensure Prometheus scrape interval reliability
			if shortWindow < MinShortWindow {
				shortWindow = MinShortWindow
			}
			if shortWindow >= longWindow {
				shortWindow = longWindow / 2
			}
			if burnFactor < 1.0 {
				return domain.AlertDefinition{}, fmt.Errorf("resulting burn rate (%.2fx) is less than 1.0x for %s alert", burnFactor, notif)
			}
		}

		if shortWindow <= 0 {
			return domain.AlertDefinition{}, fmt.Errorf("alert short window (%v) must be greater than zero", shortWindow)
		}

		actualConsumption := (burnFactor * float64(longWindow) / float64(params.TotalWindow)) * 100.0

		return domain.AlertDefinition{
			ConsumptionTarget:  util.RoundValue(actualConsumption),
			LongWindow:         util.Duration(longWindow),
			LongWindowSeconds:  longWindow.Seconds(),
			ShortWindow:        util.Duration(shortWindow),
			ShortWindowSeconds: shortWindow.Seconds(),
			BurnRate:           util.RoundValue(burnFactor),
			NotificationType:   notif,
		}, nil
	}

	fastRule, err := createRule(0.02, FastWindowRatio, domain.Page, 1*time.Hour, 5*time.Minute, 14.4)
	if err != nil {
		return domain.TableResult{}, err
	}
	slowRule, err := createRule(0.05, SlowWindowRatio, domain.Message, 6*time.Hour, 30*time.Minute, 6.0)
	if err != nil {
		return domain.TableResult{}, err
	}
	sustainedRule, err := createRule(0.10, SustainedRatio, domain.Ticket, 3*24*time.Hour, 6*time.Hour, 1.0)
	if err != nil {
		return domain.TableResult{}, err
	}

	rules := []domain.AlertDefinition{
		fastRule,
		slowRule,
		sustainedRule,
	}

	return domain.TableResult{
		TargetSLO:          params.TargetSLO,
		TotalWindow:        util.Duration(params.TotalWindow),
		TotalWindowSeconds: params.TotalWindow.Seconds(),
		Alerts:             rules,
	}, nil
}
