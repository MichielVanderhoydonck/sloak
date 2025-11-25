package dependency

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/dependency"
	port "github.com/MichielVanderhoydonck/sloak/internal/core/port/dependency"
)

var service port.AvailabilityService

func SetService(s port.AvailabilityService) {
	service = s
}

func NewDependencyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependency",
		Short: "Calculates composite availability for serial or parallel dependencies.",
		Run:   runDependencyCmd,
	}

	cmd.Flags().String("components", "", "Comma-separated list of availabilities (e.g. 99.9,99.5)")
	cmd.Flags().String("type", "serial", "Calculation type: 'serial' or 'parallel'")

	return cmd
}

func runDependencyCmd(cmd *cobra.Command, args []string) {
	compStr, _ := cmd.Flags().GetString("components")
	typeStr, _ := cmd.Flags().GetString("type")

	strValues := strings.Split(compStr, ",")
	var components []float64
	for _, s := range strValues {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			fmt.Printf("Error parsing component '%s': %v\n", s, err)
			return
		}
		components = append(components, val)
	}

	params := domain.CalculationParams{
		Components: components,
		Type:       domain.CalculationType(strings.ToLower(typeStr)),
	}

	res, err := service.CalculateCompositeAvailability(params)
	if err != nil {
		fmt.Printf("Calculation Error: %v\n", err)
		return
	}

	fmt.Printf("\n--- Dependency Availability (%s) ---\n", strings.Title(string(res.CalculationType)))
	fmt.Printf("Components: %d\n", res.ComponentCount)
	fmt.Printf("------------------------------------\n")
	fmt.Printf("Total Availability: %.6f%%\n", res.TotalAvailability)
}
