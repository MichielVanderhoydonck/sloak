package alerting

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"gopkg.in/yaml.v3"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
)

const (
	ShortWindowRatio = 12.0
	FastWindowRatio  = 720.0
	SlowWindowRatio  = 120.0
	SustainedRatio   = 10.0
)

type AlertGeneratorService struct{}

var _ domain.Service = (*AlertGeneratorService)(nil)

func NewAlertGeneratorService() *AlertGeneratorService {
	return &AlertGeneratorService{}
}

func (s *AlertGeneratorService) GenerateTable(params domain.GenerateParams) (domain.TableResult, error) {
	if params.TotalWindow <= 0 {
		return domain.TableResult{}, errors.New("total window must be greater than zero")
	}
	if params.TotalWindow < 24*time.Hour {
		return domain.TableResult{}, errors.New("total window must be at least 24h for multi-window multi-burn-rate alerting")
	}
	if params.TargetSLO.Value <= 0 || params.TargetSLO.Value >= 100 {
		return domain.TableResult{}, errors.New("SLO target must be strictly between 0 and 100")
	}

	createRule := func(consumptionThreshold float64, windowRatio float64, notif domain.NotificationType) (domain.AlertDefinition, error) {
		longWindow := time.Duration(math.Round(float64(params.TotalWindow) / windowRatio)).Round(time.Second)
		if longWindow >= params.TotalWindow {
			return domain.AlertDefinition{}, fmt.Errorf("alert long window (%v) must be smaller than total window (%v)", longWindow, params.TotalWindow)
		}
		consumptionWindows := float64(params.TotalWindow) / float64(longWindow)
		burnFactor := consumptionWindows * consumptionThreshold
		if burnFactor < 1.0 {
			return domain.AlertDefinition{}, fmt.Errorf("resulting burn rate (%.2fx) is less than 1.0x for %s alert", burnFactor, notif)
		}
		shortWindow := time.Duration(math.Round(float64(longWindow) / ShortWindowRatio)).Round(time.Second)
		if shortWindow <= 0 {
			return domain.AlertDefinition{}, fmt.Errorf("alert short window (%v) must be greater than zero", shortWindow)
		}

		return domain.AlertDefinition{
			ConsumptionTarget:  consumptionThreshold * 100,
			LongWindow:         longWindow,
			LongWindowSeconds:  longWindow.Seconds(),
			ShortWindow:        shortWindow,
			ShortWindowSeconds: shortWindow.Seconds(),
			BurnRate:           burnFactor,
			NotificationType:   notif,
		}, nil
	}

	fastRule, err := createRule(0.02, FastWindowRatio, domain.Page)
	if err != nil {
		return domain.TableResult{}, err
	}
	slowRule, err := createRule(0.05, SlowWindowRatio, domain.Message)
	if err != nil {
		return domain.TableResult{}, err
	}
	sustainedRule, err := createRule(0.10, SustainedRatio, domain.Ticket)
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
		TotalWindow:        params.TotalWindow,
		TotalWindowSeconds: params.TotalWindow.Seconds(),
		Alerts:             rules,
	}, nil
}

func (s *AlertGeneratorService) RenderTemplate(params domain.GenerateParams, config map[string]any, templateContent string) (string, error) {
	// 1. Generate the table result first to get the windows and burn rates
	tableRes, err := s.GenerateTable(params)
	if err != nil {
		return "", fmt.Errorf("failed to generate alerting table: %w", err)
	}

	// 2. Prepare the AlertingContext DTO
	budget := 1 - (params.TargetSLO.Value / 100)
	budget = math.Round(budget*1e10) / 1e10

	contextData := domain.AlertingContext{
		TargetSLO:          params.TargetSLO.Value,
		ErrorBudgetRatio:   budget,
		TotalWindowSeconds: params.TotalWindow.Seconds(),
		Windows:            make([]domain.AlertingWindow, len(tableRes.Alerts)),
		Config:             config,
	}

	for i, alert := range tableRes.Alerts {
		contextData.Windows[i] = domain.AlertingWindow{
			LongWindow:       formatPrometheusDuration(alert.LongWindow),
			ShortWindow:      formatPrometheusDuration(alert.ShortWindow),
			BurnRate:         alert.BurnRate,
			NotificationType: strings.ToLower(string(alert.NotificationType)),
		}
	}

	// 3. CUE processing
	ctx := cuecontext.New()
	dataValue := ctx.Encode(contextData)
	if dataValue.Err() != nil {
		return "", fmt.Errorf("failed to encode data to CUE: %w", dataValue.Err())
	}

	tplValue := ctx.CompileString(templateContent)
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
