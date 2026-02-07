package service

import (
	"context"
	"testing"
	"time"

	"github.com/haconeco/project-information-manager/internal/domain"
	"github.com/haconeco/project-information-manager/internal/repository"
)

type fakeStateRepo struct {
	states map[string]*domain.State
}

func newFakeStateRepo() *fakeStateRepo {
	return &fakeStateRepo{states: make(map[string]*domain.State)}
}

func (f *fakeStateRepo) Create(ctx context.Context, state *domain.State) error {
	if _, ok := f.states[state.ID]; ok {
		return domain.ErrAlreadyExists
	}
	f.states[state.ID] = cloneState(state)
	return nil
}

func (f *fakeStateRepo) Get(ctx context.Context, id string) (*domain.State, error) {
	state, ok := f.states[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneState(state), nil
}

func (f *fakeStateRepo) Update(ctx context.Context, state *domain.State) error {
	if _, ok := f.states[state.ID]; !ok {
		return domain.ErrNotFound
	}
	f.states[state.ID] = cloneState(state)
	return nil
}

func (f *fakeStateRepo) List(ctx context.Context, projectID string, opts *repository.StateListOptions) ([]*domain.State, error) {
	var out []*domain.State
	for _, state := range f.states {
		if state.ProjectID != projectID {
			continue
		}
		if opts != nil {
			if opts.Type != nil && state.Type != *opts.Type {
				continue
			}
			if opts.Status != nil && state.Status != *opts.Status {
				continue
			}
			if opts.Priority != nil && state.Priority != *opts.Priority {
				continue
			}
			if !opts.IncludeArchived && state.Status == domain.StatusArchived {
				continue
			}
		} else if state.Status == domain.StatusArchived {
			continue
		}
		out = append(out, cloneState(state))
	}
	return out, nil
}

type fakeVectorRepo struct {
	results []repository.SearchResult
}

func (f *fakeVectorRepo) Upsert(ctx context.Context, id string, content string, metadata map[string]string) error {
	return nil
}

func (f *fakeVectorRepo) Search(ctx context.Context, query string, limit int, filters map[string]string) ([]repository.SearchResult, error) {
	return f.results, nil
}

func (f *fakeVectorRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func cloneState(s *domain.State) *domain.State {
	if s == nil {
		return nil
	}
	copy := *s
	if s.Tags != nil {
		copy.Tags = append([]string(nil), s.Tags...)
	}
	if s.References != nil {
		copy.References = append([]string(nil), s.References...)
	}
	return &copy
}

func TestStateServiceCreateUpdateArchive(t *testing.T) {
	repo := newFakeStateRepo()
	svc := NewStateService(repo, nil)

	_, err := svc.Create(context.Background(), CreateStateInput{Type: "invalid", Priority: "P1"})
	if err != domain.ErrInvalidType {
		t.Fatalf("expected ErrInvalidType, got %v", err)
	}

	created, err := svc.Create(context.Background(), CreateStateInput{
		ProjectID:   "proj-1",
		Type:        "task",
		Priority:    "P2",
		Title:       "Task",
		Description: "desc",
		Tags:        []string{"t1"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != domain.StatusOpen {
		t.Fatalf("expected status open, got %s", created.Status)
	}

	newStatus := "in_progress"
	newDesc := "updated"
	newResolution := "resolved"
	newPriority := "P0"
	updated, err := svc.Update(context.Background(), created.ID, UpdateStateInput{
		Status:      &newStatus,
		Description: &newDesc,
		Resolution:  &newResolution,
		Priority:    &newPriority,
		Tags:        []string{"t2"},
		References:  []string{"STK-001"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Status != domain.StatusInProgress {
		t.Fatalf("expected in_progress, got %s", updated.Status)
	}
	if updated.Priority != domain.PriorityP0 {
		t.Fatalf("expected priority P0, got %s", updated.Priority.String())
	}

	archived, err := svc.Archive(context.Background(), created.ID, ArchiveInput{Resolution: "done"})
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if archived.Status != domain.StatusArchived || archived.ArchivedAt == nil {
		t.Fatalf("expected archived state, got %v", archived.Status)
	}
}

func TestStateServiceUpdateArchived(t *testing.T) {
	repo := newFakeStateRepo()
	archivedAt := time.Now().Add(-time.Hour)
	repo.states["STA-TASK-999"] = &domain.State{
		ID:         "STA-TASK-999",
		ProjectID:  "proj-1",
		Type:       domain.StateTypeTask,
		Status:     domain.StatusArchived,
		Priority:   domain.PriorityP1,
		Title:      "archived",
		CreatedAt:  time.Now().Add(-time.Hour * 2),
		UpdatedAt:  archivedAt,
		ArchivedAt: &archivedAt,
	}

	svc := NewStateService(repo, nil)
	_, err := svc.Update(context.Background(), "STA-TASK-999", UpdateStateInput{Description: ptrString("x")})
	if err != domain.ErrArchived {
		t.Fatalf("expected ErrArchived, got %v", err)
	}
}

func TestStateServiceSearchSummaryWithVector(t *testing.T) {
	repo := newFakeStateRepo()
	repo.states["STA-TASK-001"] = &domain.State{
		ID:         "STA-TASK-001",
		ProjectID:  "proj-1",
		Type:       domain.StateTypeTask,
		Status:     domain.StatusOpen,
		Priority:   domain.PriorityP1,
		Title:      "Task",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	vector := &fakeVectorRepo{results: []repository.SearchResult{{ID: "STA-TASK-001"}}}
	svc := NewStateService(repo, vector)

	results, err := svc.SearchSummary(context.Background(), "task", 10, "proj-1")
	if err != nil {
		t.Fatalf("search summary: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestStateServiceListSummary(t *testing.T) {
	repo := newFakeStateRepo()
	repo.states["STA-ISSUE-001"] = &domain.State{
		ID:         "STA-ISSUE-001",
		ProjectID:  "proj-1",
		Type:       domain.StateTypeIssue,
		Status:     domain.StatusOpen,
		Priority:   domain.PriorityP1,
		Title:      "Issue",
		Tags:       []string{"bug"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	svc := NewStateService(repo, nil)
	summaries, err := svc.ListSummary(context.Background(), "proj-1", nil)
	if err != nil {
		t.Fatalf("list summary: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].ID != "STA-ISSUE-001" {
		t.Fatalf("unexpected ID: %s", summaries[0].ID)
	}
}

func ptrString(v string) *string {
	return &v
}
