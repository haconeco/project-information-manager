package repository

import (
	"context"
	"strings"
	"testing"

	chromem "github.com/philippgille/chromem-go"
)

func TestChromemVectorRepositoryCRUDAndSearch(t *testing.T) {
	ctx := context.Background()
	repo, err := newChromemVectorRepository(t.TempDir(), "test-collection", testEmbeddingFunc())
	if err != nil {
		t.Fatalf("new repo error: %v", err)
	}

	if err := repo.Upsert(ctx, "doc-1", "API design guide", map[string]string{
		"type":       "stock",
		"project_id": "proj-1",
	}); err != nil {
		t.Fatalf("upsert doc-1: %v", err)
	}
	if err := repo.Upsert(ctx, "doc-2", "task progress", map[string]string{
		"type":       "state",
		"project_id": "proj-2",
	}); err != nil {
		t.Fatalf("upsert doc-2: %v", err)
	}

	exists, err := repo.Exists(ctx, "doc-1")
	if err != nil {
		t.Fatalf("exists error: %v", err)
	}
	if !exists {
		t.Fatalf("expected doc-1 to exist")
	}

	results, err := repo.Search(ctx, "api", 10, map[string]string{"project_id": "proj-1"})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(results))
	}
	if results[0].ID != "doc-1" {
		t.Fatalf("expected doc-1, got %s", results[0].ID)
	}

	if err := repo.Delete(ctx, "doc-1"); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	exists, err = repo.Exists(ctx, "doc-1")
	if err != nil {
		t.Fatalf("exists after delete error: %v", err)
	}
	if exists {
		t.Fatalf("expected doc-1 to be deleted")
	}
}

func TestChromemVectorRepositoryPersistence(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	repo1, err := newChromemVectorRepository(dir, "persist-collection", testEmbeddingFunc())
	if err != nil {
		t.Fatalf("new repo1 error: %v", err)
	}
	if err := repo1.Upsert(ctx, "doc-1", "API strategy", map[string]string{
		"type":       "stock",
		"project_id": "proj-1",
	}); err != nil {
		t.Fatalf("upsert repo1: %v", err)
	}

	repo2, err := newChromemVectorRepository(dir, "persist-collection", testEmbeddingFunc())
	if err != nil {
		t.Fatalf("new repo2 error: %v", err)
	}

	exists, err := repo2.Exists(ctx, "doc-1")
	if err != nil {
		t.Fatalf("exists repo2 error: %v", err)
	}
	if !exists {
		t.Fatalf("expected persisted doc-1 to exist")
	}

	results, err := repo2.Search(ctx, "api", 5, map[string]string{"project_id": "proj-1"})
	if err != nil {
		t.Fatalf("search repo2 error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "doc-1" {
		t.Fatalf("unexpected persisted search result: %+v", results)
	}
}

func TestNormalizedCollectionName(t *testing.T) {
	got := normalizedCollectionName("PIM Context/OpenAI:text-embedding-3-small")
	if got != "pim-context-openai-text-embedding-3-small" {
		t.Fatalf("unexpected normalized name: %s", got)
	}
}

func testEmbeddingFunc() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		lower := strings.ToLower(text)
		vec := []float32{0.1, 0.1, 0.1}
		if strings.Contains(lower, "api") {
			vec[0] = 1.0
		}
		if strings.Contains(lower, "task") {
			vec[1] = 1.0
		}
		if strings.Contains(lower, "design") {
			vec[2] = 1.0
		}
		return vec, nil
	}
}
