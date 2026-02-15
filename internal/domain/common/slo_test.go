package common_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichielVanderhoydonck/sloak/internal/domain/common"
)

func TestNewSLOTarget(t *testing.T) {
	testCases := []struct {
		name        string
		input       float64
		expectError bool
		expectedErr error
	}{
		{
			name:        "Valid SLO (99.9)",
			input:       99.9,
			expectError: false,
		},
		{
			name:        "Valid SLO (0)",
			input:       0.0,
			expectError: false,
		},
		{
			name:        "Valid SLO (100)",
			input:       100.0,
			expectError: false,
		},
		{
			name:        "Invalid SLO (Too High)",
			input:       100.001,
			expectError: true,
			expectedErr: common.ErrInvalidSLOTarget,
		},
		{
			name:        "Invalid SLO (Too Low)",
			input:       -0.1,
			expectError: true,
			expectedErr: common.ErrInvalidSLOTarget,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sloTarget, err := common.NewSLOTarget(tc.input)

			if tc.expectError {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.input, sloTarget.Value)
			}
		})
	}
}

func TestSLOTarget_String(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		expected string
	}{
		{name: "SLO Target 99.9", input: 99.9, expected: "99.900%"},
		{name: "SLO Target 100", input: 100, expected: "100.000%"},
		{name: "SLO Target 0", input: 0, expected: "0.000%"},
		{name: "SLO Target 99.999", input: 99.999, expected: "99.999%"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sloTarget, _ := common.NewSLOTarget(tc.input)
			require.Equal(t, tc.expected, sloTarget.String())
		})
	}
}
