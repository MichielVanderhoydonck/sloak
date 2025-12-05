package common

import (
	"errors"
	"fmt"
)

type SLOTarget struct {
	Value float64
}

var ErrInvalidSLOTarget = errors.New("SLO target must be between 0.0 and 100.0")

func NewSLOTarget(value float64) (SLOTarget, error) {
	if value < 0.0 || value > 100.0 {
		return SLOTarget{}, ErrInvalidSLOTarget
	}
	return SLOTarget{Value: value}, nil
}

func (s SLOTarget) String() string {
	return fmt.Sprintf("%.3f%%", s.Value)
}
