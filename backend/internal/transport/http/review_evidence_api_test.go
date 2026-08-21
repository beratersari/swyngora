package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/alertstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/pricediffstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/scannerstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// These tests hit the HTTP API (httptest) to confirm review findings with real
// request/response behavior — not static reads of the source.

const evidenceMaster = "review-master-token"

func evidenceMarket() *market.Service {
	return market.New(routerMarket{}, routerSupply{}).WithHolders(routerHolders{})
}

func evidenceDo(t *testing.T, h http.Handler, method, path, token, clientID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if clientID != "" {
		req.Header.Set("X-Client-Id", clientID)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func decodeMap(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("json: %v body=%s", err, rr.Body.String())
	}
	return m
}

func TestReviewEvidence_MasterTokenImpersonatesAnyClientID(t *testing.T) {
	// Finding: a process master token plus any X-Client-Id / clientId is that tenant.
	watch := watchlist.New(watchliststore.NewMemory())
	h := NewRouterWithOptions(evidenceMarket(), watch, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster,
	})

	add := func(client, symbol string) {
		t.Helper()
		rr := evidenceDo(t, h, http.MethodPost, "/api/v1/watchlist/items", evidenceMaster, client, map[string]any{
			"clientId": client, "exchange": "binance", "symbol": symbol,
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("add %s %s: %d %s", client, symbol, rr.Code, rr.Body.String())
		}
	}
	list := func(client string) []any {
		t.Helper()
		rr := evidenceDo(t, h, http.MethodGet, "/api/v1/watchlist?clientId="+client, evidenceMaster, client, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("get %s: %d %s", client, rr.Code, rr.Body.String())
		}
		items, _ := decodeMap(t, rr)["items"].([]any)
		return items
	}

	add("alice", "BTCUSDT")
	add("bob", "ETHUSDT")

	alice := list("alice")
	bob := list("bob")
	if len(alice) != 1 || len(bob) != 1 {
		t.Fatalf("alice=%d bob=%d", len(alice), len(bob))
	}

	// Same master token reads alice's list after acting as bob — no extra credential.
	rr := evidenceDo(t, h, http.MethodGet, "/api/v1/watchlist?clientId=alice", evidenceMaster, "bob", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("impersonate get: %d %s", rr.Code, rr.Body.String())
	}
	// Query clientId wins for Get; header is ignored when query is set. Prove header-only works too:
	rr = evidenceDo(t, h, http.MethodGet, "/api/v1/watchlist", evidenceMaster, "alice", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("header-only get: %d %s", rr.Code, rr.Body.String())
	}
	if items, _ := decodeMap(t, rr)["items"].([]any); len(items) != 1 {
		t.Fatalf("header-only alice items=%v", rr.Body.String())
	}
}

func TestReviewEvidence_ReadKeyCannotPOSTMCP(t *testing.T) {
	// Finding: APIKeyScope treats every /mcp call as trade, so a read key cannot use MCP.
	keys := apikey.New(accountstore.NewMemory())
	created, err := keys.Create(context.Background(), apikey.CreateInput{
		ClientID: "reader-1", Name: "reader", Permission: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	reached := false
	h := NewRouterWithOptions(evidenceMarket(), nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster, APIKeys: keys,
		MCPHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reached = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", created.Secret)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 for read-key POST /mcp, got %d %s", rr.Code, rr.Body.String())
	}
	if reached {
		t.Fatal("MCP handler must not run for a read-only key")
	}
	body := rr.Body.String()
	if !strings.Contains(body, "read-only") {
		t.Fatalf("body=%s", body)
	}

	// GET /mcp is exempt from requiresTradePermission (GET returns false first).
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("X-API-Key", created.Secret)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !reached {
		t.Fatalf("GET /mcp with read key: code=%d reached=%v body=%s", rr.Code, reached, rr.Body.String())
	}
}

func TestReviewEvidence_QueryTokenAuthenticatesTenantRoute(t *testing.T) {
	// Finding: extractAPIToken accepts ?token= / ?apiKey= (lands in logs and history).
	watch := watchlist.New(watchliststore.NewMemory())
	h := NewRouterWithOptions(evidenceMarket(), watch, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster,
	})
	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/watchlist/items", evidenceMaster, "ws-user", map[string]any{
		"clientId": "ws-user", "exchange": "binance", "symbol": "BTCUSDT",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("seed: %d %s", rr.Code, rr.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/watchlist?clientId=ws-user&token="+evidenceMaster, nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("?token= want 200 got %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/watchlist?clientId=ws-user&apiKey="+evidenceMaster, nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("?apiKey= want 200 got %d %s", rr.Code, rr.Body.String())
	}
}

func TestReviewEvidence_MarketTapeRoutesPublicAndJSON(t *testing.T) {
	// Finding: tape/futures HTTP exists and is public (no auth), even when a master token is set.
	h := NewRouterWithOptions(evidenceMarket(), nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster,
	})
	paths := []struct {
		path string
		want int
	}{
		{"/api/v1/market/open-interest?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/funding-rate?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/long-short-ratio?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/cvd?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/taker-flow?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/basis?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/venue-divergence?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/positioning?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/squeeze-risk?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/liquidation-hunt?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/correlation?symbol=SOLUSDT", http.StatusOK},
		{"/api/v1/market/breadth", http.StatusOK},
		{"/api/v1/market/volatility?symbol=SOLUSDT", http.StatusOK},
		{"/api/v1/market/snapshot?symbol=SOLUSDT", http.StatusOK},
		{"/api/v1/market/levels?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/whales?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/orderbook/icebergs?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/orderbook/combined?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/orderbook/liquidity?symbol=BTCUSDT", http.StatusOK},
		{"/api/v1/market/liquidations?symbol=BTCUSDT", http.StatusOK},
	}
	for _, tc := range paths {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			ct := rr.Header().Get("Content-Type")
			if !strings.Contains(ct, "json") {
				t.Fatalf("content-type=%q", ct)
			}
			var payload any
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("not JSON: %v %s", err, rr.Body.String())
			}
		})
	}
}

func TestReviewEvidence_APIKeyHTTPRejectsReservedAndTelegramClientIDs(t *testing.T) {
	keys := apikey.New(accountstore.NewMemory())
	h := NewRouterWithOptions(evidenceMarket(), nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster, APIKeys: keys,
	})

	rejectedByDomain := []string{"anonymous", "ai-assistant", "tg-123456789"}
	for _, id := range rejectedByDomain {
		if _, err := domain.NormalizeClientID(id); err == nil {
			t.Fatalf("precondition: domain must reject %q", id)
		}
		rr := evidenceDo(t, h, http.MethodPost, "/api/v1/account/api-keys", evidenceMaster, id, map[string]any{
			"clientId": id, "name": "Bot", "permission": "trade",
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: want 400 from API-key create, got %d %s", id, rr.Code, rr.Body.String())
		}
	}
}

func TestReviewEvidence_ClosedAccountAlertsStayStoredButCheckerSkips(t *testing.T) {
	store, err := alertstore.Open(filepath.Join(t.TempDir(), "alerts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	alerts := pricealert.New(store)
	alerts.AllowPrivateWebhooks = true
	acct := account.New(accountstore.NewMemory(), account.DataPurgeDeps{Alerts: store})
	h := NewRouterWithOptions(evidenceMarket(), nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster, Alerts: alerts, Accounts: acct,
	})

	const client = "closed-alerts-1"
	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/alerts", evidenceMaster, client, map[string]any{
		"clientId": client, "exchange": "binance", "symbol": "BTCUSDT",
		"condition": "above", "targetPrice": 1,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create alert: %d %s", rr.Code, rr.Body.String())
	}
	alertID, _ := decodeMap(t, rr)["id"].(string)
	if alertID == "" {
		t.Fatalf("no alert id: %s", rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/account/close", evidenceMaster, client, map[string]any{
		"clientId": client,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("close: %d %s", rr.Code, rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodGet, "/api/v1/alerts?clientId="+client, evidenceMaster, client, nil)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("HTTP after close want 403, got %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "account_closed") {
		t.Fatalf("body=%s", rr.Body.String())
	}

	active, err := alerts.ListActive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range active {
		if a.ID == alertID && a.ClientID == client {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("closed-account alert %s missing from ListActive (%d active)", alertID, len(active))
	}

	chk := &pricealert.Checker{
		Alerts:   alerts,
		Market:   evidenceTicker{},
		Accounts: acct,
	}
	chk.RunOnce(context.Background())
	got, err := alerts.Get(context.Background(), client, alertID)
	if err != nil || got.Status != domain.AlertStatusActive {
		t.Fatalf("worker must skip closed tenant: %+v %v", got, err)
	}
}

type evidenceTicker struct{}

func (evidenceTicker) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "999999"}, nil
}

func TestReviewEvidence_ClosedAccountScannerRulesStillEnabled(t *testing.T) {
	st, err := scannerstore.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	scan := scanner.New(st, scanCandles{}, scanWatch{})
	acct := account.New(accountstore.NewMemory(), account.DataPurgeDeps{Scanner: st})
	scan.SetAccountChecker(acct)
	h := NewRouterWithOptions(evidenceMarket(), nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster, Scanner: scan, Accounts: acct,
	})

	const client = "closed-scan-1"
	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/scanner/rules", evidenceMaster, client, map[string]any{
		"clientId": client, "type": "rsi", "interval": "1h",
		"rsiPeriod": 14, "rsiCondition": "below", "rsiThreshold": 30,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create rule: %d %s", rr.Code, rr.Body.String())
	}
	ruleID, _ := decodeMap(t, rr)["id"].(string)

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/account/close", evidenceMaster, client, map[string]any{
		"clientId": client,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("close: %d %s", rr.Code, rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodGet, "/api/v1/scanner/rules?clientId="+client, evidenceMaster, client, nil)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("HTTP after close want 403, got %d %s", rr.Code, rr.Body.String())
	}

	rules, err := scan.ListEnabledRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rules {
		if r.ID == ruleID && r.ClientID == client {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("closed-account scanner rule %s missing from ListEnabledRules", ruleID)
	}
	n, err := scan.RunOnce(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("scanner worker must skip closed tenant n=%d err=%v", n, err)
	}
}

type scanCandles struct{}

func (scanCandles) GetCandles(_ context.Context, _, _, _ string, _ int, _, _ *time.Time) ([]domain.Candle, error) {
	return nil, nil
}

// scanWatch matches scanner.WatchlistReader used by the existing handler test.
type scanWatch struct{}

func (scanWatch) Get(_ context.Context, actorID, ownerID string) (*domain.WatchlistAccess, error) {
	id := ownerID
	if id == "" {
		id = actorID
	}
	return &domain.WatchlistAccess{
		Watchlist:     domain.Watchlist{ClientID: id},
		OwnerClientID: id,
		Role:          domain.WatchlistRoleOwner,
	}, nil
}

func TestReviewEvidence_ClosedAccountPriceDiffWatchesStillActive(t *testing.T) {
	st, err := pricediffstore.Open(filepath.Join(t.TempDir(), "pd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	pd := pricediff.New(st, evidenceMarket())
	acct := account.New(accountstore.NewMemory(), account.DataPurgeDeps{PriceDiff: pd})
	pd.SetAccountChecker(acct)
	h := NewRouterWithOptions(evidenceMarket(), nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster, PriceDiff: pd, Accounts: acct,
	})

	const client = "closed-pdiff-1"
	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/price-diff/watches", evidenceMaster, client, map[string]any{
		"clientId": client, "symbol": "BTCUSDT", "minNetDiffPct": 0.5,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create watch: %d %s", rr.Code, rr.Body.String())
	}
	watchID, _ := decodeMap(t, rr)["id"].(string)

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/account/close", evidenceMaster, client, map[string]any{
		"clientId": client,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("close: %d %s", rr.Code, rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodGet, "/api/v1/price-diff/watches?clientId="+client, evidenceMaster, client, nil)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("HTTP after close want 403, got %d %s", rr.Code, rr.Body.String())
	}

	watches, err := st.ListActiveWatches(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range watches {
		if w.ID == watchID && w.ClientID == client {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("closed-account price-diff watch %s missing from ListActiveWatches", watchID)
	}
	created, closedN, touched, err := pd.ProcessActiveWatches(context.Background(), time.Now().UTC())
	if err != nil || created != 0 || closedN != 0 || touched != 0 {
		t.Fatalf("price-diff worker must skip closed tenant created=%d closed=%d touched=%d err=%v", created, closedN, touched, err)
	}
}

func TestReviewEvidence_HTTPMasterAIChatGrantsKeyAdmin(t *testing.T) {
	// Finding: HTTP chat without a user key (master/open) sends canTrade+canManageKeys.
	var got aiagent.ChatRequest
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"reply": "ok", "sessionId": "s"})
	}))
	t.Cleanup(aiSrv.Close)

	h := NewRouterWithOptions(evidenceMarket(), nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster,
		AI: aiagent.New(aiSrv.URL, 0),
	})
	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/ai/chat", evidenceMaster, "desk-user-1", map[string]any{
		"message": "mint a key", "sessionId": "s",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("chat: %d %s", rr.Code, rr.Body.String())
	}
	if !got.CanTrade || !got.CanManageKeys {
		t.Fatalf("master/open AI scope trade=%v keys=%v (want both true)", got.CanTrade, got.CanManageKeys)
	}
	if got.ClientID != "desk-user-1" {
		t.Fatalf("clientId=%q", got.ClientID)
	}
}
