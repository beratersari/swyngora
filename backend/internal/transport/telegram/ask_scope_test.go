package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
)

func TestAsk_ForwardsFullToolAccessToAI(t *testing.T) {
	// /ask must not paper-trade or mint keys; those stay on /buy confirm and HTTP key admin.
	var got aiagent.ChatRequest
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"reply": "created a key", "sessionId": "s"})
	}))
	t.Cleanup(aiSrv.Close)

	r := newTestRouter(t)
	r.ai = aiagent.New(aiSrv.URL, 5*time.Second)
	r.opts.AITimeout = 5 * time.Second

	out := r.Handle(context.Background(), 42, 42, "/ask create an API key")
	if out == "" {
		t.Fatal("empty reply")
	}
	if got.CanTrade || got.CanManageKeys {
		t.Fatalf("telegram /ask must be read-only, got trade=%v keys=%v", got.CanTrade, got.CanManageKeys)
	}
	if got.Message != "create an API key" {
		t.Fatalf("message=%q", got.Message)
	}
	if got.ClientID == "" {
		t.Fatal("expected bound telegram clientId")
	}
}
