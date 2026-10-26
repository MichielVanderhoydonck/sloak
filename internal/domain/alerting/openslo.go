package alerting

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
)

// SanitizePrometheusMetricName sanitizes an arbitrary string (such as an OpenSLO DNS-1123 name)
// so that it is a valid Prometheus metric name matching ^[a-zA-Z_:][a-zA-Z0-9_:]*$.
func SanitizePrometheusMetricName(name string) string {
	if name == "" {
		return "slo_errors"
	}
	var b strings.Builder
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == ':' {
			b.WriteRune(r)
		} else if r >= '0' && r <= '9' {
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	res := b.String()
	if res == "" || res == "_" {
		return "slo_errors"
	}
	return res
}

func ParseOpenSLO(filePath string) (float64, time.Duration, string, map[string]any, error) {
	bytesData, err := os.ReadFile(filePath)
	if err != nil {
		return 0, 0, "", nil, fmt.Errorf("failed to read OpenSLO file: %w", err)
	}

	objects, err := openslosdk.Decode(bytes.NewBuffer(bytesData), openslosdk.FormatYAML)
	if err != nil {
		return 0, 0, "", nil, fmt.Errorf("failed to parse OpenSLO YAML: %w", err)
	}

	if len(objects) == 0 {
		return 0, 0, "", nil, fmt.Errorf("no objects found in OpenSLO file")
	}

	var slo v1.SLO
	found := false
	for _, obj := range objects {
		if s, ok := obj.(v1.SLO); ok {
			slo = s
			found = true
			break
		}
	}

	if !found {
		return 0, 0, "", nil, fmt.Errorf("no SLO object found in OpenSLO file")
	}

	if err := openslosdk.Validate(slo); err != nil {
		return 0, 0, "", nil, fmt.Errorf("OpenSLO validation failed: %w", err)
	}

	if len(slo.Spec.Objectives) == 0 {
		return 0, 0, "", nil, fmt.Errorf("at least one objective is required in spec.objectives")
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
		return 0, 0, "", nil, fmt.Errorf("objective must specify target or targetPercent")
	}

	if len(slo.Spec.TimeWindow) == 0 {
		return 0, 0, "", nil, fmt.Errorf("timeWindow is required in spec.timeWindow")
	}

	tw := slo.Spec.TimeWindow[0]
	parsedWindow := tw.Duration.Duration()

	rawMetric := slo.Metadata.Name
	if slo.Spec.Indicator != nil && slo.Spec.Indicator.Metadata.Name != "" {
		rawMetric = slo.Spec.Indicator.Metadata.Name
	}

	metaConfig := make(map[string]any)
	metaConfig["metric_name"] = SanitizePrometheusMetricName(rawMetric)
	if slo.Metadata.DisplayName != "" {
		metaConfig["name"] = slo.Metadata.DisplayName
	} else {
		metaConfig["name"] = slo.Metadata.Name
	}
	if slo.Spec.Service != "" {
		metaConfig["service"] = slo.Spec.Service
	}

	if len(slo.Metadata.Labels) > 0 {
		metaConfig["labels"] = slo.Metadata.Labels
	}

	return target, parsedWindow, slo.Metadata.Name, metaConfig, nil
}
