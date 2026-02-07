package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/haconeco/project-information-manager/internal/domain"
	"github.com/haconeco/project-information-manager/internal/repository"
)

// StockService はStockのビジネスロジックを提供する。
type StockService struct {
	stockRepo  repository.StockRepository
	vectorRepo repository.VectorRepository
}

// NewStockService は新しいStockServiceを生成する。
func NewStockService(stockRepo repository.StockRepository, vectorRepo repository.VectorRepository) *StockService {
	return &StockService{
		stockRepo:  stockRepo,
		vectorRepo: vectorRepo,
	}
}

// CreateInput はStock作成時の入力パラメータ。
type CreateStockInput struct {
	ProjectID  string
	Category   string
	Priority   string
	Title      string
	Content    string
	Tags       []string
	References []string
}

// Create は新しいStockを作成する。
func (s *StockService) Create(ctx context.Context, input CreateStockInput) (*domain.Stock, error) {
	// バリデーション
	category := domain.StockCategory(input.Category)
	if !isValidCategory(category) {
		return nil, domain.ErrInvalidCategory
	}

	priority, err := domain.ParsePriority(input.Priority)
	if err != nil {
		return nil, err
	}

	// ID生成
	id := generateStockID(category)

	now := time.Now()
	stock := &domain.Stock{
		ID:         id,
		ProjectID:  input.ProjectID,
		Category:   category,
		Priority:   priority,
		Title:      input.Title,
		Content:    input.Content,
		Tags:       input.Tags,
		References: input.References,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.stockRepo.Create(ctx, stock); err != nil {
		return nil, fmt.Errorf("failed to create stock: %w", err)
	}

	// ベクトルインデックスに追加（利用可能な場合）
	if s.vectorRepo != nil {
		metadata := map[string]string{
			"type":       "stock",
			"project_id": stock.ProjectID,
			"category":   string(stock.Category),
			"priority":   stock.Priority.String(),
		}
		if err := s.vectorRepo.Upsert(ctx, stock.ID, stock.Title+"\n"+stock.Content, metadata); err != nil {
			// ベクトルインデックスのエラーは致命的ではない
			fmt.Printf("warning: failed to index stock in vector DB: %v\n", err)
		}
	}

	return stock, nil
}

// Get は管理番号でStockを取得する。
func (s *StockService) Get(ctx context.Context, id string) (*domain.Stock, error) {
	return s.stockRepo.Get(ctx, id)
}

// UpdateStockInput はStock更新時の入力パラメータ。
type UpdateStockInput struct {
	Content    *string
	Priority   *string
	Tags       []string
	References []string
}

// Update はStockを更新する。
func (s *StockService) Update(ctx context.Context, id string, input UpdateStockInput) (*domain.Stock, error) {
	stock, err := s.stockRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Content != nil {
		stock.Content = *input.Content
	}
	if input.Priority != nil {
		p, err := domain.ParsePriority(*input.Priority)
		if err != nil {
			return nil, err
		}
		stock.Priority = p
	}
	if input.Tags != nil {
		stock.Tags = input.Tags
	}
	if input.References != nil {
		stock.References = input.References
	}

	stock.UpdatedAt = time.Now()

	if err := s.stockRepo.Update(ctx, stock); err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	// ベクトルインデックスを更新
	if s.vectorRepo != nil {
		metadata := map[string]string{
			"type":       "stock",
			"project_id": stock.ProjectID,
			"category":   string(stock.Category),
			"priority":   stock.Priority.String(),
		}
		_ = s.vectorRepo.Upsert(ctx, stock.ID, stock.Title+"\n"+stock.Content, metadata)
	}

	return stock, nil
}

// List はプロジェクト内のStockを一覧取得する。
func (s *StockService) List(ctx context.Context, projectID string, opts *repository.StockListOptions) ([]*domain.Stock, error) {
	return s.stockRepo.List(ctx, projectID, opts)
}

// ListSummary はプロジェクト内のStockをサマリビューで一覧取得する。
func (s *StockService) ListSummary(ctx context.Context, projectID string, opts *repository.StockListOptions) ([]domain.StockSummary, error) {
	stocks, err := s.stockRepo.List(ctx, projectID, opts)
	if err != nil {
		return nil, err
	}
	summaries := make([]domain.StockSummary, 0, len(stocks))
	for _, stock := range stocks {
		summaries = append(summaries, stock.ToSummary())
	}
	return summaries, nil
}

// Search はセマンティック検索でStockを検索する。
func (s *StockService) Search(ctx context.Context, query string, limit int, projectID string) ([]*domain.Stock, error) {
	if s.vectorRepo == nil {
		// ベクトルDBが未設定の場合は空結果を返す
		return nil, nil
	}

	filters := map[string]string{
		"type":       "stock",
		"project_id": projectID,
	}

	results, err := s.vectorRepo.Search(ctx, query, limit, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search stocks: %w", err)
	}

	var stocks []*domain.Stock
	for _, result := range results {
		stock, err := s.stockRepo.Get(ctx, result.ID)
		if err != nil {
			continue
		}
		stocks = append(stocks, stock)
	}

	return stocks, nil
}

// SearchSummary はセマンティック検索でStockをサマリビューで検索する。
func (s *StockService) SearchSummary(ctx context.Context, query string, limit int, projectID string) ([]domain.StockSummary, error) {
	stocks, err := s.Search(ctx, query, limit, projectID)
	if err != nil {
		return nil, err
	}
	summaries := make([]domain.StockSummary, 0, len(stocks))
	for _, stock := range stocks {
		summaries = append(summaries, stock.ToSummary())
	}
	return summaries, nil
}

func isValidCategory(c domain.StockCategory) bool {
	for _, valid := range domain.ValidStockCategories() {
		if c == valid {
			return true
		}
	}
	return false
}

// stockIDCounter はStock IDの連番カウンターとして使用する（簡易実装）。
var stockIDCounter int

func generateStockID(category domain.StockCategory) string {
	stockIDCounter++
	prefix := strings.ToUpper(string(category))
	return fmt.Sprintf("STK-%s-%03d", prefix, stockIDCounter)
}
