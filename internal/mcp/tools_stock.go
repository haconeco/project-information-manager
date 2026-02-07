package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/haconeco/project-information-manager/internal/domain"
	"github.com/haconeco/project-information-manager/internal/repository"
	"github.com/haconeco/project-information-manager/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerStockTools はStock管理のファサードツールを登録する（1ツールに統合）。
func (s *Server) registerStockTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("stock_manage",
			mcp.WithDescription("プロダクトの静的情報（設計、ルール、方針等）を管理するStock操作ツール。actionで操作を指定。list/searchはサマリ（タイトル・優先度等のみ）を返却、readで全文取得。"),
			mcp.WithString("action", mcp.Required(), mcp.Description("操作種別: create, read, list, update, search")),
			mcp.WithString("project_id", mcp.Description("プロジェクトID（create/list/searchで必須）")),
			mcp.WithString("stock_id", mcp.Description("Stock管理番号（read/updateで必須）")),
			mcp.WithString("category", mcp.Description("カテゴリ: design, rules, management, architecture, requirement, test（createで必須、listでフィルタ）")),
			mcp.WithString("priority", mcp.Description("優先度: P0, P1, P2, P3（createで必須、update/listでオプション）")),
			mcp.WithString("title", mcp.Description("タイトル（createで必須）")),
			mcp.WithString("content", mcp.Description("Markdown形式の本文（createで必須、updateでオプション）")),
			mcp.WithString("query", mcp.Description("検索クエリ（searchで必須）")),
			mcp.WithNumber("limit", mcp.Description("検索結果の上限数（search用、デフォルト: 10）")),
		),
		s.handleStockManage,
	)
}

func (s *Server) handleStockManage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := request.GetString("action", "")

	switch action {
	case "create":
		return s.handleStockCreate(ctx, request)
	case "read":
		return s.handleStockRead(ctx, request)
	case "list":
		return s.handleStockList(ctx, request)
	case "update":
		return s.handleStockUpdate(ctx, request)
	case "search":
		return s.handleStockSearch(ctx, request)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不明なaction: %s（有効値: create, read, list, update, search）", action)), nil
	}
}

func (s *Server) handleStockCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tags := request.GetStringSlice("tags", nil)

	input := service.CreateStockInput{
		ProjectID: request.GetString("project_id", ""),
		Category:  request.GetString("category", ""),
		Priority:  request.GetString("priority", "P3"),
		Title:     request.GetString("title", ""),
		Content:   request.GetString("content", ""),
		Tags:      tags,
	}

	stock, err := s.services.Stock.Create(ctx, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Stock作成エラー: %v", err)), nil
	}

	// 作成結果はサマリビューで返却
	summary := stock.ToSummary()
	data, _ := json.MarshalIndent(summary, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Stockを作成しました:\n%s", string(data))), nil
}

func (s *Server) handleStockRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stockID := request.GetString("stock_id", "")
	if stockID == "" {
		return mcp.NewToolResultError("stock_id は必須です"), nil
	}

	stock, err := s.services.Stock.Get(ctx, stockID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Stock取得エラー: %v", err)), nil
	}

	// readはフルビューで返却
	data, _ := json.MarshalIndent(stock, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleStockList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := request.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id は必須です"), nil
	}

	opts := &repository.StockListOptions{}

	if v := request.GetString("category", ""); v != "" {
		cat := domain.StockCategory(v)
		opts.Category = &cat
	}
	if v := request.GetString("priority", ""); v != "" {
		p, err := domain.ParsePriority(v)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("無効な優先度: %v", err)), nil
		}
		opts.Priority = &p
	}

	// サマリビューで返却（Content を含まない）
	summaries, err := s.services.Stock.ListSummary(ctx, projectID, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Stock一覧取得エラー: %v", err)), nil
	}

	data, _ := json.MarshalIndent(summaries, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleStockUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stockID := request.GetString("stock_id", "")
	if stockID == "" {
		return mcp.NewToolResultError("stock_id は必須です"), nil
	}

	input := service.UpdateStockInput{}

	if v := request.GetString("content", ""); v != "" {
		input.Content = &v
	}
	if v := request.GetString("priority", ""); v != "" {
		input.Priority = &v
	}

	stock, err := s.services.Stock.Update(ctx, stockID, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Stock更新エラー: %v", err)), nil
	}

	// 更新結果はサマリビューで返却
	summary := stock.ToSummary()
	data, _ := json.MarshalIndent(summary, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Stockを更新しました:\n%s", string(data))), nil
}

func (s *Server) handleStockSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query は必須です"), nil
	}
	projectID := request.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id は必須です"), nil
	}
	limit := request.GetInt("limit", 10)

	// サマリビューで返却
	summaries, err := s.services.Stock.SearchSummary(ctx, query, limit, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Stock検索エラー: %v", err)), nil
	}

	data, _ := json.MarshalIndent(summaries, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
