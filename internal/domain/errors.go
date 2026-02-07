package domain

import "errors"

// ドメインエラー定義
var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrInvalidPriority = errors.New("invalid priority: must be P0, P1, P2, or P3")
	ErrInvalidCategory = errors.New("invalid stock category")
	ErrInvalidStatus   = errors.New("invalid state status")
	ErrInvalidType     = errors.New("invalid state type")
	ErrArchived        = errors.New("state is already archived")
)
