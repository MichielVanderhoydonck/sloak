package translator_test

import (
	"testing"
	"time"

	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/translator"
	service "github.com/MichielVanderhoydonck/sloak/internal/core/service/translator"
)

func TestTranslatorService(t *testing.T) {
	svc := service.NewTranslatorService()

	// Helper for tolerance check
	approx := func(a, b time.Duration) bool {
		diff := a - b
		if diff < 0 {
			diff = -diff
		}
		return diff <= time.Second // Allow 1s tolerance for rounding
	}

	t.Run("From Nines (99.9%)", func(t *testing.T) {
		params := domain.TranslationParams{
			Mode:  domain.ModeFromNines,
			Nines: 99.9,
		}
		res, err := svc.Translate(params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 99.9% = 0.1% downtime.
		// Daily (24h) = 1.44 mins = 1m 26s roughly
		expectedDaily := (1 * time.Minute) + (26 * time.Second)

		if !approx(res.DailyDowntime, expectedDaily) {
			t.Errorf("Expected daily ~1m26s, got %v", res.DailyDowntime)
		}
	})

	t.Run("From Duration (1h in 100h window)", func(t *testing.T) {
		params := domain.TranslationParams{
			Mode:         domain.ModeFromDowntime,
			Downtime:     1 * time.Hour,
			CustomWindow: 100 * time.Hour,
		}
		// 1h down in 100h = 1% fail = 99% availability
		res, err := svc.Translate(params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.AvailabilityPercent != 99.0 {
			t.Errorf("Expected 99.0%%, got %f", res.AvailabilityPercent)
		}
	})
}
