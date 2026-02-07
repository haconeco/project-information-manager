package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	Version        = "0.1.0"
	DefaultDataDir = "data"
)

// Config はアプリケーション全体の設定を保持する。
type Config struct {
	Version string `yaml:"-"`
	DataDir string `yaml:"data_dir"`

	// LLM設定
	LLM LLMConfig `yaml:"llm"`

	// MCPサーバー設定
	MCP MCPConfig `yaml:"mcp"`

	// RAG設定
	RAG RAGConfig `yaml:"rag"`
}

// LLMConfig はLLMプロバイダーの設定を保持する。
type LLMConfig struct {
	Provider string `yaml:"provider"` // "anthropic" | "openai"
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
}

// MCPConfig はMCPサーバーの設定を保持する。
type MCPConfig struct {
	Transport string `yaml:"transport"` // "stdio" | "sse"
	Name      string `yaml:"name"`
}

// RAGConfig はRAG機能の設定を保持する。
type RAGConfig struct {
	Enabled    bool               `yaml:"enabled"`
	Collection string             `yaml:"collection"`
	Embedding  RAGEmbeddingConfig `yaml:"embedding"`
}

// RAGEmbeddingConfig は埋め込み生成の設定を保持する。
type RAGEmbeddingConfig struct {
	Provider      string `yaml:"provider"`        // "openai" | "ollama"
	Model         string `yaml:"model"`           // 例: text-embedding-3-small
	APIKey        string `yaml:"api_key"`         // openai用
	OllamaBaseURL string `yaml:"ollama_base_url"` // ollama用
}

// Load は設定ファイルを読み込む。ファイルが存在しない場合はデフォルト値を使用する。
func Load() (*Config, error) {
	cfg := &Config{
		Version: Version,
		DataDir: DefaultDataDir,
		LLM: LLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
		},
		MCP: MCPConfig{
			Transport: "stdio",
			Name:      "project-information-manager",
		},
		RAG: RAGConfig{
			Enabled:    true,
			Collection: "pim-context",
			Embedding: RAGEmbeddingConfig{
				Provider:      "openai",
				Model:         "text-embedding-3-small",
				OllamaBaseURL: "http://localhost:11434/api",
			},
		},
	}

	// 設定ファイルのパスを決定
	configPaths := []string{
		"pim.yaml",
		"pim.yml",
		filepath.Join("configs", "default.yaml"),
	}

	for _, path := range configPaths {
		if data, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
			}
			break
		}
	}

	// 環境変数によるオーバーライド
	if v := os.Getenv("PIM_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("PIM_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("PIM_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("PIM_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("PIM_RAG_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.RAG.Enabled = enabled
		}
	}
	if v := os.Getenv("PIM_RAG_COLLECTION"); v != "" {
		cfg.RAG.Collection = v
	}
	if v := os.Getenv("PIM_RAG_EMBEDDING_PROVIDER"); v != "" {
		cfg.RAG.Embedding.Provider = v
	}
	if v := os.Getenv("PIM_RAG_EMBEDDING_MODEL"); v != "" {
		cfg.RAG.Embedding.Model = v
	}
	if v := os.Getenv("PIM_RAG_EMBEDDING_API_KEY"); v != "" {
		cfg.RAG.Embedding.APIKey = v
	}
	if v := os.Getenv("PIM_RAG_EMBEDDING_OLLAMA_BASE_URL"); v != "" {
		cfg.RAG.Embedding.OllamaBaseURL = v
	}

	return cfg, nil
}

// StocksDir はStockファイルの格納ディレクトリパスを返す。
func (c *Config) StocksDir() string {
	return filepath.Join(c.DataDir, "stocks")
}

// SkillsDir は生成されたSkillファイルの格納ディレクトリパスを返す。
func (c *Config) SkillsDir() string {
	return filepath.Join(c.DataDir, "skills")
}

// StatesDBPath はSQLiteデータベースのファイルパスを返す。
func (c *Config) StatesDBPath() string {
	return filepath.Join(c.DataDir, "states.db")
}

// VectorsDir はベクトルインデックスの格納ディレクトリパスを返す。
func (c *Config) VectorsDir() string {
	return filepath.Join(c.DataDir, "vectors")
}
