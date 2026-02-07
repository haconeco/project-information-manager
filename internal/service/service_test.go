package service

import (
	"context"
	"testing"

	"github.com/haconeco/project-information-manager/internal/domain"
	"github.com/haconeco/project-information-manager/internal/repository"
)

func TestStockServiceCreate(t *testing.T) {
	stockRepo := repository.NewFileStockRepository(t.TempDir())
	svc := NewStockService(stockRepo, nil)

	input := CreateStockInput{
		ProjectID: "test-project",
		Category:  "design",
		Priority:  "P1",
		Title:     "API設計",
		Content:   "# API設計\n\nREST APIの設計方針を記述する。",
		Tags:      []string{"api", "design"},
	}

	stock, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stock.ID == "" {
		t.Error("expected non-empty ID")
	}
	if stock.ProjectID != "test-project" {
		t.Errorf("expected project_id test-project, got %s", stock.ProjectID)
	}
	if stock.Category != domain.CategoryDesign {
		t.Errorf("expected category design, got %s", stock.Category)
	}
	if stock.Priority != domain.PriorityP1 {
		t.Errorf("expected priority P1, got %s", stock.Priority.String())
	}

	// 取得テスト
	got, err := svc.Get(context.Background(), stock.ID)
	if err != nil {
		t.Fatalf("failed to get stock: %v", err)
	}
	if got.Title != "API設計" {
		t.Errorf("expected title API設計, got %s", got.Title)
	}
}

func TestStockServiceCreateInvalidCategory(t *testing.T) {
	stockRepo := repository.NewFileStockRepository(t.TempDir())
	svc := NewStockService(stockRepo, nil)

	input := CreateStockInput{
		ProjectID: "test-project",
		Category:  "invalid",
		Priority:  "P1",
		Title:     "Test",
		Content:   "Test content",
	}

	_, err := svc.Create(context.Background(), input)
	if err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestStockServiceUpdate(t *testing.T) {
	stockRepo := repository.NewFileStockRepository(t.TempDir())
	svc := NewStockService(stockRepo, nil)

	// 作成
	input := CreateStockInput{
		ProjectID: "test-project",
		Category:  "rules",
		Priority:  "P2",
		Title:     "開発ルール",
		Content:   "初期内容",
	}
	stock, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 更新
	newContent := "更新された内容"
	newPriority := "P0"
	updated, err := svc.Update(context.Background(), stock.ID, UpdateStockInput{
		Content:  &newContent,
		Priority: &newPriority,
	})
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	if updated.Content != newContent {
		t.Errorf("expected content %s, got %s", newContent, updated.Content)
	}
	if updated.Priority != domain.PriorityP0 {
		t.Errorf("expected priority P0, got %s", updated.Priority.String())
	}
}

func TestContextServiceSearch(t *testing.T) {
	tmpDir := t.TempDir()
	stockRepo := repository.NewFileStockRepository(tmpDir + "/stocks")

	// SQLite State リポジトリ（テスト用にインメモリ不可のためtmpDir使用）
	// ContextServiceのfallbackSearchをテストする（vectorRepo = nil）
	stockSvc := NewStockService(stockRepo, nil)

	// Stockを作成
	stockInput := CreateStockInput{
		ProjectID: "test-project",
		Category:  "architecture",
		Priority:  "P0",
		Title:     "システムアーキテクチャ",
		Content:   "# アーキテクチャ\n\nマイクロサービスアーキテクチャを採用する。",
		Tags:      []string{"architecture", "microservices"},
	}
	stock, err := stockSvc.Create(context.Background(), stockInput)
	if err != nil {
		t.Fatalf("failed to create stock: %v", err)
	}

	// ContextService（ベクトルDBなしのフォールバック検索テスト）
	contextSvc := NewContextService(stockRepo, nil, nil)

	result, err := contextSvc.Search(context.Background(), "アーキテクチャ", "test-project", 10)
	if err != nil {
		t.Fatalf("failed to search context: %v", err)
	}

	if result.Total == 0 {
		t.Error("expected non-zero total results")
	}

	if len(result.Stocks) == 0 {
		t.Error("expected at least one stock in results")
	}

	if result.Stocks[0].ID != stock.ID {
		t.Errorf("expected stock ID %s, got %s", stock.ID, result.Stocks[0].ID)
	}

	if result.Stocks[0].Priority != "P0" {
		t.Errorf("expected priority P0, got %s", result.Stocks[0].Priority)
	}
}

func TestStockServiceListSummary(t *testing.T) {
	stockRepo := repository.NewFileStockRepository(t.TempDir())
	svc := NewStockService(stockRepo, nil)

	input := CreateStockInput{
		ProjectID: "test-project",
		Category:  "management",
		Priority:  "P2",
		Title:     "管理方針",
		Content:   "内容",
		Tags:      []string{"policy"},
	}
	stock, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	summaries, err := svc.ListSummary(context.Background(), "test-project", nil)
	if err != nil {
		t.Fatalf("list summary error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].ID != stock.ID {
		t.Fatalf("expected ID %s, got %s", stock.ID, summaries[0].ID)
	}
	if summaries[0].Priority != domain.PriorityP2 {
		t.Fatalf("expected priority P2, got %s", summaries[0].Priority.String())
	}
}

func TestStockServiceSearchSummaryWithVector(t *testing.T) {
	stockRepo := repository.NewFileStockRepository(t.TempDir())
	stockSvc := NewStockService(stockRepo, nil)

	stock, err := stockSvc.Create(context.Background(), CreateStockInput{
		ProjectID: "proj-1",
		Category:  "design",
		Priority:  "P1",
		Title:     "API設計",
		Content:   "content",
	})
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}

	vector := &fakeVectorRepo{results: []repository.SearchResult{{ID: stock.ID}}}
	searchSvc := NewStockService(stockRepo, vector)

	results, err := searchSvc.SearchSummary(context.Background(), "api", 10, "proj-1")
	if err != nil {
		t.Fatalf("search summary error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestStockSummary(t *testing.T) {
	stock := &domain.Stock{
		ID:        "STK-DESIGN-001",
		ProjectID: "test-project",
		Category:  domain.CategoryDesign,
		Priority:  domain.PriorityP1,
		Title:     "API設計",
		Content:   "大量のコンテンツ...",
		Tags:      []string{"api"},
	}

	summary := stock.ToSummary()

	if summary.ID != stock.ID {
		t.Errorf("expected ID %s, got %s", stock.ID, summary.ID)
	}
	if summary.Title != stock.Title {
		t.Errorf("expected title %s, got %s", stock.Title, summary.Title)
	}
	// Summary にはContentが含まれないことを型レベルで保証
	// (StockSummary構造体にContentフィールドがない)
}
