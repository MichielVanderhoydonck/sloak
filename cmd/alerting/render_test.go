package alerting_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/cmd/alerting"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/testutil"
	"github.com/spf13/viper"
)

type mockRenderService struct {
	res string
	err error
}

var _ domain.TemplateRenderer = (*mockRenderService)(nil)

func (m *mockRenderService) RenderTemplate(_ domain.GenerateParams, _ map[string]any, _ string) (string, error) {
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
