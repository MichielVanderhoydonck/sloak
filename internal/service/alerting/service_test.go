package alerting_test

import (
	"math"
	"testing"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
	service "github.com/MichielVanderhoydonck/sloak/internal/service/alerting"
)

func TestGenerateMatrix(t *testing.T) {
	svc := service.NewAlertGeneratorService()
	slo, _ := common.NewSLOTarget(99.9)

	params := domain.GenerateParams{
		TargetSLO:   slo,
		TotalWindow: 30 * 24 * time.Hour,
	}

	res, _ := svc.GenerateTable(params)

	if len(res.Alerts) != 3 {
		t.Fatalf("Expected 3 alert rules, got %d", len(res.Alerts))
	}

	assertFloat := func(name string, expected, actual float64) {
		if math.Abs(expected-actual) > 0.01 {
			t.Errorf("[%s] Expected %.2f, got %.2f", name, expected, actual)
		}
	}

	r1 := res.Alerts[0]
	assertFloat("R1 BurnRate", 14.4, r1.BurnRate)
	if r1.NotificationType != domain.Page {
		t.Errorf("R1 expected Page, got %s", r1.NotificationType)
	}
	r2 := res.Alerts[1]
	assertFloat("R2 BurnRate", 6.0, r2.BurnRate)

	if r2.NotificationType != domain.Message {
		t.Errorf("R2 expected Page, got %s", r2.NotificationType)
	}

	r3 := res.Alerts[2]
	assertFloat("R3 BurnRate", 1.0, r3.BurnRate)
	if r3.NotificationType != domain.Ticket {
		t.Errorf("R3 expected Ticket, got %s", r3.NotificationType)
	}
}
