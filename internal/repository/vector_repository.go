package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/haconeco/project-information-manager/internal/config"
	chromem "github.com/philippgille/chromem-go"
)

var collectionNameSanitizer = regexp.MustCompile(`[^a-z0-9_-]+`)

// ChromemVectorRepository は chromem-go を利用した VectorRepository 実装。
type ChromemVectorRepository struct {
	db         *chromem.DB
	collection *chromem.Collection
}

// NewChromemVectorRepository は設定に基づいてベクトルリポジトリを初期化する。
func NewChromemVectorRepository(cfg *config.Config) (*ChromemVectorRepository, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	embeddingFunc, err := buildEmbeddingFunc(cfg)
	if err != nil {
		return nil, err
	}

	collectionName := normalizedCollectionName(
		fmt.Sprintf("%s-%s-%s", cfg.RAG.Collection, cfg.RAG.Embedding.Provider, cfg.RAG.Embedding.Model),
	)
	return newChromemVectorRepository(cfg.VectorsDir(), collectionName, embeddingFunc)
}

func newChromemVectorRepository(
	vectorsDir string,
	collectionName string,
	embeddingFunc chromem.EmbeddingFunc,
) (*ChromemVectorRepository, error) {
	db, err := chromem.NewPersistentDB(vectorsDir, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create persistent vector DB: %w", err)
	}

	collection, err := db.GetOrCreateCollection(collectionName, nil, embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection %q: %w", collectionName, err)
	}

	return &ChromemVectorRepository{
		db:         db,
		collection: collection,
	}, nil
}

func buildEmbeddingFunc(cfg *config.Config) (chromem.EmbeddingFunc, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.RAG.Embedding.Provider))

	switch provider {
	case "openai":
		if strings.TrimSpace(cfg.RAG.Embedding.APIKey) == "" {
			return nil, errors.New("openai embedding requires api_key")
		}
		model := chromem.EmbeddingModelOpenAI(cfg.RAG.Embedding.Model)
		return chromem.NewEmbeddingFuncOpenAI(cfg.RAG.Embedding.APIKey, model), nil

	case "ollama":
		if strings.TrimSpace(cfg.RAG.Embedding.Model) == "" {
			return nil, errors.New("ollama embedding requires model")
		}
		return chromem.NewEmbeddingFuncOllama(cfg.RAG.Embedding.Model, cfg.RAG.Embedding.OllamaBaseURL), nil

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.RAG.Embedding.Provider)
	}
}

func normalizedCollectionName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = collectionNameSanitizer.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-_")
	if s == "" {
		return "pim-context"
	}
	return s
}

// Upsert は既存ドキュメントがあれば置き換え、なければ追加する。
func (r *ChromemVectorRepository) Upsert(ctx context.Context, id string, content string, metadata map[string]string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}

	exists, err := r.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check document existence: %w", err)
	}
	if exists {
		if err := r.collection.Delete(ctx, nil, nil, id); err != nil {
			return fmt.Errorf("failed to delete existing document %s: %w", id, err)
		}
	}

	doc := chromem.Document{
		ID:       id,
		Content:  content,
		Metadata: metadata,
	}
	if err := r.collection.AddDocument(ctx, doc); err != nil {
		return fmt.Errorf("failed to add document %s: %w", id, err)
	}
	return nil
}

// Search はフィルタ付きセマンティック検索を実行する。
func (r *ChromemVectorRepository) Search(ctx context.Context, query string, limit int, filters map[string]string) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query is required")
	}
	if limit <= 0 {
		limit = 10
	}
	count := r.collection.Count()
	if count == 0 {
		return nil, nil
	}
	if limit > count {
		limit = count
	}

	results, err := r.collection.Query(ctx, query, limit, filters, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query vector DB: %w", err)
	}

	searchResults := make([]SearchResult, 0, len(results))
	for _, res := range results {
		searchResults = append(searchResults, SearchResult{
			ID:         res.ID,
			Content:    res.Content,
			Metadata:   res.Metadata,
			Similarity: res.Similarity,
		})
	}
	return searchResults, nil
}

// Delete はベクトルインデックスからドキュメントを削除する。
func (r *ChromemVectorRepository) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}
	if err := r.collection.Delete(ctx, nil, nil, id); err != nil {
		return fmt.Errorf("failed to delete document %s: %w", id, err)
	}
	return nil
}

// Exists はドキュメントの存在有無を返す。
func (r *ChromemVectorRepository) Exists(ctx context.Context, id string) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, errors.New("id is required")
	}
	_, err := r.collection.GetByID(ctx, id)
	if err == nil {
		return true, nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return false, nil
	}
	return false, fmt.Errorf("failed to get document %s: %w", id, err)
}
