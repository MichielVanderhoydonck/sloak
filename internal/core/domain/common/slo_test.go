package common_test

import (
	"errors"
	"testing"

	"github.com/MichielVanderhoydonck/sloak/internal/core/domain/common"
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
			_, err := common.NewSLOTarget(tc.input)
			
			if tc.expectError {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error '%v', but got '%v'", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error, but got: %v", err)
				}
			}
		})
	}
}