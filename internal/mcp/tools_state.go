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

// registerStateTools はState管理のファサードツールを登録する（1ツールに統合）。
func (s *Server) registerStateTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("state_manage",
			mcp.WithDescription("プロダクト開発の動的な状態情報（タスク、課題、インシデント等）を管理するState操作ツール。actionで操作を指定。list/searchはサマリ（タイトル・ステータス等のみ）を返却、readで全文取得。"),
			mcp.WithString("action", mcp.Required(), mcp.Description("操作種別: create, read, update, archive, list, search")),
			mcp.WithString("project_id", mcp.Description("プロジェクトID（create/list/searchで必須）")),
			mcp.WithString("state_id", mcp.Description("State管理番号（read/update/archiveで必須）")),
			mcp.WithString("type", mcp.Description("種別: task, issue, incident, change（createで必須、listでフィルタ）")),
			mcp.WithString("priority", mcp.Description("優先度: P0, P1, P2, P3（createで必須）")),
			mcp.WithString("title", mcp.Description("タイトル（createで必須）")),
			mcp.WithString("description", mcp.Description("詳細説明（createで必須、updateでオプション）")),
			mcp.WithString("status", mcp.Description("ステータス: open, in_progress, resolved（updateでオプション、listでフィルタ）")),
			mcp.WithString("resolution", mcp.Description("解決内容（update/archiveでオプション）")),
			mcp.WithString("query", mcp.Description("検索クエリ（searchで必須）")),
			mcp.WithNumber("limit", mcp.Description("検索結果の上限数（search用、デフォルト: 10）")),
			mcp.WithBoolean("include_archived", mcp.Description("アーカイブ済みを含むか（list用、デフォルト: false）")),
		),
		s.handleStateManage,
	)
}

func (s *Server) handleStateManage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := request.GetString("action", "")

	switch action {
	case "create":
		return s.handleStateCreate(ctx, request)
	case "read":
		return s.handleStateRead(ctx, request)
	case "update":
		return s.handleStateUpdate(ctx, request)
	case "archive":
		return s.handleStateArchive(ctx, request)
	case "list":
		return s.handleStateList(ctx, request)
	case "search":
		return s.handleStateSearch(ctx, request)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不明なaction: %s（有効値: create, read, update, archive, list, search）", action)), nil
	}
}

func (s *Server) handleStateCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tags := request.GetStringSlice("tags", nil)

	input := service.CreateStateInput{
		ProjectID:   request.GetString("project_id", ""),
		Type:        request.GetString("type", ""),
		Priority:    request.GetString("priority", "P3"),
		Title:       request.GetString("title", ""),
		Description: request.GetString("description", ""),
		Tags:        tags,
	}

	state, err := s.services.State.Create(ctx, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("State作成エラー: %v", err)), nil
	}

	// 作成結果はサマリビューで返却
	summary := state.ToSummary()
	data, _ := json.MarshalIndent(summary, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Stateを作成しました:\n%s", string(data))), nil
}

func (s *Server) handleStateRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stateID := request.GetString("state_id", "")
	if stateID == "" {
		return mcp.NewToolResultError("state_id は必須です"), nil
	}

	state, err := s.services.State.Get(ctx, stateID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("State取得エラー: %v", err)), nil
	}

	// readはフルビューで返却
	data, _ := json.MarshalIndent(state, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleStateUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stateID := request.GetString("state_id", "")
	if stateID == "" {
		return mcp.NewToolResultError("state_id は必須です"), nil
	}

	input := service.UpdateStateInput{}
	if v := request.GetString("status", ""); v != "" {
		input.Status = &v
	}
	if v := request.GetString("description", ""); v != "" {
		input.Description = &v
	}
	if v := request.GetString("resolution", ""); v != "" {
		input.Resolution = &v
	}

	state, err := s.services.State.Update(ctx, stateID, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("State更新エラー: %v", err)), nil
	}

	// 更新結果はサマリビューで返却
	summary := state.ToSummary()
	data, _ := json.MarshalIndent(summary, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Stateを更新しました:\n%s", string(data))), nil
}

func (s *Server) handleStateArchive(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stateID := request.GetString("state_id", "")
	if stateID == "" {
		return mcp.NewToolResultError("state_id は必須です"), nil
	}

	input := service.ArchiveInput{
		Resolution:   request.GetString("resolution", ""),
		StockSummary: request.GetString("stock_summary", ""),
	}

	state, err := s.services.State.Archive(ctx, stateID, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Stateアーカイブエラー: %v", err)), nil
	}

	// アーカイブ結果はサマリビューで返却
	summary := state.ToSummary()
	data, _ := json.MarshalIndent(summary, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Stateをアーカイブしました:\n%s", string(data))), nil
}

func (s *Server) handleStateList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := request.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id は必須です"), nil
	}

	opts := &repository.StateListOptions{}

	if v := request.GetString("type", ""); v != "" {
		t := domain.StateType(v)
		opts.Type = &t
	}
	if v := request.GetString("status", ""); v != "" {
		st := domain.StateStatus(v)
		opts.Status = &st
	}
	opts.IncludeArchived = request.GetBool("include_archived", false)

	// サマリビューで返却（Description を含まない）
	summaries, err := s.services.State.ListSummary(ctx, projectID, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("State一覧取得エラー: %v", err)), nil
	}

	data, _ := json.MarshalIndent(summaries, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleStateSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	summaries, err := s.services.State.SearchSummary(ctx, query, limit, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("State検索エラー: %v", err)), nil
	}

	data, _ := json.MarshalIndent(summaries, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
