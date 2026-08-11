package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
)

func TestToolClientActiveError_ClosedAccount(t *testing.T) {
	store := accountstore.NewMemory()
	svc := account.New(store, account.DataPurgeDeps{})
	ctx := context.Background()
	if _, err := svc.Close(ctx, "alice"); err != nil {
		t.Fatal(err)
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"clientId": "alice", "symbol": "BTCUSDT"}
	err := toolClientActiveError(ctx, svc, req)
	if err == nil {
		t.Fatal("expected closed account error")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Fatalf("err=%v", err)
	}

	// Market tools without clientId stay open.
	req2 := mcp.CallToolRequest{}
	req2.Params.Arguments = map[string]any{"symbol": "BTCUSDT"}
	if err := toolClientActiveError(ctx, svc, req2); err != nil {
		t.Fatalf("no clientId: %v", err)
	}

	if err := toolClientActiveError(ctx, nil, req); err != nil {
		t.Fatalf("nil accounts: %v", err)
	}
}
