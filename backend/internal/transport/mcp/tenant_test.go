package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

func TestBindMCPTenant_ForcesClientIDAndBlocksKeyAdmin(t *testing.T) {
	ctx := middleware.WithIdentity(context.Background(), &middleware.AuthIdentity{
		ClientID: "alice", UserKey: true, CanTrade: true, KeyID: "k1",
	})

	req := mcp.CallToolRequest{}
	req.Params.Name = "get_portfolio"
	req.Params.Arguments = map[string]any{"clientId": "bob"}
	err := bindMCPTenant(ctx, &req)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("mismatch: %v", err)
	}

	req.Params.Arguments = map[string]any{"clientId": "alice", "x": 1}
	if err := bindMCPTenant(ctx, &req); err != nil {
		t.Fatal(err)
	}
	if req.GetString("clientId", "") != "alice" {
		t.Fatalf("args=%v", req.Params.Arguments)
	}

	req.Params.Name = "create_api_key"
	req.Params.Arguments = map[string]any{"clientId": "alice"}
	if err := bindMCPTenant(ctx, &req); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("create_api_key should be blocked: %v", err)
	}

	// Master/open: no identity binding.
	req = mcp.CallToolRequest{}
	req.Params.Name = "create_api_key"
	req.Params.Arguments = map[string]any{"clientId": "anyone"}
	if err := bindMCPTenant(context.Background(), &req); err != nil {
		t.Fatal(err)
	}
}

func TestBindMCPTenant_RemoteMasterCannotPickTenant(t *testing.T) {
	ctx := middleware.WithIdentity(context.Background(), &middleware.AuthIdentity{
		Master: true, DenyImpersonate: true, Loopback: false,
	})
	req := mcp.CallToolRequest{}
	req.Params.Name = "get_portfolio"
	req.Params.Arguments = map[string]any{"clientId": "victim"}
	if err := bindMCPTenant(ctx, &req); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("remote master + clientId: %v", err)
	}

	loop := middleware.WithIdentity(context.Background(), &middleware.AuthIdentity{
		Master: true, DenyImpersonate: true, Loopback: true,
	})
	req.Params.Arguments = map[string]any{"clientId": "local"}
	if err := bindMCPTenant(loop, &req); err != nil {
		t.Fatalf("loopback master: %v", err)
	}
}
