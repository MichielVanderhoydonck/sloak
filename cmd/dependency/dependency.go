package dependency

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/dependency"
)

func NewDependencyCmd(service domain.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependency",
		Short: "Calculates composite availability for serial or parallel dependencies.",
		Long: `Calculates composite availability for serial or parallel dependencies.

Note: In parallel mode, this calculation assumes independent failures between components
(i.e. no correlated outages, shared network links, or common single points of failure).`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDependencyCmd(cmd, service)
		},
	}

	cmd.Flags().String("components", "", "Comma-separated list of availabilities (e.g. 99.9,99.5)")
	cmd.Flags().String("type", "serial", "Calculation type: 'serial' (all required) or 'parallel' (redundant, assumes independent failures)")

	return cmd
}

func runDependencyCmd(cmd *cobra.Command, service domain.Service) error {
	viper.BindPFlags(cmd.Flags())
	compStr := viper.GetString("components")
	typeStr := viper.GetString("type")

	if strings.TrimSpace(compStr) == "" {
		return fmt.Errorf("--components is required")
	}

	strValues := strings.Split(compStr, ",")
	var components []float64
	for _, s := range strValues {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("parsing component '%s': %w", s, err)
		}
		components = append(components, val)
	}

	params := domain.CalculationParams{
		Components: components,
		Type:       domain.CalculationType(strings.ToLower(typeStr)),
	}

	res, err := service.CalculateCompositeAvailability(params)
	if err != nil {
		return fmt.Errorf("calculation error: %w", err)
	}

	outputFlag := viper.GetString("output")
	if outputFlag == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(res); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		return nil
	}

	fmt.Printf("\n--- Dependency Availability (%s) ---\n", capitalizeFirst(string(res.CalculationType)))
	fmt.Printf("Components: %d\n", res.ComponentCount)
	fmt.Printf("------------------------------------\n")
	fmt.Printf("Total Availability: %.4f%%\n", res.TotalAvailability)
	return nil
}

func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
