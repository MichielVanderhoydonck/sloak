package alerting

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/alerting"
)

type AlertGeneratorService interface {
	GenerateTable(params domain.GenerateParams) (domain.TableResult, error)
	GeneratePrometheus(params domain.GeneratePrometheusParams) (string, error)
}
