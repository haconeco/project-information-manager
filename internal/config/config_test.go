package config

import (
	"os"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != Version {
		t.Errorf("expected version %s, got %s", Version, cfg.Version)
	}

	if cfg.DataDir != DefaultDataDir {
		t.Errorf("expected data_dir %s, got %s", DefaultDataDir, cfg.DataDir)
	}

	if cfg.MCP.Transport != "stdio" {
		t.Errorf("expected transport stdio, got %s", cfg.MCP.Transport)
	}

	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("expected provider anthropic, got %s", cfg.LLM.Provider)
	}
	if !cfg.RAG.Enabled {
		t.Errorf("expected rag enabled true")
	}
	if cfg.RAG.Collection != "pim-context" {
		t.Errorf("expected rag collection pim-context, got %s", cfg.RAG.Collection)
	}
	if cfg.RAG.Embedding.Provider != "openai" {
		t.Errorf("expected rag embedding provider openai, got %s", cfg.RAG.Embedding.Provider)
	}
	if cfg.RAG.Embedding.Model != "text-embedding-3-small" {
		t.Errorf("expected rag embedding model text-embedding-3-small, got %s", cfg.RAG.Embedding.Model)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("PIM_DATA_DIR", "/tmp/pim-test")
	t.Setenv("PIM_LLM_API_KEY", "test-api-key")
	t.Setenv("PIM_LLM_PROVIDER", "openai")
	t.Setenv("PIM_RAG_ENABLED", "false")
	t.Setenv("PIM_RAG_COLLECTION", "proj-collection")
	t.Setenv("PIM_RAG_EMBEDDING_PROVIDER", "ollama")
	t.Setenv("PIM_RAG_EMBEDDING_MODEL", "nomic-embed-text")
	t.Setenv("PIM_RAG_EMBEDDING_API_KEY", "rag-key")
	t.Setenv("PIM_RAG_EMBEDDING_OLLAMA_BASE_URL", "http://localhost:11434/api")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DataDir != "/tmp/pim-test" {
		t.Errorf("expected data_dir /tmp/pim-test, got %s", cfg.DataDir)
	}

	if cfg.LLM.APIKey != "test-api-key" {
		t.Errorf("expected api_key test-api-key, got %s", cfg.LLM.APIKey)
	}

	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", cfg.LLM.Provider)
	}
	if cfg.RAG.Enabled {
		t.Errorf("expected rag enabled false")
	}
	if cfg.RAG.Collection != "proj-collection" {
		t.Errorf("expected rag collection proj-collection, got %s", cfg.RAG.Collection)
	}
	if cfg.RAG.Embedding.Provider != "ollama" {
		t.Errorf("expected rag provider ollama, got %s", cfg.RAG.Embedding.Provider)
	}
	if cfg.RAG.Embedding.Model != "nomic-embed-text" {
		t.Errorf("expected rag model nomic-embed-text, got %s", cfg.RAG.Embedding.Model)
	}
	if cfg.RAG.Embedding.APIKey != "rag-key" {
		t.Errorf("expected rag api key rag-key, got %s", cfg.RAG.Embedding.APIKey)
	}
	if cfg.RAG.Embedding.OllamaBaseURL != "http://localhost:11434/api" {
		t.Errorf("expected rag ollama base url http://localhost:11434/api, got %s", cfg.RAG.Embedding.OllamaBaseURL)
	}
}

func TestConfigPaths(t *testing.T) {
	cfg := &Config{DataDir: "data"}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"StocksDir", cfg.StocksDir(), "data/stocks"},
		{"StatesDBPath", cfg.StatesDBPath(), "data/states.db"},
		{"VectorsDir", cfg.VectorsDir(), "data/vectors"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// OS依存のパスセパレータを考慮
			if tt.got != tt.expected {
				// Windowsの場合のパスも許容
				expected2 := ""
				if os.PathSeparator == '\\' {
					expected2 = "data\\stocks"
				}
				if tt.got != expected2 && tt.got != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, tt.got)
				}
			}
		})
	}
}
