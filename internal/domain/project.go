package domain

import "time"

// Project はプロダクト開発プロジェクトの情報を表す。
type Project struct {
	ID          string    `json:"id"`          // プロジェクトID
	Name        string    `json:"name"`        // プロジェクト名
	Description string    `json:"description"` // プロジェクト説明
	Goal        string    `json:"goal"`        // プロダクトゴール
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
