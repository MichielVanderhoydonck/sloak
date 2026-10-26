package alerting_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/cmd/alerting"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
	"github.com/spf13/viper"
)

type mockRenderService struct {
	res              string
	err              error
	receivedTemplate string
	receivedConfig   map[string]any
}

var _ domain.TemplateRenderer = (*mockRenderService)(nil)

func (m *mockRenderService) RenderTemplate(_ domain.GenerateParams, cfg map[string]any, tpl string) (string, error) {
	m.receivedTemplate = tpl
	m.receivedConfig = cfg
	return m.res, m.err
}

func TestRenderCommand_Success(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "apiVersion: monitoring.coreos.com/v1\nkind: PrometheusRule\n"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=prometheus-operator",
		"--slo=99.9",
	})

	err := cmd.Execute()
	restore()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "apiVersion: monitoring.coreos.com/v1") {
		t.Errorf("output missing expected yaml, got: %q", outStr)
	}
}

func TestRenderCommand_MissingTemplate(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "some yaml"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--slo=99.9",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for missing --template, got nil")
	}

	if !strings.Contains(err.Error(), "--template is required") {
		t.Errorf("unexpected error message: %v", err)
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on failure, got: %q", outStr)
	}
}

func TestRenderCommand_NonExistentTemplateFile(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "some yaml"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=/nonexistent/path/template.cue",
		"--slo=99.9",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for nonexistent template file, got nil")
	}

	if !strings.Contains(err.Error(), "reading template file") {
		t.Errorf("unexpected error message: %v", err)
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on failure, got: %q", outStr)
	}
}

func TestRenderCommand_RendererError(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{err: errors.New("template execution failed")}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=prometheus-operator",
		"--slo=99.9",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error from renderer failure, got nil")
	}

	if !strings.Contains(err.Error(), "generating config") {
		t.Errorf("unexpected error message: %v", err)
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on failure, got: %q", outStr)
	}
}

func TestRenderCommand_InvalidWindow(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "some yaml"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=prometheus-operator",
		"--window=invalid_window",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for invalid window, got nil")
	}

	if !strings.Contains(err.Error(), "parsing window") {
		t.Errorf("unexpected error message: %v", err)
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on failure, got: %q", outStr)
	}
}

func TestRenderCommand_InvalidSLO(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "some yaml"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=prometheus-operator",
		"--slo=150.0",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for invalid slo, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on failure, got: %q", outStr)
	}
}

func TestRenderCommand_CustomTemplateTakesPrecedenceOverBuiltIn(t *testing.T) {
	viper.Reset()
	customContent := `// custom datadog override content`
	tmpDir := t.TempDir()
	customFile := filepath.Join(tmpDir, "datadog.cue")
	if err := os.WriteFile(customFile, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom template file: %v", err)
	}

	svc := &mockRenderService{res: "rendered yaml"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=" + customFile,
		"--slo=99.9",
	})

	err := cmd.Execute()
	restore()
	if err != nil {
		t.Fatalf("expected command to succeed, got: %v", err)
	}

	if svc.receivedTemplate != customContent {
		t.Errorf("expected service to receive custom template content, got: %q", svc.receivedTemplate)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "rendered yaml") {
		t.Errorf("expected rendered output, got: %q", outStr)
	}
}

func TestRenderCommand_NonExistentPathDoesNotFallbackToBuiltIn(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "some yaml"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=./nonexistent/datadog.cue",
		"--slo=99.9",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for nonexistent path, but command succeeded")
	}

	if !strings.Contains(err.Error(), "reading template file") {
		t.Errorf("expected error to contain 'reading template file', got: %v", err)
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on failure, got: %q", outStr)
	}
}

func TestRenderCommand_UnknownBuiltIn(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "some yaml"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=unknown-template-name",
		"--slo=99.9",
	})

	err := cmd.Execute()
	restore()
	if err == nil {
		t.Fatal("expected error for unknown template name, but command succeeded")
	}

	if !strings.Contains(err.Error(), "unknown template") {
		t.Errorf("expected error to contain 'unknown template', got: %v", err)
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on failure, got: %q", outStr)
	}
}

func TestRenderCommand_SetDottedAndMapFlags(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "some yaml"}

	output, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=prometheus-operator",
		"--slo=99.9",
		"--set", "metric_name=custom_api",
		"--set", "rule_labels={service=api,env=prod}",
		"--set", "meta_labels.release=prometheus",
		"--set", "meta_labels.role=alert-rules",
	})

	err := cmd.Execute()
	restore()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if svc.receivedConfig["metric_name"] != "custom_api" {
		t.Errorf("expected metric_name to be 'custom_api', got: %v", svc.receivedConfig["metric_name"])
	}

	ruleLabels, ok := svc.receivedConfig["rule_labels"].(map[string]any)
	if !ok {
		t.Fatalf("expected rule_labels to be map[string]any, got: %T (%v)", svc.receivedConfig["rule_labels"], svc.receivedConfig["rule_labels"])
	}
	if ruleLabels["service"] != "api" || ruleLabels["env"] != "prod" {
		t.Errorf("unexpected rule_labels content: %v", ruleLabels)
	}

	metaLabels, ok := svc.receivedConfig["meta_labels"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta_labels to be map[string]any, got: %T (%v)", svc.receivedConfig["meta_labels"], svc.receivedConfig["meta_labels"])
	}
	if metaLabels["release"] != "prometheus" || metaLabels["role"] != "alert-rules" {
		t.Errorf("unexpected meta_labels content: %v", metaLabels)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "some yaml") {
		t.Errorf("expected rendered output, got: %q", outStr)
	}
}

func TestRenderCommand_SetJSONFlags(t *testing.T) {
	viper.Reset()
	svc := &mockRenderService{res: "some yaml"}

	_, restore := testutil.CaptureOutput(t)

	cmd := alerting.NewRenderCmd(svc)
	cmd.SetArgs([]string{
		"--template=prometheus-operator",
		"--slo=99.9",
		"--set-json", `rule_labels={"service":"auth","tier":1}`,
		"--set-json", `enabled=true`,
	})

	err := cmd.Execute()
	restore()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	ruleLabels, ok := svc.receivedConfig["rule_labels"].(map[string]any)
	if !ok {
		t.Fatalf("expected rule_labels to be map[string]any, got: %T (%v)", svc.receivedConfig["rule_labels"], svc.receivedConfig["rule_labels"])
	}
	if ruleLabels["service"] != "auth" || ruleLabels["tier"] != 1 {
		t.Errorf("unexpected rule_labels content: %v", ruleLabels)
	}

	if svc.receivedConfig["enabled"] != true {
		t.Errorf("expected enabled to be true, got: %v", svc.receivedConfig["enabled"])
	}
}
