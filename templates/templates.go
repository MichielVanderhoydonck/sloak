package templates

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.cue
var FS embed.FS

var builtIns = map[string]string{
	"prometheus-operator":    "prometheus_operator_mwmbr.cue",
	"datadog":                "datadog.cue",
	"grafana-cloud":          "grafana_cloud_alerting.cue",
	"grafana-cloud-alerting": "grafana_cloud_alerting.cue",
	"otel-collector":         "otel_collector.cue",
	"incident-io":            "incident_io.cue",
	"incident_io":            "incident_io.cue",
}

// IsBuiltIn reports whether name is a recognized built-in template identifier.
func IsBuiltIn(name string) bool {
	_, ok := builtIns[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

// GetTemplate returns the content of an embedded built-in template.
// It requires an exact match on the built-in identifier, preventing collisions
// with filesystem paths or custom template files.
func GetTemplate(name string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if embeddedFile, ok := builtIns[key]; ok {
		bytes, err := FS.ReadFile(embeddedFile)
		if err != nil {
			return "", fmt.Errorf("failed to read embedded template %q: %w", embeddedFile, err)
		}
		return string(bytes), nil
	}

	return "", fmt.Errorf("not a built-in template: %q", name)
}

// GetSchema returns the shared CUE input and alert schema definition.
func GetSchema() (string, error) {
	bytes, err := FS.ReadFile("schema.cue")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded schema.cue: %w", err)
	}
	return string(bytes), nil
}
