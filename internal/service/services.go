package service

import (
	"github.com/haconeco/project-information-manager/internal/repository"
)

// Services は全サービスを束ねる構造体。
type Services struct {
	Stock   *StockService
	State   *StateService
	Context *ContextService
}

// NewServices は設定に基づいて全サービスを初期化する。
func NewServices(repos *repository.Repositories) *Services {
	stockService := NewStockService(repos.Stock, repos.Vector)
	stateService := NewStateService(repos.State, repos.Vector)
	contextService := NewContextService(repos.Stock, repos.State, repos.Vector)

	return &Services{
		Stock:   stockService,
		State:   stateService,
		Context: contextService,
	}
}
