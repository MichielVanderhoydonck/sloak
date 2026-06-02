package alerting

import (
	"maps"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	alertingService "github.com/MichielVanderhoydonck/sloak/internal/service/alerting"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
	templates "github.com/MichielVanderhoydonck/sloak/templates"
)

func NewRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Renders observability configuration from BYOT templates.",
		Long:  `Renders actionable configurations by taking a provided BYOT CUE template and injecting generic SLO math.`,
		Run:   runRenderCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")
	
	cmd.Flags().String("template", "", "Path to a BYOT CUE template file or name of built-in template (required)")
	cmd.Flags().StringSlice("set", []string{}, "Set configuration values for the template (e.g., --set namespace=monitoring)")
	cmd.Flags().String("values", "", "Path to a YAML/JSON file containing configuration values for the template")
	cmd.Flags().String("spec", "", "Path to an OpenSLO specification file to read SLO target and window from")

	return cmd
}

func runRenderCmd(cmd *cobra.Command, args []string) {
	viper.BindPFlags(cmd.Flags())
	
	templatePath := viper.GetString("template")
	setVals := viper.GetStringSlice("set")
	valuesFile := viper.GetString("values")
	specFile := viper.GetString("spec")

	if templatePath == "" {
		fmt.Println("Error: --template is required")
		return
	}

	var sloFlag float64
	var totalWindow time.Duration
	var openSLOConfig map[string]any

	if specFile != "" {
		target, window, _, metaConfig, err := alertingService.ParseOpenSLO(specFile)
		if err != nil {
			fmt.Printf("Error parsing OpenSLO spec: %v\n", err)
			return
		}
		sloFlag = target
		totalWindow = window
		openSLOConfig = metaConfig
	} else {
		sloFlag = viper.GetFloat64("slo")
		windowStr := viper.GetString("window")

		var err error
		totalWindow, err = util.ParseTimeWindow(windowStr)
		if err != nil {
			fmt.Printf("Error parsing window: %v\n", err)
			return
		}
	}

	sloTarget, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Build the generic configuration map
	configData := make(map[string]any)

	// 1. Read from OpenSLO metadata if parsed
	maps.Copy(configData, openSLOConfig)

	// 2. Read from values file (overrides OpenSLO defaults)
	if valuesFile != "" {
		bytes, err := os.ReadFile(valuesFile)
		if err != nil {
			fmt.Printf("Error reading values file: %v\n", err)
			return
		}
		if err := yaml.Unmarshal(bytes, &configData); err != nil {
			fmt.Printf("Error parsing values file: %v\n", err)
			return
		}
	}

	// 3. Read from --set flags (overrides file and OpenSLO defaults)
	for _, setVal := range setVals {
		parts := strings.SplitN(setVal, "=", 2)
		if len(parts) == 2 {
			configData[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Determine the CUE template content (check built-in templates first)
	var templateContent string
	builtInContent, err := templates.GetTemplate(templatePath)
	if err == nil {
		templateContent = builtInContent
	} else {
		bytes, err := os.ReadFile(templatePath)
		if err != nil {
			fmt.Printf("Error reading template file: %v\n", err)
			return
		}
		templateContent = string(bytes)
	}

	params := domain.GenerateParams{
		TargetSLO:   sloTarget,
		TotalWindow: totalWindow,
	}

	res, err := service.RenderTemplate(params, configData, templateContent)
	if err != nil {
		fmt.Printf("Error generating config: %v\n", err)
		return
	}

	fmt.Print(res)
}

