package alerting

import (
	"fmt"
	"math"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"gopkg.in/yaml.v3"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/alerting"
	"github.com/MichielVanderhoydonck/sloak/templates"
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
			ConsumptionTarget:  consumptionThreshold * 100,
			LongWindow:         longWindow,
			LongWindowSeconds:  longWindow.Seconds(),
			ShortWindow:        shortWindow,
			ShortWindowSeconds: shortWindow.Seconds(),
			BurnRate:           burnFactor,
			NotificationType:   notif,
		}
	}

	rules := []domain.AlertDefinition{
		createRule(0.02, WinFast, domain.Page),
		createRule(0.05, WinSlow, domain.Message),
		createRule(0.10, WinSustained, domain.Ticket),
	}

	return domain.TableResult{
		TargetSLO:          params.TargetSLO,
		TotalWindow:        params.TotalWindow,
		TotalWindowSeconds: params.TotalWindow.Seconds(),
		Alerts:             rules,
	}, nil
}

func (s *AlertGeneratorServiceImpl) GeneratePrometheus(params domain.GeneratePrometheusParams) (string, error) {
	// 1. Generate the table result first to get the windows and burn rates
	tableParams := domain.GenerateParams{
		TargetSLO:   params.TargetSLO,
		TotalWindow: 30 * 24 * time.Hour, // Standard window for alerting table
	}
	tableRes, err := s.GenerateTable(tableParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate alerting table: %w", err)
	}

	// 2. Prepare the PrometheusMatrix DTO
	budget := 1 - (params.TargetSLO.Value / 100)
	budget = math.Round(budget*1e10) / 1e10

	matrix := domain.PrometheusMatrix{
		ErrorBudgetRatio: budget,
		Alerts:           make([]domain.PrometheusAlert, len(tableRes.Alerts)),
		Config:           params.Config,
	}

	for i, alert := range tableRes.Alerts {
		matrix.Alerts[i] = domain.PrometheusAlert{
			LongWindow:       formatPrometheusDuration(alert.LongWindow),
			ShortWindow:      formatPrometheusDuration(alert.ShortWindow),
			BurnRate:         alert.BurnRate,
			NotificationType: strings.ToLower(string(alert.NotificationType)),
		}
	}

	// 3. CUE processing
	ctx := cuecontext.New()
	dataValue := ctx.Encode(matrix)
	if dataValue.Err() != nil {
		return "", fmt.Errorf("failed to encode data to CUE: %w", dataValue.Err())
	}

	tplValue := ctx.CompileString(templates.PrometheusMWMbr)
	if tplValue.Err() != nil {
		return "", fmt.Errorf("failed to compile CUE template: %w", tplValue.Err())
	}

	finalValue := tplValue.FillPath(cue.ParsePath("data"), dataValue)
	if finalValue.Err() != nil {
		return "", fmt.Errorf("failed to fill CUE data: %w", finalValue.Err())
	}

	if err := finalValue.Validate(cue.Concrete(true)); err != nil {
		return "", fmt.Errorf("failed to validate CUE result: %w", err)
	}

	var result map[string]interface{}
	if err := finalValue.Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode CUE result: %w", err)
	}

	delete(result, "data")

	yamlBytes, err := yaml.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return string(yamlBytes), nil
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
