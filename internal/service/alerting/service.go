package alerting

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	cueyaml "cuelang.org/go/encoding/yaml"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
	templates "github.com/MichielVanderhoydonck/sloak/templates"
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

func NewAlertGeneratorService() *AlertGeneratorService {
	return &AlertGeneratorService{}
}

func (s *AlertGeneratorService) GenerateTable(params domain.GenerateParams) (domain.TableResult, error) {
	if params.TotalWindow <= 0 {
		return domain.TableResult{}, errors.New("total window must be greater than zero")
	}
	if params.TotalWindow < MinAlertingWindow {
		return domain.TableResult{}, errors.New("total window must be at least 7d for multi-window multi-burn-rate alerting")
	}
	if params.TargetSLO.Value <= 0 || params.TargetSLO.Value >= 100 {
		return domain.TableResult{}, errors.New("SLO target must be strictly between 0 and 100")
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

		return domain.AlertDefinition{
			ConsumptionTarget:  consumptionThreshold * 100,
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
	// 1. Generate the table result first to get the windows and burn rates
	tableRes, err := s.GenerateTable(params)
	if err != nil {
		return "", fmt.Errorf("failed to generate alerting table: %w", err)
	}

	// 2. Prepare the AlertingContext DTO
	budget := 1 - (params.TargetSLO.Value / 100)
	budget = math.Round(budget*1e10) / 1e10

	// Collect unique recording windows in ascending order of duration
	uniqueWindowsMap := make(map[time.Duration]string)
	for _, rule := range tableRes.Alerts {
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

	contextData := domain.AlertingContext{
		TargetSLO:          params.TargetSLO.Value,
		ErrorBudgetRatio:   budget,
		TotalWindowSeconds: int64(params.TotalWindow.Seconds()),
		Windows:            make([]domain.AlertingWindow, len(tableRes.Alerts)),
		RecordingWindows:   recordingWindows,
		Config:             config,
	}

	for i, alert := range tableRes.Alerts {
		contextData.Windows[i] = domain.AlertingWindow{
			LongWindow:       formatPrometheusDuration(time.Duration(alert.LongWindow)),
			ShortWindow:      formatPrometheusDuration(time.Duration(alert.ShortWindow)),
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

	schemaContent, err := templates.GetSchema()
	if err != nil {
		return "", fmt.Errorf("failed to load shared schema: %w", err)
	}

	schemaVal := ctx.CompileString(schemaContent)
	if schemaVal.Err() != nil {
		return "", fmt.Errorf("failed to compile shared schema: %w", schemaVal.Err())
	}

	tplValue := ctx.CompileString(templateContent, cue.Scope(schemaVal))
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

	// Filter out the 'data' field while preserving field declaration order
	syntax := finalValue.Syntax(cue.Final())
	if file, ok := syntax.(*ast.File); ok {
		var filtered []ast.Decl
		for _, decl := range file.Decls {
			if f, ok := decl.(*ast.Field); ok {
				if name, _, err := ast.LabelName(f.Label); err == nil && name == "data" {
					continue
				}
			}
			filtered = append(filtered, decl)
		}
		file.Decls = filtered
	} else if lit, ok := syntax.(*ast.StructLit); ok {
		var filtered []ast.Decl
		for _, elt := range lit.Elts {
			if f, ok := elt.(*ast.Field); ok {
				if name, _, err := ast.LabelName(f.Label); err == nil && name == "data" {
					continue
				}
			}
			filtered = append(filtered, elt)
		}
		lit.Elts = filtered
	}

	var outputVal cue.Value
	switch n := syntax.(type) {
	case *ast.File:
		outputVal = ctx.BuildFile(n)
	case *ast.StructLit:
		outputVal = ctx.BuildExpr(n)
	default:
		outputVal = finalValue
	}

	if outputVal.Err() != nil {
		return "", fmt.Errorf("failed to build output CUE value: %w", outputVal.Err())
	}

	// Check if JSON format was requested
	format := strings.ToLower(strings.TrimSpace(params.Format))
	if format == "" {
		if fmtVal, ok := config["format"].(string); ok {
			format = strings.ToLower(strings.TrimSpace(fmtVal))
		}
	}

	if format == "json" {
		jsonBytes, err := outputVal.MarshalJSON()
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, jsonBytes, "", "  "); err == nil {
			return pretty.String() + "\n", nil
		}
		return string(jsonBytes) + "\n", nil
	}

	// Default to YAML (preserving field order)
	yamlBytes, err := cueyaml.Encode(outputVal)
	if err != nil {
		return "", fmt.Errorf("failed to encode YAML: %w", err)
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
