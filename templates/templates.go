package templates

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed *.cue
var FS embed.FS

func GetTemplate(nameOrPath string) (string, error) {
	builtIns := map[string]string{
		"prometheus-operator":    "prometheus_operator_mwmbr.cue",
		"datadog":                "datadog.cue",
		"grafana-cloud":          "grafana_cloud_alerting.cue",
		"grafana-cloud-alerting": "grafana_cloud_alerting.cue",
		"otel-collector":         "otel_collector.cue",
	}

	normalized := strings.ToLower(filepath.Base(nameOrPath))
	normalized = strings.TrimSuffix(normalized, ".cue")

	if embeddedFile, ok := builtIns[normalized]; ok {
		bytes, err := FS.ReadFile(embeddedFile)
		if err != nil {
			return "", fmt.Errorf("failed to read embedded template %q: %w", embeddedFile, err)
		}
		return string(bytes), nil
	}

	return "", fmt.Errorf("not a built-in template")
}
