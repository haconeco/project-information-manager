package repository

import (
	"context"

	"github.com/haconeco/project-information-manager/internal/domain"
)

// StockRepository はStockの永続化を担うインターフェース。
// ファイルシステムベースの実装を想定する。
type StockRepository interface {
	// Create は新しいStockを保存する。
	Create(ctx context.Context, stock *domain.Stock) error

	// Get は管理番号でStockを取得する。
	Get(ctx context.Context, id string) (*domain.Stock, error)

	// Update はStockを更新する。
	Update(ctx context.Context, stock *domain.Stock) error

	// Delete はStockを削除する。
	Delete(ctx context.Context, id string) error

	// List はプロジェクト内のStockを一覧取得する。
	List(ctx context.Context, projectID string, opts *StockListOptions) ([]*domain.Stock, error)
}

// StockListOptions はStock一覧取得時のフィルタリングオプション。
type StockListOptions struct {
	Category *domain.StockCategory
	Priority *domain.Priority
	Tags     []string
	Limit    int
	Offset   int
}

// StateRepository はStateの永続化を担うインターフェース。
// SQLiteベースの実装を想定する。
type StateRepository interface {
	// Create は新しいStateを保存する。
	Create(ctx context.Context, state *domain.State) error

	// Get は管理番号でStateを取得する。
	Get(ctx context.Context, id string) (*domain.State, error)

	// Update はStateを更新する。
	Update(ctx context.Context, state *domain.State) error

	// List はプロジェクト内のStateを一覧取得する。
	List(ctx context.Context, projectID string, opts *StateListOptions) ([]*domain.State, error)
}

// StateListOptions はState一覧取得時のフィルタリングオプション。
type StateListOptions struct {
	Type           *domain.StateType
	Status         *domain.StateStatus
	Priority       *domain.Priority
	IncludeArchived bool
	Limit          int
	Offset         int
}

// VectorRepository はベクトルインデックスの管理を担うインターフェース。
// chromem-goベースの実装を想定する。
type VectorRepository interface {
	// Upsert はドキュメントをベクトルインデックスに追加・更新する。
	Upsert(ctx context.Context, id string, content string, metadata map[string]string) error

	// Search はセマンティック検索を実行する。
	Search(ctx context.Context, query string, limit int, filters map[string]string) ([]SearchResult, error)

	// Delete はドキュメントをベクトルインデックスから削除する。
	Delete(ctx context.Context, id string) error
}

// SearchResult はベクトル検索の結果を表す。
type SearchResult struct {
	ID         string            // ドキュメントID
	Content    string            // ドキュメント内容
	Metadata   map[string]string // メタデータ
	Similarity float32           // 類似度スコア (0.0 ~ 1.0)
}
