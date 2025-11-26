package alerting_test

import (
	"math"
	"testing"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
	service "github.com/MichielVanderhoydonck/sloak/internal/core/service/alerting"
)

func TestAlertGeneratorService(t *testing.T) {
	svc := service.NewAlertGeneratorService()

	mustSLO := func(v float64) common.SLOTarget {
		s, _ := common.NewSLOTarget(v)
		return s
	}

	testCases := []struct {
		name             string
		params           domain.GenerateParams
		expectedPageRate float64
		expectedPageWin  time.Duration
		expectedTickRate float64
	}{
		{
			name: "Standard 30d Window",
			params: domain.GenerateParams{
				TargetSLO:   mustSLO(99.9),
				TotalWindow: 30 * 24 * time.Hour,
				PageTTE:     24 * time.Hour,
				TicketTTE:   3 * 24 * time.Hour,
			},
			expectedPageRate: 30.0,
			expectedPageWin:  28*time.Minute + 48*time.Second,
			expectedTickRate: 10.0,
		},
		{
			name: "Aggressive 7d Window",
			params: domain.GenerateParams{
				TargetSLO:   mustSLO(99.9),
				TotalWindow: 168 * time.Hour,
				PageTTE:     4 * time.Hour,
				TicketTTE:   24 * time.Hour,
			},
			expectedPageRate: 42.0,
			expectedPageWin:  4*time.Minute + 48*time.Second,
			expectedTickRate: 7.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := svc.GenerateThresholds(tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if math.Abs(res.PageRule.BurnRate-tc.expectedPageRate) > 0.01 {
				t.Errorf("Expected Page BurnRate %.2f, got %.2f", tc.expectedPageRate, res.PageRule.BurnRate)
			}
			diff := res.PageRule.RecallWindow - tc.expectedPageWin
			if diff < -time.Second || diff > time.Second {
				t.Errorf("Expected Page Window %v, got %v", tc.expectedPageWin, res.PageRule.RecallWindow)
			}

			if math.Abs(res.TicketRule.BurnRate-tc.expectedTickRate) > 0.01 {
				t.Errorf("Expected Ticket BurnRate %.2f, got %.2f", tc.expectedTickRate, res.TicketRule.BurnRate)
			}
		})
	}
}
