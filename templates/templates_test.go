package templates_test

import (
	"strings"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/templates"
)

func TestGetTemplate_BuiltIns(t *testing.T) {
	builtIns := []string{
		"prometheus-operator",
		"datadog",
		"grafana-cloud",
		"grafana-cloud-alerting",
		"otel-collector",
		"incident-io",
		"incident_io",
	}

	for _, name := range builtIns {
		t.Run(name, func(t *testing.T) {
			content, err := templates.GetTemplate(name)
			if err != nil {
				t.Fatalf("expected template %q to load successfully, got: %v", name, err)
			}
			if len(content) == 0 {
				t.Errorf("expected non-empty content for template %q", name)
			}
		})
	}
}

func TestGetTemplate_RejectPathsAndExtensions(t *testing.T) {
	invalidNames := []string{
		"./datadog.cue",
		"datadog.cue",
		"/tmp/datadog.cue",
		"../../templates/datadog.cue",
		"prometheus-operator.cue",
		"unknown-template",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			_, err := templates.GetTemplate(name)
			if err == nil {
				t.Errorf("expected GetTemplate(%q) to fail, but got nil", name)
			}
		})
	}
}

func TestGetSchema(t *testing.T) {
	schema, err := templates.GetSchema()
	if err != nil {
		t.Fatalf("expected GetSchema to succeed, got: %v", err)
	}
	if !strings.Contains(schema, "#Alert:") || !strings.Contains(schema, "#Input:") {
		t.Errorf("expected schema to contain #Alert and #Input, got: %s", schema)
	}
}
