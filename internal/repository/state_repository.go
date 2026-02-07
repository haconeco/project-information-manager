package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/haconeco/project-information-manager/internal/domain"
)

// SQLiteStateRepository はSQLiteベースのStateリポジトリ実装。
type SQLiteStateRepository struct {
	db *sql.DB
}

// NewSQLiteStateRepository は新しいSQLiteStateRepositoryを生成する。
func NewSQLiteStateRepository(db *sql.DB) (*SQLiteStateRepository, error) {
	repo := &SQLiteStateRepository{db: db}
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate states table: %w", err)
	}
	return repo, nil
}

func (r *SQLiteStateRepository) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS states (
		id          TEXT PRIMARY KEY,
		project_id  TEXT NOT NULL,
		type        TEXT NOT NULL,
		status      TEXT NOT NULL DEFAULT 'open',
		priority    INTEGER NOT NULL DEFAULT 3,
		title       TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		resolution  TEXT NOT NULL DEFAULT '',
		tags        TEXT NOT NULL DEFAULT '[]',
		ref_ids     TEXT NOT NULL DEFAULT '[]',
		created_at  DATETIME NOT NULL,
		updated_at  DATETIME NOT NULL,
		archived_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_states_project_id ON states(project_id);
	CREATE INDEX IF NOT EXISTS idx_states_status ON states(status);
	CREATE INDEX IF NOT EXISTS idx_states_type ON states(type);
	CREATE INDEX IF NOT EXISTS idx_states_priority ON states(priority);
	`
	_, err := r.db.Exec(query)
	return err
}

// Create は新しいStateをSQLiteに保存する。
func (r *SQLiteStateRepository) Create(ctx context.Context, state *domain.State) error {
	tagsJSON := "[]"
	if len(state.Tags) > 0 {
		tagsJSON = `["` + strings.Join(state.Tags, `","`) + `"]`
	}
	refsJSON := "[]"
	if len(state.References) > 0 {
		refsJSON = `["` + strings.Join(state.References, `","`) + `"]`
	}

	query := `
	INSERT INTO states (id, project_id, type, status, priority, title, description, resolution, tags, ref_ids, created_at, updated_at, archived_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		state.ID,
		state.ProjectID,
		string(state.Type),
		string(state.Status),
		int(state.Priority),
		state.Title,
		state.Description,
		state.Resolution,
		tagsJSON,
		refsJSON,
		state.CreatedAt,
		state.UpdatedAt,
		state.ArchivedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert state: %w", err)
	}
	return nil
}

// Get は管理番号でStateを取得する。
func (r *SQLiteStateRepository) Get(ctx context.Context, id string) (*domain.State, error) {
	query := `
	SELECT id, project_id, type, status, priority, title, description, resolution, tags, ref_ids, created_at, updated_at, archived_at
	FROM states WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanState(row)
}

// Update はStateを更新する。
func (r *SQLiteStateRepository) Update(ctx context.Context, state *domain.State) error {
	tagsJSON := "[]"
	if len(state.Tags) > 0 {
		tagsJSON = `["` + strings.Join(state.Tags, `","`) + `"]`
	}
	refsJSON := "[]"
	if len(state.References) > 0 {
		refsJSON = `["` + strings.Join(state.References, `","`) + `"]`
	}

	query := `
	UPDATE states
	SET project_id = ?, type = ?, status = ?, priority = ?, title = ?, description = ?,
	    resolution = ?, tags = ?, ref_ids = ?, updated_at = ?, archived_at = ?
	WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		state.ProjectID,
		string(state.Type),
		string(state.Status),
		int(state.Priority),
		state.Title,
		state.Description,
		state.Resolution,
		tagsJSON,
		refsJSON,
		state.UpdatedAt,
		state.ArchivedAt,
		state.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// List はプロジェクト内のStateを一覧取得する。
func (r *SQLiteStateRepository) List(ctx context.Context, projectID string, opts *StateListOptions) ([]*domain.State, error) {
	var conditions []string
	var args []any

	if projectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, projectID)
	}

	if opts != nil {
		if opts.Type != nil {
			conditions = append(conditions, "type = ?")
			args = append(args, string(*opts.Type))
		}
		if opts.Status != nil {
			conditions = append(conditions, "status = ?")
			args = append(args, string(*opts.Status))
		}
		if opts.Priority != nil {
			conditions = append(conditions, "priority = ?")
			args = append(args, int(*opts.Priority))
		}
		if !opts.IncludeArchived {
			conditions = append(conditions, "status != 'archived'")
		}
	} else {
		conditions = append(conditions, "status != 'archived'")
	}

	query := `
	SELECT id, project_id, type, status, priority, title, description, resolution, tags, ref_ids, created_at, updated_at, archived_at
	FROM states`
	if len(conditions) > 0 {
		query += `
	WHERE ` + strings.Join(conditions, " AND ")
	}
	query += `
	ORDER BY priority ASC, updated_at DESC
	`

	if opts != nil && opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query states: %w", err)
	}
	defer rows.Close()

	var states []*domain.State
	for rows.Next() {
		state, err := r.scanStateRows(rows)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func (r *SQLiteStateRepository) scanState(row *sql.Row) (*domain.State, error) {
	var (
		state      domain.State
		typeStr    string
		statusStr  string
		priority   int
		tagsJSON   string
		refsJSON   string
		archivedAt sql.NullTime
	)

	err := row.Scan(
		&state.ID,
		&state.ProjectID,
		&typeStr,
		&statusStr,
		&priority,
		&state.Title,
		&state.Description,
		&state.Resolution,
		&tagsJSON,
		&refsJSON,
		&state.CreatedAt,
		&state.UpdatedAt,
		&archivedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan state: %w", err)
	}

	state.Type = domain.StateType(typeStr)
	state.Status = domain.StateStatus(statusStr)
	state.Priority = domain.Priority(priority)
	state.Tags = parseJSONStringArray(tagsJSON)
	state.References = parseJSONStringArray(refsJSON)
	if archivedAt.Valid {
		state.ArchivedAt = &archivedAt.Time
	}

	return &state, nil
}

func (r *SQLiteStateRepository) scanStateRows(rows *sql.Rows) (*domain.State, error) {
	var (
		state      domain.State
		typeStr    string
		statusStr  string
		priority   int
		tagsJSON   string
		refsJSON   string
		archivedAt sql.NullTime
	)

	err := rows.Scan(
		&state.ID,
		&state.ProjectID,
		&typeStr,
		&statusStr,
		&priority,
		&state.Title,
		&state.Description,
		&state.Resolution,
		&tagsJSON,
		&refsJSON,
		&state.CreatedAt,
		&state.UpdatedAt,
		&archivedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan state row: %w", err)
	}

	state.Type = domain.StateType(typeStr)
	state.Status = domain.StateStatus(statusStr)
	state.Priority = domain.Priority(priority)
	state.Tags = parseJSONStringArray(tagsJSON)
	state.References = parseJSONStringArray(refsJSON)
	if archivedAt.Valid {
		state.ArchivedAt = &archivedAt.Time
	}

	return &state, nil
}

// parseJSONStringArray は JSON 配列文字列を []string にパースする。
func parseJSONStringArray(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return nil
	}
	// 簡易パース: ["a","b","c"] → [a, b, c]
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// 以下はテスト用のヘルパー関数。将来的に使用可能。
func timePtr(t time.Time) *time.Time {
	return &t
}
