package alerting

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

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

	sloVal := v.GetFloat64("slo")
	windowStr := v.GetString("window")

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

	var valuesBytes []byte
	if valuesFile != "" {
		valuesBytes, err = os.ReadFile(valuesFile)
		if err != nil {
			return fmt.Errorf("reading values file: %w", err)
		}
	}

	configData, err := BuildRenderConfig(openSLOConfig, valuesBytes, setVals, setJSONVals)
	if err != nil {
		return err
	}

	formatFlag, err := resolveOutputFormat(cmd, v.GetString("format"), v.GetString("output"), templatePath)
	if err != nil {
		return err
	}

	templateContent, err := templates.ResolveTemplate(templatePath)
	if err != nil {
		return err
	}

	params := domain.GenerateParams{
		TargetSLO:   sloTarget,
		TotalWindow: totalWindow,
		Format:      formatFlag,
		Template:    templatePath,
	}

	res, err := renderService.RenderTemplate(params, configData, templateContent)
	if err != nil {
		return fmt.Errorf("generating config: %w", err)
	}

	fmt.Fprint(cmd.OutOrStdout(), res)
	return nil
}

func resolveOutputFormat(cmd *cobra.Command, cfgFormat, cfgOutput, templatePath string) (string, error) {
	formatFlag, _ := cmd.Flags().GetString("format")
	if formatFlag == "" {
		formatFlag = cfgFormat
	}
	if formatFlag == "" {
		formatFlag = cfgOutput
	}
	if formatFlag == "" {
		formatFlag = templates.GetDefaultFormat(templatePath)
	}

	formatFlag = strings.ToLower(strings.TrimSpace(formatFlag))
	if formatFlag != "yaml" && formatFlag != "json" {
		return "", fmt.Errorf("invalid format %q: must be yaml or json", formatFlag)
	}
	return formatFlag, nil
}
