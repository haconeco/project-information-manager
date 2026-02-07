package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/haconeco/project-information-manager/internal/domain"
)

func newTestSQLiteRepo(t *testing.T) *SQLiteStateRepository {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "states.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	repo, err := NewSQLiteStateRepository(db)
	if err != nil {
		_ = db.Close()
		t.Fatalf("failed to create repo: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return repo
}

func TestSQLiteStateRepositoryCRUDAndList(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)

	now := time.Now()
	state1 := &domain.State{
		ID:          "STA-TASK-001",
		ProjectID:   "proj-1",
		Type:        domain.StateTypeTask,
		Status:      domain.StatusOpen,
		Priority:    domain.PriorityP1,
		Title:       "Task 1",
		Description: "desc",
		Tags:        []string{"tag-a", "tag-b"},
		References:  []string{"STK-001"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state2 := &domain.State{
		ID:          "STA-ISSUE-002",
		ProjectID:   "proj-1",
		Type:        domain.StateTypeIssue,
		Status:      domain.StatusOpen,
		Priority:    domain.PriorityP0,
		Title:       "Issue",
		Description: "issue",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := repo.Create(ctx, state1); err != nil {
		t.Fatalf("create state1: %v", err)
	}
	if err := repo.Create(ctx, state2); err != nil {
		t.Fatalf("create state2: %v", err)
	}

	got, err := repo.Get(ctx, state1.ID)
	if err != nil {
		t.Fatalf("get state1: %v", err)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "tag-a" {
		t.Fatalf("unexpected tags: %v", got.Tags)
	}
	if len(got.References) != 1 || got.References[0] != "STK-001" {
		t.Fatalf("unexpected references: %v", got.References)
	}

	archivedAt := now.Add(time.Hour)
	state1.Status = domain.StatusArchived
	state1.Resolution = "done"
	state1.ArchivedAt = &archivedAt
	state1.UpdatedAt = archivedAt
	if err := repo.Update(ctx, state1); err != nil {
		t.Fatalf("update state1: %v", err)
	}

	listActive, err := repo.List(ctx, "proj-1", &StateListOptions{IncludeArchived: false})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(listActive) != 1 {
		t.Fatalf("expected 1 active state, got %d", len(listActive))
	}

	listAll, err := repo.List(ctx, "proj-1", &StateListOptions{IncludeArchived: true})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(listAll) != 2 {
		t.Fatalf("expected 2 states, got %d", len(listAll))
	}

	listType, err := repo.List(ctx, "proj-1", &StateListOptions{Type: ptrStateType(domain.StateTypeIssue), IncludeArchived: true})
	if err != nil {
		t.Fatalf("list type: %v", err)
	}
	if len(listType) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(listType))
	}

	missing := &domain.State{ID: "STA-MISSING-999"}
	if err := repo.Update(ctx, missing); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound on update missing, got %v", err)
	}
	if _, err := repo.Get(ctx, "missing"); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound on get missing, got %v", err)
	}
}

func TestParseJSONStringArray(t *testing.T) {
	if got := parseJSONStringArray(""); got != nil {
		t.Fatalf("expected nil for empty string, got %v", got)
	}
	if got := parseJSONStringArray("[]"); got != nil {
		t.Fatalf("expected nil for empty array, got %v", got)
	}
	got := parseJSONStringArray("[\"a\",\"b\"]")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected parse result: %v", got)
	}
}

func ptrStateType(v domain.StateType) *domain.StateType {
	return &v
}
