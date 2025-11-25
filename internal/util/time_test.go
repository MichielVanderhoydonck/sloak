package util_test

import (
	"testing"
	"time"

	"github.com/MichielVanderhoydonck/sloak/internal/util"
)

func TestParseTimeWindow(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectedDur time.Duration
		expectError bool
	}{
		{
			name:        "Standard Go Duration (hours)",
			input:       "720h",
			expectedDur: 720 * time.Hour,
			expectError: false,
		},
		{
			name:        "Custom 'd' unit (30 days)",
			input:       "30d",
			expectedDur: 30 * 24 * time.Hour, // 720h
			expectError: false,
		},
		{
			name:        "Custom 'w' unit (2 weeks)",
			input:       "2w",
			expectedDur: 2 * 7 * 24 * time.Hour, // 336h
			expectError: false,
		},
		{
			name:        "Custom 'y' unit (1 year)",
			input:       "1y",
			expectedDur: 365 * 24 * time.Hour, // 8760h
			expectError: false,
		},
		{
			name:        "Float 'd' unit (1.5 days)",
			input:       "1.5d",
			expectedDur: 36 * time.Hour, // 24h + 12h
			expectError: false,
		},
		{
			name:        "Standard Go Duration (minutes)",
			input:       "30m",
			expectedDur: 30 * time.Minute,
			expectError: false,
		},
		{
			name:        "Invalid format (no number)",
			input:       "d",
			expectError: true,
		},
		{
			name:        "Invalid format (just number)",
			input:       "30",
			expectError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- Act ---
			result, err := util.ParseTimeWindow(tc.input)

			// --- Assert ---
			if tc.expectError {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error, but got: %v", err)
				}
				if result != tc.expectedDur {
					t.Errorf("expected duration %v, but got %v", tc.expectedDur, result)
				}
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "Clean second duration",
			input:    21*time.Minute + 36*time.Second,
			expected: "21m36s",
		},
		{
			name:     "Sub-second duration (should round)",
			input:    (1*time.Hour + 30*time.Minute + 43*time.Second + 200*time.Millisecond),
			expected: "1h30m43s", // Rounds down
		},
		{
			name:     "Sub-second duration (should round up)",
			input:    (1*time.Hour + 30*time.Minute + 43*time.Second + 800*time.Millisecond),
			expected: "1h30m44s", // Rounds up
		},
		{
			name:     "Zero duration",
			input:    0,
			expected: "0s",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := util.FormatDuration(tc.input)

			if result != tc.expected {
				t.Errorf("expected format '%s', but got '%s'", tc.expected, result)
			}
		})
	}
}
