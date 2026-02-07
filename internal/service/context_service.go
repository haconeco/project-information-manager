package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/haconeco/project-information-manager/internal/domain"
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
	Type     string  `json:"type"` // "stock" or "state"
	Title    string  `json:"title"`
	Category string  `json:"category"` // Stock: category, State: state_type
	Priority string  `json:"priority"`
	Status   string  `json:"status,omitempty"` // Stateのみ
	Score    float32 `json:"score,omitempty"`  // 類似度スコア
}

type scoredContextItem struct {
	item      ContextSearchItem
	weighted  float32
	updatedAt time.Time
}

// Search はStock/Stateを横断してRAGセマンティック検索を行い、
// サマリビューで結果を返却する。
func (s *ContextService) Search(ctx context.Context, query string, projectID string, limit int) (*ContextSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	if s.vectorRepo == nil {
		return s.fallbackSearch(ctx, query, projectID, limit)
	}

	searchLimit := limit * 4
	if searchLimit < limit {
		searchLimit = limit
	}
	searchResults, err := s.vectorRepo.Search(ctx, query, searchLimit, map[string]string{
		"project_id": projectID,
	})
	if err != nil {
		slog.Warn("vector context search failed, fallback to keyword search", "error", err)
		return s.fallbackSearch(ctx, query, projectID, limit)
	}

	candidates := make([]scoredContextItem, 0, len(searchResults))
	for _, sr := range searchResults {
		docType := sr.Metadata["type"]
		switch docType {
		case "stock":
			stock, err := s.stockRepo.Get(ctx, sr.ID)
			if err != nil {
				continue
			}
			if projectID != "" && stock.ProjectID != projectID {
				continue
			}
			weighted := sr.Similarity * priorityWeight(stock.Priority)
			candidates = append(candidates, scoredContextItem{
				item: ContextSearchItem{
					ID:       stock.ID,
					Type:     "stock",
					Title:    stock.Title,
					Category: string(stock.Category),
					Priority: stock.Priority.String(),
					Score:    weighted,
				},
				weighted:  weighted,
				updatedAt: stock.UpdatedAt,
			})

		case "state":
			state, err := s.stateRepo.Get(ctx, sr.ID)
			if err != nil {
				continue
			}
			if projectID != "" && state.ProjectID != projectID {
				continue
			}
			if state.Status == domain.StatusArchived {
				continue
			}
			weighted := sr.Similarity * priorityWeight(state.Priority)
			candidates = append(candidates, scoredContextItem{
				item: ContextSearchItem{
					ID:       state.ID,
					Type:     "state",
					Title:    state.Title,
					Category: string(state.Type),
					Priority: state.Priority.String(),
					Status:   string(state.Status),
					Score:    weighted,
				},
				weighted:  weighted,
				updatedAt: state.UpdatedAt,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].weighted != candidates[j].weighted {
			return candidates[i].weighted > candidates[j].weighted
		}
		return candidates[i].updatedAt.After(candidates[j].updatedAt)
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	result := &ContextSearchResult{
		Stocks: make([]ContextSearchItem, 0, len(candidates)),
		States: make([]ContextSearchItem, 0, len(candidates)),
	}
	for _, c := range candidates {
		if c.item.Type == "stock" {
			result.Stocks = append(result.Stocks, c.item)
		} else if c.item.Type == "state" {
			result.States = append(result.States, c.item)
		}
	}
	result.Total = len(result.Stocks) + len(result.States)
	return result, nil
}

// fallbackSearch はベクトルDBなしの場合のフォールバック。
// タイトル・本文・タグの部分一致で検索し、優先度と更新日時で並び替える。
func (s *ContextService) fallbackSearch(ctx context.Context, query string, projectID string, limit int) (*ContextSearchResult, error) {
	candidates := make([]scoredContextItem, 0)

	if s.stockRepo != nil {
		stocks, err := s.stockRepo.List(ctx, projectID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list stocks for fallback search: %w", err)
		}
		for _, stock := range stocks {
			if !matchesQuery(query, stock.Title, stock.Content, joinTags(stock.Tags)) {
				continue
			}
			score := priorityWeight(stock.Priority)
			candidates = append(candidates, scoredContextItem{
				item: ContextSearchItem{
					ID:       stock.ID,
					Type:     "stock",
					Title:    stock.Title,
					Category: string(stock.Category),
					Priority: stock.Priority.String(),
					Score:    score,
				},
				weighted:  score,
				updatedAt: stock.UpdatedAt,
			})
		}
	}

	if s.stateRepo != nil {
		states, err := s.stateRepo.List(ctx, projectID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list states for fallback search: %w", err)
		}
		for _, state := range states {
			if state.Status == domain.StatusArchived {
				continue
			}
			if !matchesQuery(query, state.Title, state.Description, joinTags(state.Tags)) {
				continue
			}
			score := priorityWeight(state.Priority)
			candidates = append(candidates, scoredContextItem{
				item: ContextSearchItem{
					ID:       state.ID,
					Type:     "state",
					Title:    state.Title,
					Category: string(state.Type),
					Priority: state.Priority.String(),
					Status:   string(state.Status),
					Score:    score,
				},
				weighted:  score,
				updatedAt: state.UpdatedAt,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].weighted != candidates[j].weighted {
			return candidates[i].weighted > candidates[j].weighted
		}
		return candidates[i].updatedAt.After(candidates[j].updatedAt)
	})

	if limit <= 0 {
		limit = 10
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	result := &ContextSearchResult{
		Stocks: make([]ContextSearchItem, 0, len(candidates)),
		States: make([]ContextSearchItem, 0, len(candidates)),
	}
	for _, c := range candidates {
		if c.item.Type == "stock" {
			result.Stocks = append(result.Stocks, c.item)
		} else if c.item.Type == "state" {
			result.States = append(result.States, c.item)
		}
	}
	result.Total = len(result.Stocks) + len(result.States)
	return result, nil
}
