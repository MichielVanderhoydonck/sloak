package templates

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.cue
var FS embed.FS

type BuiltInInfo struct {
	FileName      string
	DefaultFormat string
}

var builtIns = map[string]BuiltInInfo{
	"prometheus-operator":    {"prometheus_operator_mwmbr.cue", "yaml"},
	"datadog":                {"datadog.cue", "json"},
	"grafana-cloud":          {"grafana_cloud_alerting.cue", "yaml"},
	"grafana-cloud-alerting": {"grafana_cloud_alerting.cue", "yaml"},
	"otel-collector":         {"otel_collector.cue", "yaml"},
	"incident-io":            {"incident_io.cue", "json"},
	"incident_io":            {"incident_io.cue", "json"},
}

// IsBuiltIn reports whether name is a recognized built-in template identifier.
func IsBuiltIn(name string) bool {
	_, ok := builtIns[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

// GetDefaultFormat returns the default output format for a template identifier.
// Built-in templates default to their designated format (e.g. json for Datadog and Incident.io);
// custom templates default to yaml.
func GetDefaultFormat(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	if info, ok := builtIns[key]; ok {
		return info.DefaultFormat
	}
	return "yaml"
}

// GetTemplate returns the content of an embedded built-in template.
// It requires an exact match on the built-in identifier, preventing collisions
// with filesystem paths or custom template files.
func GetTemplate(name string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if info, ok := builtIns[key]; ok {
		bytes, err := FS.ReadFile(info.FileName)
		if err != nil {
			return "", fmt.Errorf("failed to read embedded template %q: %w", info.FileName, err)
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
