package alerting

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
)

type AlertGeneratorService interface {
	GenerateThresholds(params domain.GenerateParams) (domain.Result, error)
}
