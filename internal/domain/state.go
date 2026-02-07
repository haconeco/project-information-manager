package domain

import "time"

// StateType はStateの種別を表す。
type StateType string

const (
	StateTypeTask     StateType = "task"
	StateTypeIssue    StateType = "issue"
	StateTypeIncident StateType = "incident"
	StateTypeChange   StateType = "change"
)

// StateStatus はStateのライフサイクル上の状態を表す。
type StateStatus string

const (
	StatusOpen       StateStatus = "open"
	StatusInProgress StateStatus = "in_progress"
	StatusResolved   StateStatus = "resolved"
	StatusArchived   StateStatus = "archived"
)

// State はプロダクト開発プロジェクトの動的な状態情報を表す。
// チケット管理形式で、各トピックについての状態と対処を記述する。
// 完了したらアーカイブし、重要情報はStockに転記する。
type State struct {
	ID          string      `json:"id"`           // 管理番号 (例: "STA-TASK-042")
	ProjectID   string      `json:"project_id"`   // 所属プロジェクトID
	Type        StateType   `json:"type"`         // 種別
	Status      StateStatus `json:"status"`       // ステータス
	Priority    Priority    `json:"priority"`     // 優先度
	Title       string      `json:"title"`        // タイトル
	Description string      `json:"description"`  // 詳細説明
	Resolution  string      `json:"resolution"`   // 解決内容（resolved/archived時）
	Tags        []string    `json:"tags"`         // 検索用タグ
	References  []string    `json:"references"`   // 関連Stock/StateのID
	CreatedAt   time.Time   `json:"created_at"`   // 作成日時
	UpdatedAt   time.Time   `json:"updated_at"`   // 更新日時
	ArchivedAt  *time.Time  `json:"archived_at"`  // アーカイブ日時
}

// IsActive はStateがアクティブ（アーカイブされていない）かを返す。
func (s *State) IsActive() bool {
	return s.Status != StatusArchived
}

// Archive はStateをアーカイブ状態にする。
func (s *State) Archive(resolution string, now time.Time) {
	s.Status = StatusArchived
	s.Resolution = resolution
	s.ArchivedAt = &now
	s.UpdatedAt = now
}

// StateSummary はStateのサマリビュー。list/search時に使用し、
// Description/Resolution を含まないことでレスポンスのトークン消費を抑制する。
type StateSummary struct {
	ID        string      `json:"id"`
	ProjectID string      `json:"project_id"`
	Type      StateType   `json:"type"`
	Status    StateStatus `json:"status"`
	Priority  Priority    `json:"priority"`
	Title     string      `json:"title"`
	Tags      []string    `json:"tags"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// ToSummary はStateからStateSummaryを生成する。
func (s *State) ToSummary() StateSummary {
	return StateSummary{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Type:      s.Type,
		Status:    s.Status,
		Priority:  s.Priority,
		Title:     s.Title,
		Tags:      s.Tags,
		UpdatedAt: s.UpdatedAt,
	}
}
