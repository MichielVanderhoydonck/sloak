package convert_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/cmd/convert"
	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/convert"
	util "github.com/MichielVanderhoydonck/sloak/internal/util"
	"github.com/spf13/viper"
)

type mockService struct {
	res domain.ConversionResult
	err error
}

func (m *mockService) Convert(_ domain.ConversionParams) (domain.ConversionResult, error) {
	return m.res, m.err
}

func TestConvertCommand(t *testing.T) {
	t.Parallel()
	viper.Reset()
	mockRes := domain.ConversionResult{
		AvailabilityPercent: 99.9,
		DailyDowntime:       util.Duration(1*time.Minute + 26*time.Second),
		YearlyDowntime:      util.Duration(8*time.Hour + 45*time.Minute),
	}

	svc := &mockService{res: mockRes, err: nil}

	var output bytes.Buffer
	cmd := convert.NewConvertCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--nines=99.9"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := output.String()
	if !strings.Contains(outStr, "Availability: 99.90000%") {
		t.Errorf("Missing availability in output: %s", outStr)
	}
	if !strings.Contains(outStr, "Daily Allowed:     1m26s") {
		t.Errorf("Missing daily downtime in output: %s", outStr)
	}
}

func TestConvertCommand_BothFlagsError(t *testing.T) {
	t.Parallel()
	viper.Reset()
	svc := &mockService{}

	var output bytes.Buffer
	cmd := convert.NewConvertCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--nines=99.9", "--downtime=10m"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when both flags are passed, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestConvertCommand_NeitherFlagError(t *testing.T) {
	t.Parallel()
	viper.Reset()
	svc := &mockService{}

	var output bytes.Buffer
	cmd := convert.NewConvertCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when neither flag is passed, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestConvertCommand_InvalidDowntime(t *testing.T) {
	t.Parallel()
	viper.Reset()
	svc := &mockService{}

	var output bytes.Buffer
	cmd := convert.NewConvertCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--downtime=invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid downtime, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}

func TestConvertCommand_ServiceError(t *testing.T) {
	t.Parallel()
	viper.Reset()
	svc := &mockService{err: errors.New("conversion failed")}

	var output bytes.Buffer
	cmd := convert.NewConvertCmd(svc)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--nines=99.9"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from service failure, got nil")
	}

	outStr := output.String()
	if outStr != "" {
		t.Errorf("expected empty stdout on error, got: %q", outStr)
	}
}
