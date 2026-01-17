package convert

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/convert"
)

type ConvertService interface {
	Convert(params domain.ConversionParams) (domain.ConversionResult, error)
}
