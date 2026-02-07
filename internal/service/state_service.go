package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/haconeco/project-information-manager/internal/domain"
	"github.com/haconeco/project-information-manager/internal/repository"
)

// StateService はStateのビジネスロジックを提供する。
type StateService struct {
	stateRepo  repository.StateRepository
	vectorRepo repository.VectorRepository
}

// NewStateService は新しいStateServiceを生成する。
func NewStateService(stateRepo repository.StateRepository, vectorRepo repository.VectorRepository) *StateService {
	return &StateService{
		stateRepo:  stateRepo,
		vectorRepo: vectorRepo,
	}
}

// CreateStateInput はState作成時の入力パラメータ。
type CreateStateInput struct {
	ProjectID   string
	Type        string
	Priority    string
	Title       string
	Description string
	Tags        []string
	References  []string
}

// Create は新しいStateを作成する。
func (s *StateService) Create(ctx context.Context, input CreateStateInput) (*domain.State, error) {
	stateType := domain.StateType(input.Type)
	if !isValidStateType(stateType) {
		return nil, domain.ErrInvalidType
	}

	priority, err := domain.ParsePriority(input.Priority)
	if err != nil {
		return nil, err
	}

	id := generateStateID(stateType)

	now := time.Now()
	state := &domain.State{
		ID:          id,
		ProjectID:   input.ProjectID,
		Type:        stateType,
		Status:      domain.StatusOpen,
		Priority:    priority,
		Title:       input.Title,
		Description: input.Description,
		Tags:        input.Tags,
		References:  input.References,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.stateRepo.Create(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}

	// ベクトルインデックスに追加
	if s.vectorRepo != nil {
		metadata := map[string]string{
			"type":       "state",
			"project_id": state.ProjectID,
			"state_type": string(state.Type),
			"status":     string(state.Status),
			"priority":   state.Priority.String(),
		}
		_ = s.vectorRepo.Upsert(ctx, state.ID, state.Title+"\n"+state.Description, metadata)
	}

	return state, nil
}

// Get は管理番号でStateを取得する。
func (s *StateService) Get(ctx context.Context, id string) (*domain.State, error) {
	return s.stateRepo.Get(ctx, id)
}

// UpdateStateInput はState更新時の入力パラメータ。
type UpdateStateInput struct {
	Status      *string
	Description *string
	Resolution  *string
	Priority    *string
	Tags        []string
	References  []string
}

// Update はStateを更新する。
func (s *StateService) Update(ctx context.Context, id string, input UpdateStateInput) (*domain.State, error) {
	state, err := s.stateRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if state.Status == domain.StatusArchived {
		return nil, domain.ErrArchived
	}

	if input.Status != nil {
		state.Status = domain.StateStatus(*input.Status)
	}
	if input.Description != nil {
		state.Description = *input.Description
	}
	if input.Resolution != nil {
		state.Resolution = *input.Resolution
	}
	if input.Priority != nil {
		p, err := domain.ParsePriority(*input.Priority)
		if err != nil {
			return nil, err
		}
		state.Priority = p
	}
	if input.Tags != nil {
		state.Tags = input.Tags
	}
	if input.References != nil {
		state.References = input.References
	}

	state.UpdatedAt = time.Now()

	if err := s.stateRepo.Update(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	if s.vectorRepo != nil {
		if state.Status == domain.StatusArchived {
			_ = s.vectorRepo.Delete(ctx, state.ID)
		} else {
			metadata := map[string]string{
				"type":       "state",
				"project_id": state.ProjectID,
				"state_type": string(state.Type),
				"status":     string(state.Status),
				"priority":   state.Priority.String(),
			}
			_ = s.vectorRepo.Upsert(ctx, state.ID, state.Title+"\n"+state.Description, metadata)
		}
	}

	return state, nil
}

// ArchiveInput はStateアーカイブ時の入力パラメータ。
type ArchiveInput struct {
	Resolution   string
	StockSummary string // Stockに転記する要約（空の場合は転記しない）
}

// Archive はStateをアーカイブする。
func (s *StateService) Archive(ctx context.Context, id string, input ArchiveInput) (*domain.State, error) {
	state, err := s.stateRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if state.Status == domain.StatusArchived {
		return nil, domain.ErrArchived
	}

	now := time.Now()
	state.Archive(input.Resolution, now)

	if err := s.stateRepo.Update(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to archive state: %w", err)
	}

	// ベクトルインデックスから削除（アーカイブされたStateは通常検索対象外）
	if s.vectorRepo != nil {
		_ = s.vectorRepo.Delete(ctx, state.ID)
	}

	return state, nil
}

// List はプロジェクト内のStateを一覧取得する。
func (s *StateService) List(ctx context.Context, projectID string, opts *repository.StateListOptions) ([]*domain.State, error) {
	return s.stateRepo.List(ctx, projectID, opts)
}

// ListSummary はプロジェクト内のStateをサマリビューで一覧取得する。
func (s *StateService) ListSummary(ctx context.Context, projectID string, opts *repository.StateListOptions) ([]domain.StateSummary, error) {
	states, err := s.stateRepo.List(ctx, projectID, opts)
	if err != nil {
		return nil, err
	}
	summaries := make([]domain.StateSummary, 0, len(states))
	for _, state := range states {
		summaries = append(summaries, state.ToSummary())
	}
	return summaries, nil
}

// Search はセマンティック検索でStateを検索する。
func (s *StateService) Search(ctx context.Context, query string, limit int, projectID string) ([]*domain.State, error) {
	if s.vectorRepo != nil {
		filters := map[string]string{
			"type":       "state",
			"project_id": projectID,
		}

		results, err := s.vectorRepo.Search(ctx, query, limit, filters)
		if err == nil {
			var states []*domain.State
			for _, result := range results {
				state, err := s.stateRepo.Get(ctx, result.ID)
				if err != nil {
					continue
				}
				if state.Status == domain.StatusArchived {
					continue
				}
				states = append(states, state)
			}
			return states, nil
		}
		slog.Warn("vector state search failed, fallback to keyword search", "error", err)
	}

	return s.fallbackSearch(ctx, query, limit, projectID)
}

// SearchSummary はセマンティック検索でStateをサマリビューで検索する。
func (s *StateService) SearchSummary(ctx context.Context, query string, limit int, projectID string) ([]domain.StateSummary, error) {
	states, err := s.Search(ctx, query, limit, projectID)
	if err != nil {
		return nil, err
	}
	summaries := make([]domain.StateSummary, 0, len(states))
	for _, state := range states {
		summaries = append(summaries, state.ToSummary())
	}
	return summaries, nil
}

func (s *StateService) fallbackSearch(ctx context.Context, query string, limit int, projectID string) ([]*domain.State, error) {
	states, err := s.stateRepo.List(ctx, projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list states for fallback search: %w", err)
	}

	matched := make([]*domain.State, 0, len(states))
	for _, state := range states {
		if state.Status == domain.StatusArchived {
			continue
		}
		if matchesQuery(query, state.Title, state.Description, joinTags(state.Tags)) {
			matched = append(matched, state)
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority < matched[j].Priority
		}
		return matched[i].UpdatedAt.After(matched[j].UpdatedAt)
	})

	if limit <= 0 {
		limit = 10
	}
	if len(matched) > limit {
		matched = matched[:limit]
	}

	return matched, nil
}

func isValidStateType(t domain.StateType) bool {
	switch t {
	case domain.StateTypeTask, domain.StateTypeIssue, domain.StateTypeIncident, domain.StateTypeChange:
		return true
	default:
		return false
	}
}

var stateIDCounter int

func generateStateID(stateType domain.StateType) string {
	stateIDCounter++
	prefix := fmt.Sprintf("%s", stateType)
	return fmt.Sprintf("STA-%s-%03d", prefix, stateIDCounter)
}
