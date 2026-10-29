package dependency

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichielVanderhoydonck/sloak/cmd/cliutil"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/dependency"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
)

func NewDependencyCmd(service domain.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependency",
		Short: "Calculates composite availability for serial or parallel dependencies.",
		Long: `Calculates composite availability for serial or parallel dependencies.
Supports named components (e.g. api=99.99,db=99.9) and inline parallel redundancy groups using '|' (e.g. api=99.99,db1=99.9|db2=99.9).

Note: In parallel mode, this calculation assumes independent failures between components
(i.e. no correlated outages, shared network links, or common single points of failure).`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDependencyCmd(cmd, service)
		},
	}

	cmd.Flags().String("components", "", "Comma-separated list of availabilities, optionally named or grouped with '|' (e.g. 99.9,99.5 or api=99.99,db1=99.9|db2=99.9)")
	cmd.Flags().String("type", "serial", "Calculation type: 'serial' (all required) or 'parallel' (redundant, assumes independent failures)")
	cmd.Flags().Float64P("slo", "s", 0, "Optional target SLO percentage to verify whether dependencies can support it (e.g. 99.9)")
	cmd.Flags().StringP("window", "w", "30d", "Time window for composite expected downtime calculation (e.g. 30d)")

	return cmd
}

func runDependencyCmd(cmd *cobra.Command, service domain.Service) error {
	v, err := cliutil.ScopedConfig(cmd)
	if err != nil {
		return err
	}
	compStr := v.GetString("components")
	typeStr := v.GetString("type")

	if strings.TrimSpace(compStr) == "" {
		return fmt.Errorf("--components is required")
	}

	stages, components, err := parseComponentsFlag(compStr)
	if err != nil {
		return err
	}

	var timeWindow util.Duration
	if windowStr := v.GetString("window"); strings.TrimSpace(windowStr) != "" {
		parsedWin, err := util.ParseTimeWindow(windowStr)
		if err != nil {
			return fmt.Errorf("parsing window: %w", err)
		}
		timeWindow = util.Duration(parsedWin)
	}

	var targetSLO *common.SLOTarget
	if cmd.Flags().Changed("slo") {
		sloVal := v.GetFloat64("slo")
		vo, err := common.NewSLOTarget(sloVal)
		if err != nil {
			return err
		}
		targetSLO = &vo
	}

	params := domain.CalculationParams{
		Components: components,
		Stages:     stages,
		Type:       domain.CalculationType(strings.ToLower(typeStr)),
		TargetSLO:  targetSLO,
		TimeWindow: timeWindow,
	}

	res, err := service.CalculateCompositeAvailability(params)
	if err != nil {
		return fmt.Errorf("calculation error: %w", err)
	}

	return cliutil.PrintResult(cmd, v.GetString("output"), res, func(w io.Writer) error {
		fmt.Fprintf(w, "\n--- Dependency Availability (%s) ---\n", capitalizeFirst(string(res.CalculationType)))
		fmt.Fprintf(w, "Components: %d\n", res.ComponentCount)
		fmt.Fprintf(w, "------------------------------------\n")
		fmt.Fprintf(w, "Total Availability: %.4f%%\n", res.TotalAvailability)
		if res.AllowedDowntime != nil && res.TimeWindow != nil {
			fmt.Fprintf(w, "Expected Downtime (%s): %s\n", *res.TimeWindow, *res.AllowedDowntime)
		}
		if res.Bottleneck != nil && res.Bottleneck.Name != "" {
			fmt.Fprintf(w, "Bottleneck: %s (%.4f%%)\n", res.Bottleneck.Name, res.Bottleneck.Availability)
		}
		if res.TargetSLO != nil && res.MeetsTarget != nil {
			if *res.MeetsTarget {
				fmt.Fprintf(w, "\nStatus: MEETS_TARGET (Target: %.3f%%)\n", res.TargetSLO.Value)
			} else {
				fmt.Fprintf(w, "\nStatus: BELOW_TARGET (Target: %.3f%%)\n", res.TargetSLO.Value)
			}
		}
		return nil
	})
}

func parseComponentsFlag(compStr string) ([]domain.Stage, []float64, error) {
	strValues := strings.Split(compStr, ",")
	var stages []domain.Stage
	var flatComponents []float64
	hasNamedOrGrouped := false

	for _, rawStage := range strValues {
		rawStage = strings.TrimSpace(rawStage)
		if rawStage == "" {
			continue
		}
		if strings.Contains(rawStage, "|") {
			hasNamedOrGrouped = true
			parts := strings.Split(rawStage, "|")
			var members []domain.Component
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				comp, err := parseSingleComponent(part)
				if err != nil {
					return nil, nil, err
				}
				members = append(members, comp)
				flatComponents = append(flatComponents, comp.Availability)
			}
			if len(members) > 0 {
				stages = append(stages, domain.Stage{RedundantMembers: members})
			}
			continue
		}

		comp, err := parseSingleComponent(rawStage)
		if err != nil {
			return nil, nil, err
		}
		if comp.Name != "" {
			hasNamedOrGrouped = true
		}
		stages = append(stages, domain.Stage{
			Name:         comp.Name,
			Availability: comp.Availability,
		})
		flatComponents = append(flatComponents, comp.Availability)
	}

	if !hasNamedOrGrouped {
		return nil, flatComponents, nil
	}
	return stages, flatComponents, nil
}

func parseSingleComponent(token string) (domain.Component, error) {
	if idx := strings.Index(token, "="); idx != -1 {
		name := strings.TrimSpace(token[:idx])
		valStr := strings.TrimSpace(token[idx+1:])
		if name == "" {
			return domain.Component{}, fmt.Errorf("parsing component '%s': empty component name", token)
		}
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return domain.Component{}, fmt.Errorf("parsing component '%s': %w", token, err)
		}
		return domain.Component{Name: name, Availability: val}, nil
	}

	val, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return domain.Component{}, fmt.Errorf("parsing component '%s': %w", token, err)
	}
	return domain.Component{Availability: val}, nil
}

func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
