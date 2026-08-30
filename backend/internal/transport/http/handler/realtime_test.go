package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/realtime"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

type rtMarket struct {
	last string
}

func (m *rtMarket) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	p := m.last
	if p == "" {
		p = "42"
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p, PriceChangePercent: "1"}, nil
}

type rtAccess struct {
	deny error
}

func (a *rtAccess) CanViewPortfolio(_ context.Context, _, _ string) error { return a.deny }

func (a *rtAccess) RealtimeSnapshot(_ context.Context, actor, book string) (*domain.PortfolioView, []domain.PendingOrder, error) {
	if a.deny != nil {
		return nil, nil, a.deny
	}
	return &domain.PortfolioView{
		ID: book, ClientID: "owner", Name: "Main", Role: domain.PortfolioRoleViewer,
		CashBalance: 1000, Equity: 1000, Positions: []domain.PositionView{},
	}, nil, nil
}

func TestRealtimeIssueTicketHTTP(t *testing.T) {
	hub := realtime.NewHub(realtime.Options{Market: &rtMarket{}, Access: &rtAccess{}, Interval: time.Hour})
	iss := middleware.NewWSTicketIssuer()
	h := NewRealtimeHandler(hub, nil).WithTickets(iss)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/realtime/ticket", nil)
	req.Header.Set("X-Client-Id", "alice")
	rec := httptest.NewRecorder()
	h.IssueTicket(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.Bytes())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	tok, _ := body["ticket"].(string)
	if tok == "" || body["clientId"] != "alice" {
		t.Fatalf("body=%v", body)
	}
	id, err := iss.Consume(tok)
	if err != nil || id == nil || id.ClientID != "alice" {
		t.Fatalf("consume: %+v %v", id, err)
	}
}

func TestRealtimeInfoHTTP(t *testing.T) {
	hub := realtime.NewHub(realtime.Options{Market: &rtMarket{}, Access: &rtAccess{}, Interval: time.Hour})
	h := NewRealtimeHandler(hub, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/realtime", nil)
	rec := httptest.NewRecorder()
	h.Info(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.Bytes())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["path"] != "/api/v1/ws" {
		t.Fatalf("body=%v", body)
	}
}

func TestRealtimeWS_SubscribePricesAndPortfolioAccess(t *testing.T) {
	acc := &rtAccess{}
	hub := realtime.NewHub(realtime.Options{Market: &rtMarket{last: "99.5"}, Access: acc, Interval: time.Hour})
	h := NewRealtimeHandler(hub, []string{"*"})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ws", h.ServeWS)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws?clientId=viewer-1"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	hello := readWSMap(t, conn)
	if hello["type"] != "hello" || hello["clientId"] != "viewer-1" {
		t.Fatalf("hello=%v", hello)
	}

	if err := conn.WriteJSON(map[string]any{
		"type":    "subscribe_prices",
		"symbols": []map[string]string{{"exchange": "binance", "symbol": "BTCUSDT"}},
	}); err != nil {
		t.Fatal(err)
	}
	ack := readWSMap(t, conn)
	if ack["type"] != "ack" || ack["op"] != "subscribe_prices" {
		t.Fatalf("ack=%v", ack)
	}
	price := readWSMap(t, conn)
	if price["type"] != "price" || price["lastPrice"] != "99.5" || price["symbol"] != "BTCUSDT" {
		t.Fatalf("price=%v", price)
	}

	if err := conn.WriteJSON(map[string]any{"type": "subscribe_portfolio", "portfolioId": "book-9"}); err != nil {
		t.Fatal(err)
	}
	pack := readWSMap(t, conn)
	if pack["type"] != "ack" || pack["op"] != "subscribe_portfolio" {
		t.Fatalf("pack=%v", pack)
	}
	snap := readWSMap(t, conn)
	if snap["type"] != "portfolio" || snap["reason"] != "snapshot" {
		t.Fatalf("snap=%v", snap)
	}
	pf, _ := snap["portfolio"].(map[string]any)
	if pf["id"] != "book-9" || pf["role"] != "viewer" {
		t.Fatalf("portfolio dto=%v", pf)
	}

	// Second connection cannot see owner-only book if access denied.
	acc.deny = domain.ErrForbidden
	if err := conn.WriteJSON(map[string]any{"type": "subscribe_portfolio", "portfolioId": "secret"}); err != nil {
		t.Fatal(err)
	}
	denied := readWSMap(t, conn)
	if denied["type"] != "error" || denied["code"] != "forbidden" {
		t.Fatalf("denied=%v", denied)
	}
}

func TestRealtimeWS_RequiresClientID(t *testing.T) {
	hub := realtime.NewHub(realtime.Options{Interval: time.Hour})
	h := NewRealtimeHandler(hub, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	rec := httptest.NewRecorder()
	h.ServeWS(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.Bytes())
	}
}

func readWSMap(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json %s: %v", data, err)
	}
	return m
}
