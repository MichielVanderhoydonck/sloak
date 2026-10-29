package openslo

import (
	"bytes"
	"fmt"
	"os"

	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/OpenSLO/go-sdk/pkg/openslosdk"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
)

// OpenSLOLoader implements domain.SpecLoader for OpenSLO definitions.
type OpenSLOLoader struct{}

// NewSpecLoader returns a new OpenSLO spec loader.
func NewSpecLoader() domain.SpecLoader {
	return &OpenSLOLoader{}
}

// LoadSpec reads and parses an OpenSLO YAML file into a domain SLOSpec.
func (l *OpenSLOLoader) LoadSpec(filePath string) (domain.SLOSpec, error) {
	return ParseFile(filePath)
}

// ParseFile reads and parses an OpenSLO YAML file into a domain SLOSpec.
func ParseFile(filePath string) (domain.SLOSpec, error) {
	bytesData, err := os.ReadFile(filePath)
	if err != nil {
		return domain.SLOSpec{}, fmt.Errorf("failed to read OpenSLO file: %w", err)
	}

	return Parse(bytesData)
}

// Parse parses OpenSLO YAML content into a domain SLOSpec.
func Parse(data []byte) (domain.SLOSpec, error) {
	objects, err := openslosdk.Decode(bytes.NewBuffer(data), openslosdk.FormatYAML)
	if err != nil {
		return domain.SLOSpec{}, fmt.Errorf("failed to parse OpenSLO YAML: %w", err)
	}

	if len(objects) == 0 {
		return domain.SLOSpec{}, fmt.Errorf("no objects found in OpenSLO file")
	}

	var slo v1.SLO
	found := false
	sliByName := make(map[string]v1.SLI)
	for _, obj := range objects {
		switch o := obj.(type) {
		case v1.SLO:
			if !found {
				slo = o
				found = true
			}
		case v1.SLI:
			if o.Metadata.Name != "" {
				sliByName[o.Metadata.Name] = o
			}
		}
	}

	if !found {
		return domain.SLOSpec{}, fmt.Errorf("no SLO object found in OpenSLO file")
	}

	if err := openslosdk.Validate(slo); err != nil {
		return domain.SLOSpec{}, fmt.Errorf("OpenSLO validation failed: %w", err)
	}

	if len(slo.Spec.Objectives) == 0 {
		return domain.SLOSpec{}, fmt.Errorf("at least one objective is required in spec.objectives")
	}

	obj := slo.Spec.Objectives[0]
	var target float64
	if obj.TargetPercent != nil {
		target = *obj.TargetPercent
	} else if obj.Target != nil {
		val := *obj.Target
		if val > 1.0 {
			target = val
		} else {
			target = val * 100.0
		}
	} else {
		return domain.SLOSpec{}, fmt.Errorf("objective must specify target or targetPercent")
	}

	if len(slo.Spec.TimeWindow) == 0 {
		return domain.SLOSpec{}, fmt.Errorf("timeWindow is required in spec.timeWindow")
	}

	tw := slo.Spec.TimeWindow[0]
	parsedWindow := tw.Duration.Duration()

	rawMetric := slo.Metadata.Name
	var sliSpec *v1.SLISpec
	if slo.Spec.Indicator != nil {
		if slo.Spec.Indicator.Metadata.Name != "" {
			rawMetric = slo.Spec.Indicator.Metadata.Name
		}
		sliSpec = &slo.Spec.Indicator.Spec
	} else if slo.Spec.IndicatorRef != nil {
		if referencedSLI, ok := sliByName[*slo.Spec.IndicatorRef]; ok {
			if err := openslosdk.Validate(referencedSLI); err != nil {
				return domain.SLOSpec{}, fmt.Errorf("OpenSLO SLI validation failed: %w", err)
			}
			if referencedSLI.Metadata.Name != "" {
				rawMetric = referencedSLI.Metadata.Name
			}
			sliSpec = &referencedSLI.Spec
		}
	}

	metaConfig := make(map[string]any)
	metaConfig["metric_name"] = domain.SanitizePrometheusMetricName(rawMetric)
	if slo.Metadata.DisplayName != "" {
		metaConfig["name"] = slo.Metadata.DisplayName
	} else {
		metaConfig["name"] = slo.Metadata.Name
	}
	if slo.Spec.Description != "" {
		metaConfig["description"] = slo.Spec.Description
	}
	if slo.Spec.Service != "" {
		metaConfig["service"] = slo.Spec.Service
	}

	if len(slo.Metadata.Labels) > 0 {
		metaConfig["labels"] = slo.Metadata.Labels
	}

	if sliSpec != nil && sliSpec.RatioMetric != nil {
		extractRatioMetricQueries(sliSpec.RatioMetric, metaConfig)
	}

	sloTarget, err := common.NewSLOTarget(target)
	if err != nil {
		return domain.SLOSpec{}, err
	}

	return domain.SLOSpec{
		TargetSLO:   sloTarget,
		TotalWindow: parsedWindow,
		Config:      metaConfig,
	}, nil
}

func extractRatioMetricQueries(rm *v1.SLIRatioMetric, metaConfig map[string]any) {
	if badQ := extractMetricQuery(rm.Bad); badQ != "" {
		metaConfig["errors_query"] = badQ
	}
	if goodQ := extractMetricQuery(rm.Good); goodQ != "" {
		metaConfig["good_query"] = goodQ
	}
	if totQ := extractMetricQuery(rm.Total); totQ != "" {
		metaConfig["total_query"] = totQ
	}
	if rawQ := extractMetricQuery(rm.Raw); rawQ != "" {
		if rm.RawType == v1.SLIRawMetricTypeFailure || rm.RawType == "" {
			metaConfig["error_ratio_query"] = rawQ
		}
	}
}

func extractMetricQuery(m *v1.SLIMetricSpec) string {
	if m == nil || m.MetricSource.Spec == nil {
		return ""
	}
	if q, ok := m.MetricSource.Spec["query"].(string); ok {
		return q
	}
	return ""
}
