package alerting

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	cueyaml "cuelang.org/go/encoding/yaml"
	"github.com/prometheus/prometheus/promql/parser"

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
	if err := validateDatadogConfig(params, config); err != nil {
		return "", err
	}
	if err := validateQueryConfig(config); err != nil {
		return "", err
	}

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

	cfgCopy := make(map[string]any)
	for k, v := range config {
		cfgCopy[k] = v
	}
	if err := normalizeAggregationLabels(cfgCopy); err != nil {
		return "", err
	}

	contextData := domain.AlertingContext{
		TargetSLO:          params.TargetSLO.Value,
		ErrorBudgetRatio:   budget,
		TotalWindowSeconds: int64(params.TotalWindow.Seconds()),
		Windows:            make([]domain.AlertingWindow, len(tableRes.Alerts)),
		RecordingWindows:   recordingWindows,
		Config:             cfgCopy,
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

	// Validate rendered PromQL expressions in 'expr' fields directly on the CUE value before encoding.
	// This covers both JSON and YAML output formats, handles arbitrary rule nesting, and avoids decoding twice.
	if err := validateCueExprs(outputVal); err != nil {
		return "", err
	}

	// Check if JSON format was requested
	format := strings.ToLower(strings.TrimSpace(params.Format))

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

func validateCueExprs(val cue.Value) error {
	switch val.Kind() {
	case cue.StructKind:
		iter, err := val.Fields()
		if err != nil {
			return nil
		}
		for iter.Next() {
			sel := iter.Selector()
			child := iter.Value()
			if sel.Unquoted() == "expr" && child.Kind() == cue.StringKind {
				exprStr, err := child.String()
				if err == nil && strings.TrimSpace(exprStr) != "" {
					if _, err := parsePromQLExpr(exprStr); err != nil {
						return fmt.Errorf("rendered invalid PromQL expression %q: %w", exprStr, err)
					}
				}
			}
			if err := validateCueExprs(child); err != nil {
				return err
			}
		}
	case cue.ListKind:
		iter, err := val.List()
		if err != nil {
			return nil
		}
		for iter.Next() {
			if err := validateCueExprs(iter.Value()); err != nil {
				return err
			}
		}
	}
	return nil
}

func parsePromQLExpr(input string) (parser.Expr, error) {
	p := parser.NewParser(parser.Options{})
	return p.ParseExpr(input)
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

func validateDatadogConfig(params domain.GenerateParams, config map[string]any) error {
	info, isBuiltIn := templates.GetBuiltInInfo(params.Template)
	isDatadog := (isBuiltIn && info.FileName == "datadog.cue") || strings.EqualFold(strings.TrimSuffix(filepath.Base(params.Template), ".cue"), "datadog")
	if !isDatadog {
		return nil
	}
	if config != nil {
		if tf, ok := config["timeframe"].(string); ok && tf != "" {
			if tf != "7d" && tf != "30d" && tf != "90d" {
				return fmt.Errorf("datadog SLO timeframe %q is invalid: must be 7d, 30d, or 90d", tf)
			}
			return nil
		}
	}
	if len(info.AllowedWindows) > 0 {
		var matched bool
		for _, w := range info.AllowedWindows {
			if params.TotalWindow == w {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("datadog SLOs only support 7d, 30d, or 90d timeframes (window is %s); use --set timeframe=7d|30d|90d to override explicitly", util.FormatDuration(params.TotalWindow))
		}
	} else {
		winSec := params.TotalWindow.Seconds()
		if winSec != 7*86400 && winSec != 30*86400 && winSec != 90*86400 {
			return fmt.Errorf("datadog SLOs only support 7d, 30d, or 90d timeframes (window is %s); use --set timeframe=7d|30d|90d to override explicitly", util.FormatDuration(params.TotalWindow))
		}
	}
	return nil
}

func validateQueryPart(name, q string) error {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	hasPlaceholder := strings.Contains(q, "$window") || strings.Contains(q, "{{window}}") || strings.Contains(q, "[window]")
	if hasPlaceholder {
		sub := strings.ReplaceAll(q, "$window", "1m")
		sub = strings.ReplaceAll(sub, "{{window}}", "1m")
		sub = strings.ReplaceAll(sub, "[window]", "[1m]")
		if _, err := parsePromQLExpr(sub); err != nil {
			return fmt.Errorf("%s %q is not valid PromQL: %w", name, q, err)
		}
		return nil
	}

	if strings.HasPrefix(q, "vector(") {
		if _, err := parsePromQLExpr(q); err != nil {
			return fmt.Errorf("%s %q is not valid PromQL: %w", name, q, err)
		}
		return nil
	}

	expr, err := parsePromQLExpr(q)
	if err != nil {
		return fmt.Errorf("%s %q is not valid PromQL: %w", name, q, err)
	}

	var hasFixedRange bool
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		switch node.(type) {
		case *parser.MatrixSelector, *parser.SubqueryExpr:
			hasFixedRange = true
			return errors.New("stop")
		}
		return nil
	})
	if hasFixedRange {
		return fmt.Errorf("%s %q contains a fixed range selector without a window placeholder; queries with a range selector must use $window or {{window}}", name, q)
	}

	if _, ok := expr.(*parser.VectorSelector); !ok {
		return fmt.Errorf("%s %q is a complex PromQL expression (%T), not a bare metric selector; please specify a range window placeholder such as [$window] or use subquery syntax", name, q, expr)
	}
	return nil
}

func validateQueryConfig(config map[string]any) error {
	if config == nil {
		return nil
	}
	errQ, _ := config["errors_query"].(string)
	totQ, _ := config["total_query"].(string)
	errorRatioQ, _ := config["error_ratio_query"].(string)
	sliQ, _ := config["sli_query"].(string)

	errQ = strings.TrimSpace(errQ)
	totQ = strings.TrimSpace(totQ)
	errorRatioQ = strings.TrimSpace(errorRatioQ)
	sliQ = strings.TrimSpace(sliQ)

	// Determine effective ratio query without mutating the caller's config map
	effectiveRatioQ := errorRatioQ
	if effectiveRatioQ == "" {
		effectiveRatioQ = sliQ
	}

	if effectiveRatioQ != "" {
		if err := validateQueryPart("error_ratio_query", effectiveRatioQ); err != nil {
			return err
		}
	}
	if errQ != "" {
		if err := validateQueryPart("errors_query", errQ); err != nil {
			return err
		}
	}
	if totQ != "" {
		if err := validateQueryPart("total_query", totQ); err != nil {
			return err
		}
	}

	if effectiveRatioQ != "" && (errQ != "" || totQ != "") {
		return fmt.Errorf("error_ratio_query cannot be combined with errors_query or total_query; use errors_query + total_query for counter metrics, or error_ratio_query for precomputed ratios")
	}

	if errQ != "" && totQ == "" && effectiveRatioQ == "" {
		if !strings.HasPrefix(errQ, "vector(") {
			return fmt.Errorf("errors_query requires total_query to compute the error ratio; if your query is already an error ratio, use error_ratio_query instead")
		}
	}
	if totQ != "" && errQ == "" && effectiveRatioQ == "" {
		return fmt.Errorf("total_query provided without errors_query; both errors_query and total_query are required to compute the error ratio")
	}

	return nil
}

var validLabelRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func validateLabelIdentifier(label string) error {
	if !validLabelRegex.MatchString(label) {
		return fmt.Errorf("invalid aggregation label %q: label names must match ^[a-zA-Z_][a-zA-Z0-9_]*$", label)
	}
	return nil
}

func normalizeAggregationLabels(config map[string]any) error {
	if config == nil {
		return nil
	}
	val, ok := config["aggregation_labels"]
	if !ok {
		config["aggregation_labels"] = "job"
		return nil
	}
	switch v := val.(type) {
	case string:
		parts := strings.Split(v, ",")
		var cleaned []string
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				if err := validateLabelIdentifier(s); err != nil {
					return err
				}
				cleaned = append(cleaned, s)
			}
		}
		if len(cleaned) == 0 && strings.TrimSpace(v) == "" {
			config["aggregation_labels"] = ""
		} else {
			config["aggregation_labels"] = strings.Join(cleaned, ", ")
		}
	case []string:
		var cleaned []string
		for _, p := range v {
			if s := strings.TrimSpace(p); s != "" {
				if err := validateLabelIdentifier(s); err != nil {
					return err
				}
				cleaned = append(cleaned, s)
			}
		}
		config["aggregation_labels"] = strings.Join(cleaned, ", ")
	case []any:
		var items []string
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				trimmed := strings.TrimSpace(s)
				if err := validateLabelIdentifier(trimmed); err != nil {
					return err
				}
				items = append(items, trimmed)
			}
		}
		config["aggregation_labels"] = strings.Join(items, ", ")
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" {
			if err := validateLabelIdentifier(s); err != nil {
				return err
			}
		}
		config["aggregation_labels"] = s
	}
	return nil
}
