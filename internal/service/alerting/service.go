package alerting

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
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

type AlertGeneratorService struct {
	engine domain.RenderEngine
}

var _ domain.Service = (*AlertGeneratorService)(nil)

func NewAlertGeneratorService(engine domain.RenderEngine) domain.Service {
	return &AlertGeneratorService{engine: engine}
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

func (s *AlertGeneratorService) RenderTemplate(params domain.GenerateParams, config map[string]any, templateContent string) (string, error) {
	if s.engine == nil {
		return "", errors.New("render engine is not configured")
	}

	// 1. Generate the table result first to get the windows and burn rates
	tableRes, err := s.GenerateTable(params)
	if err != nil {
		return "", fmt.Errorf("failed to generate alerting table: %w", err)
	}

	// 2. Prepare the AlertingContext DTO
	budget := params.TargetSLO.ErrorBudgetRatio()
	recordingWindows := collectRecordingWindows(tableRes.Alerts)

	contextData := domain.AlertingContext{
		TargetSLO:          params.TargetSLO.Value,
		ErrorBudgetRatio:   budget,
		TotalWindowSeconds: int64(params.TotalWindow.Seconds()),
		Windows:            make([]domain.AlertingWindow, len(tableRes.Alerts)),
		RecordingWindows:   recordingWindows,
	}

	for i, alert := range tableRes.Alerts {
		contextData.Windows[i] = domain.AlertingWindow{
			LongWindow:       formatPrometheusDuration(time.Duration(alert.LongWindow)),
			ShortWindow:      formatPrometheusDuration(time.Duration(alert.ShortWindow)),
			BurnRate:         alert.BurnRate,
			NotificationType: strings.ToLower(string(alert.NotificationType)),
		}
	}

	return s.engine.EvaluateAndRender(contextData, config, params, templateContent)
}

func collectRecordingWindows(alerts []domain.AlertDefinition) []string {
	uniqueWindowsMap := make(map[time.Duration]string)
	for _, rule := range alerts {
		dLong := time.Duration(rule.LongWindow)
		dShort := time.Duration(rule.ShortWindow)
		uniqueWindowsMap[dLong] = formatPrometheusDuration(dLong)
		uniqueWindowsMap[dShort] = formatPrometheusDuration(dShort)
	}
	sortedDurations := make([]time.Duration, 0, len(uniqueWindowsMap))
	for d := range uniqueWindowsMap {
		sortedDurations = append(sortedDurations, d)
	}
	sort.Slice(sortedDurations, func(i, j int) bool {
		return sortedDurations[i] < sortedDurations[j]
	})
	recordingWindows := make([]string, len(sortedDurations))
	for i, d := range sortedDurations {
		recordingWindows[i] = uniqueWindowsMap[d]
	}
	return recordingWindows
}

func formatPrometheusDuration(d time.Duration) string {
	if d.Seconds() == 0 {
		return "0s"
	}
	if d%(24*time.Hour) == 0 {
		return fmt.Sprintf("%dd", d/(24*time.Hour))
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", d/time.Hour)
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", d/time.Minute)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
