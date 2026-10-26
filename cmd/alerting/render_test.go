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
}

var _ domain.TemplateRenderer = (*mockRenderService)(nil)

func (m *mockRenderService) RenderTemplate(_ domain.GenerateParams, _ map[string]any, tpl string) (string, error) {
	m.receivedTemplate = tpl
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
