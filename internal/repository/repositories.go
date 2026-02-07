package repository

import (
	"database/sql"
	"fmt"

	"github.com/haconeco/project-information-manager/internal/config"

	_ "modernc.org/sqlite"
)

// Repositories は全リポジトリを束ねる構造体。
type Repositories struct {
	Stock  StockRepository
	State  StateRepository
	Vector VectorRepository

	db *sql.DB // closeのために保持
}

// NewRepositories は設定に基づいて全リポジトリを初期化する。
func NewRepositories(cfg *config.Config) (*Repositories, error) {
	// SQLite接続
	db, err := sql.Open("sqlite", cfg.StatesDBPath())
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// WALモード有効化（並行読み取りの性能向上）
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// State リポジトリ
	stateRepo, err := NewSQLiteStateRepository(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Repositories{
		Stock: NewFileStockRepository(cfg.StocksDir()),
		State: stateRepo,
		// TODO: Vector リポジトリは chromem-go 統合時に実装
		db: db,
	}, nil
}

// Close はリポジトリのリソースを解放する。
func (r *Repositories) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
