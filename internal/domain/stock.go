package domain

import "time"

// Priority はStockおよびStateの優先度を表す。
type Priority int

const (
	PriorityP0 Priority = iota // 最高: 常時ロード（プロダクトゴール、アーキテクチャ方針）
	PriorityP1                 // 高: セッション開始時ロード（概要設計、管理方針）
	PriorityP2                 // 中: RAGオンデマンドロード（基本設計、方式設計）
	PriorityP3                 // 低: 明示的クエリ時のみ（詳細実装メモ、過去議事録）
)

// String はPriorityの文字列表現を返す。
func (p Priority) String() string {
	switch p {
	case PriorityP0:
		return "P0"
	case PriorityP1:
		return "P1"
	case PriorityP2:
		return "P2"
	case PriorityP3:
		return "P3"
	default:
		return "unknown"
	}
}

// ParsePriority は文字列からPriorityを解析する。
func ParsePriority(s string) (Priority, error) {
	switch s {
	case "P0":
		return PriorityP0, nil
	case "P1":
		return PriorityP1, nil
	case "P2":
		return PriorityP2, nil
	case "P3":
		return PriorityP3, nil
	default:
		return PriorityP3, ErrInvalidPriority
	}
}

// StockCategory はStockのカテゴリを表す。
type StockCategory string

const (
	CategoryDesign       StockCategory = "design"
	CategoryRules        StockCategory = "rules"
	CategoryManagement   StockCategory = "management"
	CategoryArchitecture StockCategory = "architecture"
	CategoryRequirement  StockCategory = "requirement"
	CategoryTest         StockCategory = "test"
)

// ValidStockCategories は有効なStockCategoryの一覧を返す。
func ValidStockCategories() []StockCategory {
	return []StockCategory{
		CategoryDesign,
		CategoryRules,
		CategoryManagement,
		CategoryArchitecture,
		CategoryRequirement,
		CategoryTest,
	}
}

// Stock は静的に定義されるプロダクト情報（設計、ルール、方針など）を表す。
// Wikiのように構造化された知識として永続化される。
type Stock struct {
	ID         string        `json:"id"`          // 管理番号 (例: "STK-DESIGN-001")
	ProjectID  string        `json:"project_id"`  // 所属プロジェクトID
	Category   StockCategory `json:"category"`    // カテゴリ
	Priority   Priority      `json:"priority"`    // 参照優先度
	Title      string        `json:"title"`       // タイトル
	Content    string        `json:"content"`     // Markdown形式の本文
	Tags       []string      `json:"tags"`        // 検索用タグ
	References []string      `json:"references"`  // 関連Stock/StateのID
	CreatedAt  time.Time     `json:"created_at"`  // 作成日時
	UpdatedAt  time.Time     `json:"updated_at"`  // 更新日時
}

// StockSummary はStockのサマリビュー。list/search時に使用し、
// Content を含まないことでレスポンスのトークン消費を抑制する。
type StockSummary struct {
	ID        string        `json:"id"`
	ProjectID string        `json:"project_id"`
	Category  StockCategory `json:"category"`
	Priority  Priority      `json:"priority"`
	Title     string        `json:"title"`
	Tags      []string      `json:"tags"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ToSummary はStockからStockSummaryを生成する。
func (s *Stock) ToSummary() StockSummary {
	return StockSummary{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Category:  s.Category,
		Priority:  s.Priority,
		Title:     s.Title,
		Tags:      s.Tags,
		UpdatedAt: s.UpdatedAt,
	}
}
