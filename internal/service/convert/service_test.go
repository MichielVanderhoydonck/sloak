package convert_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/convert"
	convertSvc "github.com/MichielVanderhoydonck/sloak/internal/service/convert"
)

func TestConvertService(t *testing.T) {
	service := convertSvc.NewConvertService()

	t.Run("converts from nines successfully", func(t *testing.T) {
		params := convert.ConversionParams{
			Mode:  convert.ModeFromNines,
			Nines: 99.9,
		}

		res, err := service.Convert(params)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if math.Abs(res.AvailabilityPercent-99.9) > 0.001 {
			t.Errorf("expected availability ~99.9, got %f", res.AvailabilityPercent)
		}

		// 0.1% of 24h = 86.4s, which rounds to 86s. The current implementation rounds to 1m26s.
		expectedDowntime := 1*time.Minute + 26*time.Second
		if res.DailyDowntime != expectedDowntime {
			t.Errorf("expected daily downtime %v, got %v", expectedDowntime, res.DailyDowntime)
		}
	})

	t.Run("converts from downtime successfully", func(t *testing.T) {
		params := convert.ConversionParams{
			Mode:         convert.ModeFromDowntime,
			Downtime:     1 * time.Hour,
			CustomWindow: 24 * time.Hour,
		}

		res, err := service.Convert(params)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedAvailability := 95.833333
		if math.Abs(res.AvailabilityPercent-expectedAvailability) > 0.001 {
			t.Errorf("expected availability ~%f, got %f", expectedAvailability, res.AvailabilityPercent)
		}
	})

	t.Run("returns validation error for invalid nines", func(t *testing.T) {
		params := convert.ConversionParams{
			Mode:  convert.ModeFromNines,
			Nines: 101,
		}

		_, err := service.Convert(params)
		expectedErr := errors.New("availability must be between 0 and 100")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if err.Error() != expectedErr.Error() {
			t.Errorf("expected error '%v', got '%v'", expectedErr, err)
		}
	})
}
