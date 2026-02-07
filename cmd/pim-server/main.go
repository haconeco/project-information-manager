package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/haconeco/project-information-manager/internal/config"
	"github.com/haconeco/project-information-manager/internal/mcp"
	"github.com/haconeco/project-information-manager/internal/repository"
	"github.com/haconeco/project-information-manager/internal/service"
)

func main() {
	// ロガー初期化
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// データディレクトリ初期化
	if err := ensureDataDirs(cfg); err != nil {
		slog.Error("failed to initialize data directories", "error", err)
		os.Exit(1)
	}

	// リポジトリ層初期化
	repos, err := repository.NewRepositories(cfg)
	if err != nil {
		slog.Error("failed to initialize repositories", "error", err)
		os.Exit(1)
	}
	defer repos.Close()

	// サービス層初期化
	services := service.NewServices(repos)
	if err := services.BootstrapVectorIndex(context.Background()); err != nil {
		slog.Warn("failed to bootstrap vector index; continuing without blocking startup", "error", err)
	}

	// MCPサーバー初期化・起動
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// シグナルハンドリング
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	server, err := mcp.NewServer(services, cfg)
	if err != nil {
		slog.Error("failed to create MCP server", "error", err)
		os.Exit(1)
	}

	slog.Info("starting PIM MCP server", "version", cfg.Version)
	if err := server.Run(ctx); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func ensureDataDirs(cfg *config.Config) error {
	dirs := []string{
		cfg.DataDir,
		cfg.StocksDir(),
		cfg.VectorsDir(),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
