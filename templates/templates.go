package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed *.cue
var FS embed.FS

type BuiltInInfo struct {
	DisplayName     string
	FileName        string
	DefaultFormat   string
	AllowedWindows  []time.Duration
	TimeframeConfig string
	NonPromQL       bool
}

var builtIns = map[string]BuiltInInfo{
	"prometheus-operator":    {DisplayName: "prometheus-operator", FileName: "prometheus_operator_mwmbr.cue", DefaultFormat: "yaml"},
	"datadog":                {DisplayName: "datadog SLO", FileName: "datadog.cue", DefaultFormat: "json", AllowedWindows: []time.Duration{7 * 24 * time.Hour, 30 * 24 * time.Hour, 90 * 24 * time.Hour}, TimeframeConfig: "timeframe", NonPromQL: true},
	"grafana-cloud":          {DisplayName: "grafana-cloud", FileName: "grafana_cloud_alerting.cue", DefaultFormat: "yaml"},
	"grafana-cloud-alerting": {DisplayName: "grafana-cloud", FileName: "grafana_cloud_alerting.cue", DefaultFormat: "yaml"},
	"otel-collector":         {DisplayName: "otel-collector", FileName: "otel_collector.cue", DefaultFormat: "yaml", NonPromQL: true},
	"incident-io":            {DisplayName: "incident-io", FileName: "incident_io.cue", DefaultFormat: "json", NonPromQL: true},
	"incident_io":            {DisplayName: "incident-io", FileName: "incident_io.cue", DefaultFormat: "json", NonPromQL: true},
}

// IsBuiltIn reports whether name is a recognized built-in template identifier.
func IsBuiltIn(name string) bool {
	_, ok := GetBuiltInInfo(name)
	return ok
}

// GetBuiltInInfo returns information about a built-in template, matching either
// the direct template key or its base file name (with or without .cue extension).
func GetBuiltInInfo(name string) (BuiltInInfo, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if info, ok := builtIns[key]; ok {
		return info, true
	}
	base := strings.TrimSuffix(filepath.Base(key), ".cue")
	if info, ok := builtIns[base]; ok {
		return info, true
	}
	return BuiltInInfo{}, false
}

// GetDefaultFormat returns the default output format for a template identifier.
// Built-in templates default to their designated format (e.g. json for Datadog and Incident.io);
// custom templates default to yaml.
func GetDefaultFormat(name string) string {
	if info, ok := GetBuiltInInfo(name); ok {
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

// ResolveTemplate loads a CUE template by checking:
// 1. If templatePath exists on disk, load it as a custom file (custom templates take precedence).
// 2. Otherwise, check if templatePath matches an exact built-in template identifier.
// 3. If neither, return a descriptive error.
func ResolveTemplate(templatePath string) (string, error) {
	if _, statErr := os.Stat(templatePath); statErr == nil {
		bytes, err := os.ReadFile(templatePath)
		if err != nil {
			return "", fmt.Errorf("reading template file: %w", err)
		}
		return string(bytes), nil
	} else if builtInContent, err := GetTemplate(templatePath); err == nil {
		return builtInContent, nil
	} else {
		if strings.ContainsAny(templatePath, `/\`) || strings.HasSuffix(templatePath, ".cue") {
			return "", fmt.Errorf("reading template file: %w", statErr)
		}
		return "", fmt.Errorf("unknown template %q (not a built-in template and file does not exist)", templatePath)
	}
}

// GetSchema returns the shared CUE input and alert schema definition.
func GetSchema() (string, error) {
	bytes, err := FS.ReadFile("schema.cue")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded schema.cue: %w", err)
	}
	return string(bytes), nil
}
