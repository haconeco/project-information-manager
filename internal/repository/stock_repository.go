package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/haconeco/project-information-manager/internal/domain"
)

// FileStockRepository はファイルシステムベースのStockリポジトリ実装。
// 各StockはJSON形式で個別ファイルに保存される。
type FileStockRepository struct {
	baseDir string
}

// NewFileStockRepository は新しいFileStockRepositoryを生成する。
func NewFileStockRepository(baseDir string) *FileStockRepository {
	return &FileStockRepository{baseDir: baseDir}
}

func (r *FileStockRepository) stockPath(id string) string {
	// IDからプロジェクトIDとカテゴリを推定してパスを構築
	// 例: "STK-DESIGN-001" → stocks/{projectID}/design/STK-DESIGN-001.json
	// 簡易実装: フラットにstocks/{id}.json
	return filepath.Join(r.baseDir, id+".json")
}

func (r *FileStockRepository) projectDir(projectID string) string {
	return filepath.Join(r.baseDir, projectID)
}

// Create は新しいStockをファイルとして保存する。
func (r *FileStockRepository) Create(ctx context.Context, stock *domain.Stock) error {
	path := r.stockPath(stock.ID)

	// 既存チェック
	if _, err := os.Stat(path); err == nil {
		return domain.ErrAlreadyExists
	}

	// ディレクトリ作成
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return r.writeStock(path, stock)
}

// Get は管理番号でStockを取得する。
func (r *FileStockRepository) Get(ctx context.Context, id string) (*domain.Stock, error) {
	path := r.stockPath(id)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to read stock file: %w", err)
	}

	var stock domain.Stock
	if err := json.Unmarshal(data, &stock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stock: %w", err)
	}

	return &stock, nil
}

// Update はStockを更新する。
func (r *FileStockRepository) Update(ctx context.Context, stock *domain.Stock) error {
	path := r.stockPath(stock.ID)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return domain.ErrNotFound
	}

	return r.writeStock(path, stock)
}

// Delete はStockを削除する。
func (r *FileStockRepository) Delete(ctx context.Context, id string) error {
	path := r.stockPath(id)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to delete stock file: %w", err)
	}

	return nil
}

// List はプロジェクト内のStockを一覧取得する。
func (r *FileStockRepository) List(ctx context.Context, projectID string, opts *StockListOptions) ([]*domain.Stock, error) {
	// baseDir以下のすべてのJSONファイルをスキャン
	var stocks []*domain.Stock

	err := filepath.Walk(r.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // skip unreadable files
		}

		var stock domain.Stock
		if err := json.Unmarshal(data, &stock); err != nil {
			return nil // skip invalid files
		}

		// projectID が指定されている場合のみフィルタ
		if projectID != "" && stock.ProjectID != projectID {
			return nil
		}

		// オプションによるフィルタ
		if opts != nil {
			if opts.Category != nil && stock.Category != *opts.Category {
				return nil
			}
			if opts.Priority != nil && stock.Priority != *opts.Priority {
				return nil
			}
		}

		stocks = append(stocks, &stock)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list stocks: %w", err)
	}

	// Limit/Offset
	if opts != nil {
		if opts.Offset > 0 && opts.Offset < len(stocks) {
			stocks = stocks[opts.Offset:]
		}
		if opts.Limit > 0 && opts.Limit < len(stocks) {
			stocks = stocks[:opts.Limit]
		}
	}

	return stocks, nil
}

func (r *FileStockRepository) writeStock(path string, stock *domain.Stock) error {
	data, err := json.MarshalIndent(stock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stock: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write stock file: %w", err)
	}

	return nil
}
