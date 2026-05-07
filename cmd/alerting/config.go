package alerting

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	common "github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Generates observability configuration from templates.",
		Long:  `Generates actionable configurations by rendering a provided BYOT CUE template with generic SLO math.`,
		Run:   runConfigCmd,
	}

	cmd.Flags().Float64P("slo", "s", 99.9, "SLO target percentage")
	cmd.Flags().StringP("window", "w", "30d", "Total time window")
	
	cmd.Flags().String("template", "", "Path to a BYOT CUE template file (required)")
	cmd.Flags().StringSlice("set", []string{}, "Set configuration values for the template (e.g., --set namespace=monitoring)")
	cmd.Flags().String("values", "", "Path to a YAML/JSON file containing configuration values for the template")

	return cmd
}

func runConfigCmd(cmd *cobra.Command, args []string) {
	viper.BindPFlags(cmd.Flags())
	sloFlag := viper.GetFloat64("slo")
	windowStr := viper.GetString("window")
	
	templatePath := viper.GetString("template")
	setVals := viper.GetStringSlice("set")
	valuesFile := viper.GetString("values")

	if templatePath == "" {
		fmt.Println("Error: --template is required")
		return
	}

	totalWindow, err := util.ParseTimeWindow(windowStr)
	if err != nil {
		fmt.Printf("Error parsing window: %v\n", err)
		return
	}

	sloTarget, err := common.NewSLOTarget(sloFlag)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Build the generic configuration map
	configData := make(map[string]interface{})

	// 1. Read from values file
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

	// 2. Read from --set flags (overrides file)
	for _, setVal := range setVals {
		parts := strings.SplitN(setVal, "=", 2)
		if len(parts) == 2 {
			configData[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Determine the CUE template content
	bytes, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Printf("Error reading template file: %v\n", err)
		return
	}
	templateContent := string(bytes)

	params := domain.GenerateParams{
		TargetSLO:   sloTarget,
		TotalWindow: totalWindow,
	}

	fmt.Printf("DEBUG: valuesFile is %q\n", valuesFile)
	fmt.Printf("DEBUG: configData is %#v\n", configData)

	res, err := service.RenderTemplate(params, configData, templateContent)
	if err != nil {
		fmt.Printf("Error generating config: %v\n", err)
		return
	}

	fmt.Print(res)
}
