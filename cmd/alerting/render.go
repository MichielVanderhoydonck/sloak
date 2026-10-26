package alerting

import (
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/MichielVanderhoydonck/sloak/cmd/cliutil"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
	templates "github.com/MichielVanderhoydonck/sloak/templates"
)

func NewRenderCmd(renderService domain.TemplateRenderer, loader domain.SpecLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "render",
		Short:         "Renders observability configuration from BYOT templates.",
		Long:          `Renders actionable configurations by taking a provided BYOT CUE template and injecting generic SLO math.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRenderCmd(cmd, renderService, loader)
		},
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")
	cmd.Flags().StringP("spec", "f", "", "Path to OpenSLO specification file (overrides --slo and --window)")
	cmd.Flags().String("values", "", "Path to custom values file (YAML)")
	cmd.Flags().StringArray("set", []string{}, "Set individual values (e.g. key=val, dotted.path=val, or map=k1=v1,k2=v2)")
	cmd.Flags().StringArray("set-json", []string{}, "Set individual JSON/YAML values (e.g. key={\"foo\":\"bar\"})")
	cmd.Flags().StringP("format", "o", "", "Output format (yaml or json). If not specified, defaults based on template.")

	cmd.Flags().String("template", "", "Path to a BYOT CUE template file or name of built-in template (required)")

	return cmd
}

func runRenderCmd(cmd *cobra.Command, renderService domain.TemplateRenderer, loader domain.SpecLoader) error {
	v, err := cliutil.ScopedConfig(cmd)
	if err != nil {
		return err
	}

	templatePath := v.GetString("template")
	setVals, _ := cmd.Flags().GetStringArray("set")
	if len(setVals) == 0 {
		setVals = v.GetStringSlice("set")
	}
	setJSONVals, _ := cmd.Flags().GetStringArray("set-json")
	if len(setJSONVals) == 0 {
		setJSONVals = v.GetStringSlice("set-json")
	}
	valuesFile := v.GetString("values")
	specFile := v.GetString("spec")

	if templatePath == "" {
		return fmt.Errorf("--template is required")
	}

	sloFlag := v.GetFloat64("slo")
	windowStr := v.GetString("window")

	var sloVal float64 = sloFlag
	var totalWindow time.Duration
	var openSLOConfig map[string]any

	if specFile != "" {
		if loader == nil {
			return fmt.Errorf("spec loader is not configured")
		}
		spec, err := loader.LoadSpec(specFile)
		if err != nil {
			return fmt.Errorf("parsing OpenSLO spec: %w", err)
		}
		sloVal = spec.TargetSLO
		totalWindow = spec.TotalWindow
		openSLOConfig = spec.Config
	} else {
		totalWindow, err = util.ParseTimeWindow(windowStr)
		if err != nil {
			return fmt.Errorf("parsing window: %w", err)
		}
	}

	sloTarget, err := common.NewSLOTarget(sloVal)
	if err != nil {
		return err
	}

	// Build the generic configuration map
	configData := make(map[string]any)

	// 1. Read from OpenSLO metadata if parsed
	maps.Copy(configData, openSLOConfig)

	// 2. Read from values file (overrides OpenSLO defaults)
	if valuesFile != "" {
		bytes, err := os.ReadFile(valuesFile)
		if err != nil {
			return fmt.Errorf("reading values file: %w", err)
		}
		if err := yaml.Unmarshal(bytes, &configData); err != nil {
			return fmt.Errorf("parsing values file: %w", err)
		}
	}

	// 3. Read from --set flags (overrides file and OpenSLO defaults)
	for _, setVal := range setVals {
		parts := strings.SplitN(setVal, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --set flag %q: expected key=value format", setVal)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return fmt.Errorf("invalid --set flag %q: key cannot be empty", setVal)
		}
		val := strings.TrimSpace(parts[1])
		parsedVal := parseSetVal(val)
		setPath(configData, strings.Split(key, "."), parsedVal)
	}

	// 4. Read from --set-json flags
	for _, setJSONVal := range setJSONVals {
		parts := strings.SplitN(setJSONVal, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --set-json flag %q: expected key=value format", setJSONVal)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return fmt.Errorf("invalid --set-json flag %q: key cannot be empty", setJSONVal)
		}
		rawVal := strings.TrimSpace(parts[1])
		var parsedVal any
		if err := yaml.Unmarshal([]byte(rawVal), &parsedVal); err != nil {
			return fmt.Errorf("invalid json/yaml in --set-json %q: %w", setJSONVal, err)
		}
		setPath(configData, strings.Split(key, "."), cleanYAMLValue(parsedVal))
	}

	// 5. Determine output format (yaml or json)
	formatFlag, _ := cmd.Flags().GetString("format")
	if formatFlag == "" {
		formatFlag = v.GetString("format")
	}
	if formatFlag == "" {
		formatFlag = v.GetString("output")
	}

	if formatFlag == "" {
		formatFlag = templates.GetDefaultFormat(templatePath)
	}

	formatFlag = strings.ToLower(strings.TrimSpace(formatFlag))
	if formatFlag != "yaml" && formatFlag != "json" {
		return fmt.Errorf("invalid format %q: must be yaml or json", formatFlag)
	}

	// Determine the CUE template content:
	// 1. If templatePath exists on disk, load it as a custom file (custom templates take precedence).
	// 2. Otherwise, check if templatePath matches an exact built-in template identifier.
	// 3. If neither, return a descriptive error.
	var templateContent string
	if _, statErr := os.Stat(templatePath); statErr == nil {
		bytes, err := os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("reading template file: %w", err)
		}
		templateContent = string(bytes)
	} else if builtInContent, err := templates.GetTemplate(templatePath); err == nil {
		templateContent = builtInContent
	} else {
		if strings.ContainsAny(templatePath, `/\`) || strings.HasSuffix(templatePath, ".cue") {
			return fmt.Errorf("reading template file: %w", statErr)
		}
		return fmt.Errorf("unknown template %q (not a built-in template and file does not exist)", templatePath)
	}

	if templatePath == "datadog" {
		if tf, ok := configData["timeframe"].(string); ok && tf != "" {
			if tf != "7d" && tf != "30d" && tf != "90d" {
				return fmt.Errorf("datadog SLO timeframe %q is invalid: must be 7d, 30d, or 90d", tf)
			}
		} else {
			winSec := totalWindow.Seconds()
			if winSec != 7*86400 && winSec != 30*86400 && winSec != 90*86400 {
				return fmt.Errorf("datadog SLOs only support 7d, 30d, or 90d timeframes (window is %s); use --set timeframe=7d|30d|90d to override explicitly", windowStr)
			}
		}
	}

	params := domain.GenerateParams{
		TargetSLO:   sloTarget,
		TotalWindow: totalWindow,
		Format:      formatFlag,
	}

	res, err := renderService.RenderTemplate(params, configData, templateContent)
	if err != nil {
		return fmt.Errorf("generating config: %w", err)
	}

	fmt.Fprint(cmd.OutOrStdout(), res)
	return nil
}

func setPath(target map[string]any, path []string, value any) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		if existingMap, ok := target[path[0]].(map[string]any); ok {
			if newMap, ok := value.(map[string]any); ok {
				for k, v := range newMap {
					existingMap[k] = v
				}
				return
			}
		}
		target[path[0]] = value
		return
	}

	currKey := path[0]
	subMap, ok := target[currKey].(map[string]any)
	if !ok {
		subMap = make(map[string]any)
		target[currKey] = subMap
	}
	setPath(subMap, path[1:], value)
}

func parseSetVal(val string) any {
	trimmed := strings.TrimSpace(val)
	// Explicit map syntax: {k1=v1,k2=v2} or {k1: v1, k2: v2}
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		if inner == "" {
			return make(map[string]any)
		}
		// Check comma-separated k=v pairs: {k1=v1,k2=v2}
		if strings.Contains(inner, "=") {
			pairs := strings.Split(inner, ",")
			m := make(map[string]any)
			validMap := true
			for _, pair := range pairs {
				k, v, ok := strings.Cut(pair, "=")
				if !ok || strings.TrimSpace(k) == "" {
					validMap = false
					break
				}
				m[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
			if validMap && len(m) > 0 {
				return m
			}
		}
		// Otherwise try JSON/YAML map: {k1: v1, k2: v2}
		var parsed any
		if err := yaml.Unmarshal([]byte(trimmed), &parsed); err == nil {
			if m, ok := parsed.(map[string]any); ok {
				return cleanYAMLValue(m)
			}
		}
	}
	return val
}

func cleanYAMLValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[k] = cleanYAMLValue(item)
		}
		return res
	case map[any]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[fmt.Sprintf("%v", k)] = cleanYAMLValue(item)
		}
		return res
	case []any:
		res := make([]any, len(val))
		for i, item := range val {
			res[i] = cleanYAMLValue(item)
		}
		return res
	default:
		return val
	}
}
