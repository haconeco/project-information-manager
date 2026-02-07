package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerContextTools はコンテキスト横断検索ツールを登録する。
func (s *Server) registerContextTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("context_search",
			mcp.WithDescription("Stock（静的情報）とState（動的状態）を横断してRAGセマンティック検索を行い、関連するコンテキスト情報のサマリを返却します。詳細が必要な場合は stock_manage action=read / state_manage action=read で個別に全文取得してください。"),
			mcp.WithString("query", mcp.Required(), mcp.Description("検索クエリ（自然言語で記述）")),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("プロジェクトID")),
			mcp.WithNumber("limit", mcp.Description("結果件数の上限（デフォルト: 10）")),
		),
		s.handleContextSearch,
	)
}

func (s *Server) handleContextSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query は必須です"), nil
	}
	projectID := request.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id は必須です"), nil
	}
	limit := request.GetInt("limit", 10)

	// Stock と State を横断検索してサマリで返却
	result, err := s.services.Context.Search(ctx, query, projectID, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("コンテキスト検索エラー: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
