package cue

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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

// Engine implements domain.RenderEngine using CUE and the Prometheus PromQL parser.
type Engine struct{}

var _ domain.RenderEngine = (*Engine)(nil)

func NewEngine() domain.RenderEngine {
	return &Engine{}
}

func (e *Engine) EvaluateAndRender(contextData domain.AlertingContext, config map[string]any, params domain.GenerateParams, templateContent string) (string, error) {
	if err := validateTemplateConstraints(params, config); err != nil {
		return "", err
	}
	if err := validateQueryConfig(config); err != nil {
		return "", err
	}

	cfgCopy := make(map[string]any, len(config))
	for k, v := range config {
		cfgCopy[k] = v
	}
	if err := normalizeAggregationLabels(cfgCopy); err != nil {
		return "", err
	}
	contextData.Config = cfgCopy

	return evaluateAndEncodeCUE(contextData, templateContent, params.Format)
}

func evaluateAndEncodeCUE(contextData domain.AlertingContext, templateContent, rawFormat string) (string, error) {
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
	switch node := syntax.(type) {
	case *ast.File:
		node.Decls = filterDataDecl(node.Decls)
	case *ast.StructLit:
		node.Elts = filterDataDecl(node.Elts)
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
	if err := validateCueExprs(outputVal); err != nil {
		return "", err
	}

	format := strings.ToLower(strings.TrimSpace(rawFormat))
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

	yamlBytes, err := cueyaml.Encode(outputVal)
	if err != nil {
		return "", fmt.Errorf("failed to encode YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func filterDataDecl(decls []ast.Decl) []ast.Decl {
	var filtered []ast.Decl
	for _, decl := range decls {
		if f, ok := decl.(*ast.Field); ok {
			if name, _, err := ast.LabelName(f.Label); err == nil && name == "data" {
				continue
			}
		}
		filtered = append(filtered, decl)
	}
	return filtered
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

func validateTemplateConstraints(params domain.GenerateParams, config map[string]any) error {
	info, ok := templates.GetBuiltInInfo(params.Template)
	if !ok || len(info.AllowedWindows) == 0 {
		return nil
	}

	allowedLabels := make([]string, len(info.AllowedWindows))
	for i, w := range info.AllowedWindows {
		allowedLabels[i] = formatPrometheusDuration(w)
	}

	if info.TimeframeConfig != "" && config != nil {
		if tf, ok := config[info.TimeframeConfig].(string); ok && tf != "" {
			var validOverride bool
			for _, label := range allowedLabels {
				if tf == label {
					validOverride = true
					break
				}
			}
			if !validOverride {
				return fmt.Errorf("%s %s %q is invalid: must be 7d, 30d, or 90d", info.DisplayName, info.TimeframeConfig, tf)
			}
			return nil
		}
	}

	var matched bool
	for _, w := range info.AllowedWindows {
		if params.TotalWindow == w {
			matched = true
			break
		}
	}
	if !matched {
		return fmt.Errorf("%ss only support 7d, 30d, or 90d timeframes (window is %s); use --set %s=%s to override explicitly",
			info.DisplayName, util.FormatDuration(params.TotalWindow), info.TimeframeConfig, strings.Join(allowedLabels, "|"))
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
