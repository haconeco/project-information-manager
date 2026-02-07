package mcp

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haconeco/project-information-manager/internal/config"
	"github.com/haconeco/project-information-manager/internal/domain"
	"github.com/haconeco/project-information-manager/internal/repository"
	"github.com/haconeco/project-information-manager/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
)

func newTestServer(t *testing.T) (*Server, repository.StockRepository, repository.StateRepository) {
	t.Helper()

	stockRepo := repository.NewFileStockRepository(t.TempDir())

	dbPath := filepath.Join(t.TempDir(), "states.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	stateRepo, err := repository.NewSQLiteStateRepository(db)
	if err != nil {
		_ = db.Close()
		t.Fatalf("failed to create state repo: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	repos := &repository.Repositories{Stock: stockRepo, State: stateRepo}
	services := service.NewServices(repos)
	cfg := &config.Config{Version: "test", MCP: config.MCPConfig{Name: "pim", Transport: "stdio"}}

	srv, err := NewServer(services, cfg)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	return srv, stockRepo, stateRepo
}

func newRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

func getText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatalf("expected content, got nil")
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("expected text content")
	}
	return text.Text
}

func TestServerRunUnsupportedTransport(t *testing.T) {
	srv, _, _ := newTestServer(t)
	srv.cfg.MCP.Transport = "invalid"

	if err := srv.Run(context.Background()); err == nil {
		t.Fatalf("expected error for unsupported transport")
	}
}

func TestStockManageHandlers(t *testing.T) {
	srv, stockRepo, _ := newTestServer(t)
	ctx := context.Background()

	result, _ := srv.handleStockManage(ctx, newRequest(map[string]any{"action": "unknown"}))
	if !result.IsError {
		t.Fatalf("expected error for unknown action")
	}

	result, _ = srv.handleStockManage(ctx, newRequest(map[string]any{
		"action":     "create",
		"project_id": "proj-1",
		"category":   "design",
		"priority":   "P1",
		"title":      "Design",
		"content":    "content",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on create: %s", getText(t, result))
	}

	stocks, err := stockRepo.List(ctx, "proj-1", nil)
	if err != nil || len(stocks) != 1 {
		t.Fatalf("expected 1 stock, got %d, err=%v", len(stocks), err)
	}
	stockID := stocks[0].ID

	result, _ = srv.handleStockManage(ctx, newRequest(map[string]any{
		"action":   "read",
		"stock_id": stockID,
	}))
	if result.IsError {
		t.Fatalf("unexpected error on read: %s", getText(t, result))
	}

	result, _ = srv.handleStockManage(ctx, newRequest(map[string]any{
		"action":     "list",
		"project_id": "proj-1",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on list: %s", getText(t, result))
	}

	result, _ = srv.handleStockManage(ctx, newRequest(map[string]any{
		"action":   "update",
		"stock_id": stockID,
		"priority": "P0",
		"content":  "updated",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on update: %s", getText(t, result))
	}

	result, _ = srv.handleStockManage(ctx, newRequest(map[string]any{
		"action":     "list",
		"project_id": "proj-1",
		"priority":   "INVALID",
	}))
	if !result.IsError {
		t.Fatalf("expected error for invalid priority")
	}
}

func TestStateManageHandlers(t *testing.T) {
	srv, _, stateRepo := newTestServer(t)
	ctx := context.Background()

	result, _ := srv.handleStateManage(ctx, newRequest(map[string]any{"action": "unknown"}))
	if !result.IsError {
		t.Fatalf("expected error for unknown action")
	}

	result, _ = srv.handleStateManage(ctx, newRequest(map[string]any{
		"action":      "create",
		"project_id":  "proj-1",
		"type":        "task",
		"priority":    "P2",
		"title":       "Task",
		"description": "desc",
		"tags":        []string{"t1"},
	}))
	if result.IsError {
		t.Fatalf("unexpected error on create: %s", getText(t, result))
	}

	states, err := stateRepo.List(ctx, "proj-1", nil)
	if err != nil || len(states) != 1 {
		t.Fatalf("expected 1 state, got %d, err=%v", len(states), err)
	}
	stateID := states[0].ID

	result, _ = srv.handleStateManage(ctx, newRequest(map[string]any{
		"action":   "read",
		"state_id": stateID,
	}))
	if result.IsError {
		t.Fatalf("unexpected error on read: %s", getText(t, result))
	}

	result, _ = srv.handleStateManage(ctx, newRequest(map[string]any{
		"action":     "update",
		"state_id":   stateID,
		"status":     "in_progress",
		"resolution": "done",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on update: %s", getText(t, result))
	}

	result, _ = srv.handleStateManage(ctx, newRequest(map[string]any{
		"action":     "archive",
		"state_id":   stateID,
		"resolution": "done",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on archive: %s", getText(t, result))
	}

	result, _ = srv.handleStateManage(ctx, newRequest(map[string]any{
		"action":     "list",
		"project_id": "proj-1",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on list: %s", getText(t, result))
	}

	result, _ = srv.handleStateManage(ctx, newRequest(map[string]any{
		"action":     "search",
		"project_id": "proj-1",
		"query":      "task",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on search: %s", getText(t, result))
	}
}

func TestContextSearchHandlerValidation(t *testing.T) {
	srv, _, _ := newTestServer(t)
	ctx := context.Background()

	result, _ := srv.handleContextSearch(ctx, newRequest(map[string]any{"project_id": "proj-1"}))
	if !result.IsError || !strings.Contains(getText(t, result), "query") {
		t.Fatalf("expected query validation error")
	}

	result, _ = srv.handleContextSearch(ctx, newRequest(map[string]any{"query": "x"}))
	if !result.IsError || !strings.Contains(getText(t, result), "project_id") {
		t.Fatalf("expected project_id validation error")
	}
}

func TestSearchHandlersFallbackWithoutVector(t *testing.T) {
	srv, stockRepo, _ := newTestServer(t)
	ctx := context.Background()

	stock := &domain.Stock{
		ID:        "STK-DESIGN-001",
		ProjectID: "proj-1",
		Category:  domain.CategoryDesign,
		Priority:  domain.PriorityP0,
		Title:     "API Design",
		Content:   "Design content",
	}
	if err := stockRepo.Create(ctx, stock); err != nil {
		t.Fatalf("create stock: %v", err)
	}

	result, _ := srv.handleStockManage(ctx, newRequest(map[string]any{
		"action":     "search",
		"project_id": "proj-1",
		"query":      "api",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on stock fallback search: %s", getText(t, result))
	}
	if !strings.Contains(getText(t, result), "STK-DESIGN-001") {
		t.Fatalf("expected fallback stock search result, got: %s", getText(t, result))
	}

	result, _ = srv.handleContextSearch(ctx, newRequest(map[string]any{
		"project_id": "proj-1",
		"query":      "api",
	}))
	if result.IsError {
		t.Fatalf("unexpected error on context fallback search: %s", getText(t, result))
	}
	if !strings.Contains(getText(t, result), "STK-DESIGN-001") {
		t.Fatalf("expected context fallback result, got: %s", getText(t, result))
	}
}

func TestStateListFilters(t *testing.T) {
	srv, _, _ := newTestServer(t)
	ctx := context.Background()

	_, _ = srv.handleStateManage(ctx, newRequest(map[string]any{
		"action":      "create",
		"project_id":  "proj-1",
		"type":        "issue",
		"priority":    "P1",
		"title":       "Issue",
		"description": "desc",
	}))

	result, _ := srv.handleStateManage(ctx, newRequest(map[string]any{
		"action":     "list",
		"project_id": "proj-1",
		"type":       string(domain.StateTypeIssue),
		"status":     string(domain.StatusOpen),
	}))
	if result.IsError {
		t.Fatalf("unexpected error on filtered list: %s", getText(t, result))
	}
}
