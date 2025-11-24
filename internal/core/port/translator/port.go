package translator

import (
	domain "github.com/MichielVanderhoydonck/sloak/internal/core/domain/translator"
)

type TranslatorService interface {
	Translate(params domain.TranslationParams) (domain.TranslationResult, error)
}