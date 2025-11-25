package errorbudget

import "github.com/MichielVanderhoydonck/sloak/internal/core/domain/errorbudget"

type CalculatorService interface {
	CalculateBudget(params errorbudget.CalculationParams) (errorbudget.BudgetResult, error)
}
