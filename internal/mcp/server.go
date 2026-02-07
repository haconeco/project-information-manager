package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/haconeco/project-information-manager/internal/config"
	"github.com/haconeco/project-information-manager/internal/service"
	gomcp "github.com/mark3labs/mcp-go/server"
)

// Server はMCPサーバーの実装。
type Server struct {
	mcpServer *gomcp.MCPServer
	services  *service.Services
	cfg       *config.Config
}

// NewServer は新しいMCPサーバーを生成する。
func NewServer(services *service.Services, cfg *config.Config) (*Server, error) {
	mcpServer := gomcp.NewMCPServer(
		cfg.MCP.Name,
		cfg.Version,
	)

	s := &Server{
		mcpServer: mcpServer,
		services:  services,
		cfg:       cfg,
	}

	// MCPツールを登録（ファサードパターン: 3ツールのみ）
	s.registerStockTools()
	s.registerStateTools()
	s.registerContextTools()

	return s, nil
}

// Run はMCPサーバーを起動する。
func (s *Server) Run(ctx context.Context) error {
	slog.Info("starting MCP server",
		"transport", s.cfg.MCP.Transport,
		"name", s.cfg.MCP.Name,
	)

	switch s.cfg.MCP.Transport {
	case "stdio":
		return s.runStdio(ctx)
	default:
		return fmt.Errorf("unsupported transport: %s", s.cfg.MCP.Transport)
	}
}

// runStdio はstdioトランスポートでMCPサーバーを実行する。
func (s *Server) runStdio(ctx context.Context) error {
	stdioServer := gomcp.NewStdioServer(s.mcpServer)
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}
