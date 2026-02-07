package service

import (
	"context"
	"fmt"

	"github.com/haconeco/project-information-manager/internal/repository"
)

// ContextService はStock/Stateを横断してRAG検索を行い、
// コンテキスト情報を集約して返却するサービス。
// 旧SkillServiceの1:1 Stock→Skill生成を廃止し、
// サーバーサイドでのRAG統合検索に置き換えたもの。
type ContextService struct {
	stockRepo  repository.StockRepository
	stateRepo  repository.StateRepository
	vectorRepo repository.VectorRepository
}

// NewContextService は新しいContextServiceを生成する。
func NewContextService(
	stockRepo repository.StockRepository,
	stateRepo repository.StateRepository,
	vectorRepo repository.VectorRepository,
) *ContextService {
	return &ContextService{
		stockRepo:  stockRepo,
		stateRepo:  stateRepo,
		vectorRepo: vectorRepo,
	}
}

// ContextSearchResult はコンテキスト横断検索の結果。
type ContextSearchResult struct {
	Stocks []ContextSearchItem `json:"stocks"`
	States []ContextSearchItem `json:"states"`
	Total  int                 `json:"total"`
}

// ContextSearchItem は検索結果の各アイテム（Stock/State共通のサマリ）。
type ContextSearchItem struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`     // "stock" or "state"
	Title    string  `json:"title"`
	Category string  `json:"category"` // Stock: category, State: state_type
	Priority string  `json:"priority"`
	Status   string  `json:"status,omitempty"` // Stateのみ
	Score    float32 `json:"score,omitempty"`  // 類似度スコア
}

// Search はStock/Stateを横断してRAGセマンティック検索を行い、
// サマリビューで結果を返却する。
func (s *ContextService) Search(ctx context.Context, query string, projectID string, limit int) (*ContextSearchResult, error) {
	result := &ContextSearchResult{
		Stocks: make([]ContextSearchItem, 0),
		States: make([]ContextSearchItem, 0),
	}

	if s.vectorRepo == nil {
		// ベクトルDBが未設定の場合はフォールバック: Stock/State全件をリスト返却
		return s.fallbackSearch(ctx, projectID, limit)
	}

	// ベクトル検索（全タイプ横断）
	searchResults, err := s.vectorRepo.Search(ctx, query, limit*2, map[string]string{
		"project_id": projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search context: %w", err)
	}

	stockCount, stateCount := 0, 0

	for _, sr := range searchResults {
		if stockCount+stateCount >= limit {
			break
		}

		docType := sr.Metadata["type"]

		switch docType {
		case "stock":
			if stockCount >= limit/2+limit%2 {
				continue
			}
			stock, err := s.stockRepo.Get(ctx, sr.ID)
			if err != nil {
				continue
			}
			result.Stocks = append(result.Stocks, ContextSearchItem{
				ID:       stock.ID,
				Type:     "stock",
				Title:    stock.Title,
				Category: string(stock.Category),
				Priority: stock.Priority.String(),
				Score:    sr.Similarity,
			})
			stockCount++

		case "state":
			if stateCount >= limit/2+limit%2 {
				continue
			}
			state, err := s.stateRepo.Get(ctx, sr.ID)
			if err != nil {
				continue
			}
			result.States = append(result.States, ContextSearchItem{
				ID:       state.ID,
				Type:     "state",
				Title:    state.Title,
				Category: string(state.Type),
				Priority: state.Priority.String(),
				Status:   string(state.Status),
				Score:    sr.Similarity,
			})
			stateCount++
		}
	}

	result.Total = stockCount + stateCount
	return result, nil
}

// fallbackSearch はベクトルDBなしの場合のフォールバック。
// Stock/Stateの一覧からサマリを返却する。
func (s *ContextService) fallbackSearch(ctx context.Context, projectID string, limit int) (*ContextSearchResult, error) {
	result := &ContextSearchResult{
		Stocks: make([]ContextSearchItem, 0),
		States: make([]ContextSearchItem, 0),
	}

	// Stockを取得
	if s.stockRepo != nil {
		stocks, err := s.stockRepo.List(ctx, projectID, nil)
		if err == nil {
			for i, stock := range stocks {
				if i >= limit/2 {
					break
				}
				result.Stocks = append(result.Stocks, ContextSearchItem{
					ID:       stock.ID,
					Type:     "stock",
					Title:    stock.Title,
					Category: string(stock.Category),
					Priority: stock.Priority.String(),
				})
			}
		}
	}

	// Stateを取得
	if s.stateRepo != nil {
		states, err := s.stateRepo.List(ctx, projectID, nil)
		if err == nil {
			for i, state := range states {
				if i >= limit/2 {
					break
				}
				result.States = append(result.States, ContextSearchItem{
					ID:       state.ID,
					Type:     "state",
					Title:    state.Title,
					Category: string(state.Type),
					Priority: state.Priority.String(),
					Status:   string(state.Status),
				})
			}
		}
	}

	result.Total = len(result.Stocks) + len(result.States)
	return result, nil
}
