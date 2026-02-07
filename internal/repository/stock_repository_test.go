package repository

import (
	"context"
	"testing"
	"time"

	"github.com/haconeco/project-information-manager/internal/domain"
)

func TestFileStockRepositoryCRUDAndList(t *testing.T) {
	ctx := context.Background()
	repo := NewFileStockRepository(t.TempDir())

	stock1 := &domain.Stock{
		ID:        "STK-DESIGN-001",
		ProjectID: "proj-1",
		Category:  domain.CategoryDesign,
		Priority:  domain.PriorityP1,
		Title:     "Design Doc",
		Content:   "initial",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	stock2 := &domain.Stock{
		ID:        "STK-RULES-002",
		ProjectID: "proj-1",
		Category:  domain.CategoryRules,
		Priority:  domain.PriorityP2,
		Title:     "Rules",
		Content:   "rules",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	stock3 := &domain.Stock{
		ID:        "STK-ARCH-003",
		ProjectID: "proj-2",
		Category:  domain.CategoryArchitecture,
		Priority:  domain.PriorityP0,
		Title:     "Arch",
		Content:   "arch",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := repo.Create(ctx, stock1); err != nil {
		t.Fatalf("create stock1: %v", err)
	}
	if err := repo.Create(ctx, stock2); err != nil {
		t.Fatalf("create stock2: %v", err)
	}
	if err := repo.Create(ctx, stock3); err != nil {
		t.Fatalf("create stock3: %v", err)
	}
	if err := repo.Create(ctx, stock1); err != domain.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	got, err := repo.Get(ctx, stock1.ID)
	if err != nil {
		t.Fatalf("get stock1: %v", err)
	}
	if got.Title != stock1.Title {
		t.Fatalf("expected title %s, got %s", stock1.Title, got.Title)
	}

	stock1.Content = "updated"
	stock1.UpdatedAt = time.Now()
	if err := repo.Update(ctx, stock1); err != nil {
		t.Fatalf("update stock1: %v", err)
	}
	updated, err := repo.Get(ctx, stock1.ID)
	if err != nil {
		t.Fatalf("get updated stock1: %v", err)
	}
	if updated.Content != "updated" {
		t.Fatalf("expected updated content, got %s", updated.Content)
	}

	list, err := repo.List(ctx, "proj-1", &StockListOptions{Category: &stock1.Category})
	if err != nil {
		t.Fatalf("list with category filter: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 stock, got %d", len(list))
	}

	listAll, err := repo.List(ctx, "proj-1", nil)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(listAll) != 2 {
		t.Fatalf("expected 2 stocks for project, got %d", len(listAll))
	}

	limited, err := repo.List(ctx, "proj-1", &StockListOptions{Limit: 1})
	if err != nil {
		t.Fatalf("list with limit: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("expected 1 stock with limit, got %d", len(limited))
	}

	if err := repo.Delete(ctx, stock2.ID); err != nil {
		t.Fatalf("delete stock2: %v", err)
	}
	if _, err := repo.Get(ctx, stock2.ID); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	if _, err := repo.Get(ctx, "missing"); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound for missing stock, got %v", err)
	}
	if err := repo.Delete(ctx, "missing"); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound for delete missing, got %v", err)
	}
}
