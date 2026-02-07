package service

import (
	"context"
	"log/slog"

	"github.com/haconeco/project-information-manager/internal/domain"
	"github.com/haconeco/project-information-manager/internal/repository"
)

// Services は全サービスを束ねる構造体。
type Services struct {
	Stock   *StockService
	State   *StateService
	Context *ContextService

	vectorRepo repository.VectorRepository
}

// NewServices は設定に基づいて全サービスを初期化する。
func NewServices(repos *repository.Repositories) *Services {
	stockService := NewStockService(repos.Stock, repos.Vector)
	stateService := NewStateService(repos.State, repos.Vector)
	contextService := NewContextService(repos.Stock, repos.State, repos.Vector)

	return &Services{
		Stock:      stockService,
		State:      stateService,
		Context:    contextService,
		vectorRepo: repos.Vector,
	}
}

// BootstrapVectorIndex は既存データのうち未インデックス分だけをベクトルDBへ補完する。
func (s *Services) BootstrapVectorIndex(ctx context.Context) error {
	if s.vectorRepo == nil {
		return nil
	}

	stocks, err := s.Stock.stockRepo.List(ctx, "", nil)
	if err != nil {
		return err
	}
	for _, stock := range stocks {
		exists, err := s.vectorRepo.Exists(ctx, stock.ID)
		if err != nil {
			slog.Warn("failed to check stock vector existence", "stock_id", stock.ID, "error", err)
			continue
		}
		if exists {
			continue
		}

		metadata := map[string]string{
			"type":       "stock",
			"project_id": stock.ProjectID,
			"category":   string(stock.Category),
			"priority":   stock.Priority.String(),
		}
		if err := s.vectorRepo.Upsert(ctx, stock.ID, stock.Title+"\n"+stock.Content, metadata); err != nil {
			slog.Warn("failed to upsert stock vector", "stock_id", stock.ID, "error", err)
		}
	}

	states, err := s.State.stateRepo.List(ctx, "", &repository.StateListOptions{IncludeArchived: true})
	if err != nil {
		return err
	}
	for _, state := range states {
		exists, err := s.vectorRepo.Exists(ctx, state.ID)
		if err != nil {
			slog.Warn("failed to check state vector existence", "state_id", state.ID, "error", err)
			continue
		}

		if state.Status == domain.StatusArchived {
			if exists {
				if err := s.vectorRepo.Delete(ctx, state.ID); err != nil {
					slog.Warn("failed to delete archived state vector", "state_id", state.ID, "error", err)
				}
			}
			continue
		}
		if exists {
			continue
		}

		metadata := map[string]string{
			"type":       "state",
			"project_id": state.ProjectID,
			"state_type": string(state.Type),
			"status":     string(state.Status),
			"priority":   state.Priority.String(),
		}
		if err := s.vectorRepo.Upsert(ctx, state.ID, state.Title+"\n"+state.Description, metadata); err != nil {
			slog.Warn("failed to upsert state vector", "state_id", state.ID, "error", err)
		}
	}

	return nil
}
