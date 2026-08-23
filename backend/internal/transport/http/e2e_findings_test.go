package httpx

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/deliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// haltVenue has no live book for GONEUSDT; last print is only on candles.
type haltVenue struct{ routerMarket }

func (haltVenue) GetTicker24h(context.Context, string) (*domain.Ticker24h, error) {
	return nil, fmt.Errorf("symbol removed from book")
}

func (haltVenue) GetCandles(_ context.Context, _ domain.CandleQuery) ([]domain.Candle, error) {
	return []domain.Candle{{
		OpenTime: time.Unix(1_700_000_000, 0).UTC(), Open: "80", High: "81", Low: "76", Close: "77.5",
		Volume: "10", CloseTime: time.Unix(1_700_003_600, 0).UTC(), QuoteVolume: "775", TradeCount: 4,
	}}, nil
}

func e2ePortfolioRouter(t *testing.T, mkt *market.Service) (*portfolio.Service, http.Handler) {
	t.Helper()
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	pf := portfolio.New(st, mkt).WithPaperCosts(domain.ZeroTradingCosts)
	h := NewRouterWithOptions(mkt, nil, RouterOptions{
		RateLimitRPS: 0, Portfolio: pf,
	})
	return pf, h
}

func TestE2E_ReadAPIKeyCannotPlacePaperOrder(t *testing.T) {
	// Production path: Settings create-key (HTTP) → browser would send that secret
	// on later POSTs. A read key must 403 paper orders through the real router.
	keys := apikey.New(accountstore.NewMemory())
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	mkt := evidenceMarket()
	pf := portfolio.New(st, mkt).WithPaperCosts(domain.ZeroTradingCosts)
	h := NewRouterWithOptions(mkt, nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster, APIKeys: keys, Portfolio: pf,
	})

	const client = "e2e-readkey"
	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/portfolio", evidenceMaster, client, map[string]any{
		"clientId": client, "startingBalance": 10000,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create book %d %s", rr.Code, rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/account/api-keys", evidenceMaster, client, map[string]any{
		"clientId": client, "name": "desk-read", "permission": "read",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create key %d %s", rr.Code, rr.Body.String())
	}
	created := decodeMap(t, rr)
	secret, _ := created["secret"].(string)
	if secret == "" || created["permission"] != "read" {
		t.Fatalf("expected one-time read secret, got %s", rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/portfolio/orders", secret, client, map[string]any{
		"clientId": client, "symbol": "BTCUSDT", "side": "buy", "quantity": 1,
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("read key paper order want 403, got %d %s", rr.Code, rr.Body.String())
	}
	if got := rr.Body.String(); !strings.Contains(got, "read-only") {
		t.Fatalf("want read-only error, got %s", got)
	}
}

func TestE2E_WatchlistRequiresTokenWhenAPIAuthEnabled(t *testing.T) {
	// Mobile prepareHeaders only sets X-Client-Id. Off-loopback the API requires a token.
	watch := watchlist.New(watchliststore.NewMemory())
	h := NewRouterWithOptions(evidenceMarket(), watch, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster,
	})
	rr := evidenceDo(t, h, http.MethodGet, "/api/v1/watchlist", "", "mobile-user", nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("watchlist without token want 401, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestE2E_MarginOpenInterestCursorAndWorker(t *testing.T) {
	// HTTP market-open a margin position, then run the same ProcessMarginInterest
	// the background worker calls.
	mkt := evidenceMarket()
	pf, h := e2ePortfolioRouter(t, mkt)
	const client = "e2e-margin"

	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/portfolio", "", client, map[string]any{
		"clientId": client, "startingBalance": 10000,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	openedAtWall := time.Now().UTC()
	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/portfolio/margin/orders", "", client, map[string]any{
		"clientId": client, "symbol": "BTCUSDT", "side": "long", "type": "market",
		"quantity": 1, "leverage": 2,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("open %d %s", rr.Code, rr.Body.String())
	}
	pos := asMap(t, decodeMap(t, rr)["position"])
	openedAt, err := time.Parse(time.RFC3339Nano, str(pos["openedAt"]))
	if err != nil {
		openedAt, err = time.Parse(time.RFC3339, str(pos["openedAt"]))
	}
	if err != nil {
		t.Fatalf("openedAt %v %s", err, rr.Body.String())
	}
	lastInt, err := time.Parse(time.RFC3339Nano, str(pos["lastInterestAt"]))
	if err != nil {
		lastInt, err = time.Parse(time.RFC3339, str(pos["lastInterestAt"]))
	}
	if err != nil {
		t.Fatalf("lastInterestAt %v %s", err, rr.Body.String())
	}
	if lastInt.Before(openedAt.Add(-time.Millisecond)) {
		t.Fatalf("HTTP lastInterestAt=%s is before openedAt=%s (truncated to hour %s)",
			lastInt, openedAt, openedAt.Truncate(time.Hour))
	}

	// Worker tick at the next clock hour — still less than 1h after open unless we
	// opened in the first millisecond of the hour.
	nextHour := openedAt.UTC().Truncate(time.Hour).Add(time.Hour).Add(time.Second)
	if nextHour.Sub(openedAt) >= time.Hour {
		nextHour = openedAt.Add(30 * time.Minute)
	}
	accrued, liquidated, err := pf.ProcessMarginInterest(context.Background(), nextHour)
	if err != nil {
		t.Fatal(err)
	}
	if accrued != 0 || liquidated != 0 {
		t.Fatalf("worker charged interest %d (liq=%d) at %s after only %s open (opened=%s last=%s)",
			accrued, liquidated, nextHour, nextHour.Sub(openedAt), openedAt, lastInt)
	}

	rr = evidenceDo(t, h, http.MethodGet, "/api/v1/portfolio/margin/positions", "", client, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list %d %s", rr.Code, rr.Body.String())
	}
	items, _ := decodeMap(t, rr)["positions"].([]any)
	if len(items) != 1 {
		t.Fatalf("positions %s", rr.Body.String())
	}
	got := asMap(t, items[0])
	if n, _ := got["debtInterest"].(float64); n != 0 {
		t.Fatalf("GET debtInterest=%v after early worker tick; opened=%s wall=%s", n, openedAt, openedAtWall)
	}
}

func TestE2E_HaltedDelistTickerDoesNotFillPaperOrder(t *testing.T) {
	// Pair is gone from the live book; market.GetTicker24h falls back to the last
	// halt candle and marks Halted. Paper lastPrice must not treat that as live.
	delist := deliststore.NewMemory()
	delist.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{{
		Symbol: "GONEUSDT", Exchange: domain.ExchangeBinance, DelistTime: time.Now().UTC().Add(-time.Hour),
	}})
	mkt := market.New(haltVenue{}, routerSupply{}).WithDelistStore(delist)
	_, h := e2ePortfolioRouter(t, mkt)
	const client = "e2e-halt"

	rr := evidenceDo(t, h, http.MethodGet, "/api/v1/market/ticker/24h?exchange=binance&symbol=GONEUSDT", "", "", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("ticker %d %s", rr.Code, rr.Body.String())
	}
	tkr := decodeMap(t, rr)
	if tkr["halted"] != true {
		t.Fatalf("expected halted ticker, got %s", rr.Body.String())
	}
	if tkr["lastPrice"] != "77.5" {
		t.Fatalf("expected last print 77.5, got %s", rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/portfolio", "", client, map[string]any{
		"clientId": client, "startingBalance": 10000,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/portfolio/orders", "", client, map[string]any{
		"clientId": client, "exchange": "binance", "symbol": "GONEUSDT", "side": "buy", "quantity": 1,
	})
	if rr.Code == http.StatusOK {
		t.Fatalf("filled halted last print as live: %s", rr.Body.String())
	}
	if rr.Code < 400 {
		t.Fatalf("want client/upstream error, got %d %s", rr.Code, rr.Body.String())
	}
}

func asMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("want object, got %T %v", v, v)
	}
	return m
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
