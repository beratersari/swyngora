package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/realtime"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = 25 * time.Second
	wsMaxMsg     = 32 << 10
)

// RealtimeHandler upgrades WebSocket connections onto the shared hub.
type RealtimeHandler struct {
	hub      *realtime.Hub
	upgrader websocket.Upgrader
}

// NewRealtimeHandler constructs the WS adapter.
func NewRealtimeHandler(hub *realtime.Hub, allowOrigins []string) *RealtimeHandler {
	return &RealtimeHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     wsCheckOrigin(allowOrigins),
		},
	}
}

type clientWSMessage struct {
	Type        string `json:"type"`
	PortfolioID string `json:"portfolioId"`
	Symbols     []struct {
		Exchange string `json:"exchange"`
		Symbol   string `json:"symbol"`
	} `json:"symbols"`
}

// Info handles GET /api/v1/realtime — protocol description (MCP / clients).
func (h *RealtimeHandler) Info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, realtimeInfoDTO())
}

// ServeWS handles GET /api/v1/ws.
func (h *RealtimeHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.hub == nil {
		writeError(w, fmt.Errorf("%w: realtime not configured", domain.ErrUpstream))
		return
	}
	clientID := clientIDFrom(r)
	if clientID == "" {
		writeError(w, fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument))
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	sess := h.hub.Register(uuid.NewString(), clientID)
	go h.writeLoop(conn, sess)
	h.readLoop(conn, sess)
}

func (h *RealtimeHandler) readLoop(conn *websocket.Conn, sess *realtime.Session) {
	defer func() {
		h.hub.Unregister(sess)
		_ = conn.Close()
	}()
	conn.SetReadLimit(wsMaxMsg)
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
		var msg clientWSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			sessSendError(sess, "invalid_argument", "invalid JSON")
			continue
		}
		h.handleClient(sess, msg)
	}
}

func (h *RealtimeHandler) handleClient(sess *realtime.Session, msg clientWSMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	switch strings.ToLower(strings.TrimSpace(msg.Type)) {
	case domain.RealtimeOpPing:
		sessSend(sess, realtime.Outbound{"type": domain.RealtimeTypePong, "ts": time.Now().UTC().Format(time.RFC3339Nano)})
	case domain.RealtimeOpSubscribePrices:
		refs, err := parseSymbolRefs(msg.Symbols)
		if err != nil {
			sessSendError(sess, "invalid_argument", publicMessage(err))
			return
		}
		if err := h.hub.SubscribePrices(ctx, sess, refs); err != nil {
			sessSendError(sess, errorCode(err), publicMessage(err))
		}
	case domain.RealtimeOpUnsubscribePrices:
		var refs []domain.SymbolRef
		if len(msg.Symbols) > 0 {
			var err error
			refs, err = parseSymbolRefs(msg.Symbols)
			if err != nil {
				sessSendError(sess, "invalid_argument", publicMessage(err))
				return
			}
		}
		h.hub.UnsubscribePrices(sess, refs)
	case domain.RealtimeOpSubscribePortfolio:
		if err := h.hub.SubscribePortfolio(ctx, sess, msg.PortfolioID); err != nil {
			sessSendError(sess, errorCode(err), publicMessage(err))
		}
	case domain.RealtimeOpUnsubscribePortfolio:
		h.hub.UnsubscribePortfolio(sess)
	default:
		sessSendError(sess, "invalid_argument", "unknown type")
	}
}

func (h *RealtimeHandler) writeLoop(conn *websocket.Conn, sess *realtime.Session) {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	for {
		select {
		case msg, ok := <-sess.Out:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			payload := encodeRealtimeMessage(msg)
			if err := conn.WriteJSON(payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func encodeRealtimeMessage(msg any) any {
	switch v := msg.(type) {
	case realtime.Outbound:
		return v
	case domain.PortfolioChange:
		return encodePortfolioChange(v)
	default:
		return msg
	}
}

func encodePortfolioChange(ev domain.PortfolioChange) map[string]any {
	out := map[string]any{
		"type":        domain.RealtimeTypePortfolio,
		"reason":      ev.Reason,
		"portfolioId": ev.PortfolioID,
	}
	if ev.View != nil {
		out["portfolio"] = portfolioViewDTO(ev.View)
		if ev.PortfolioID == "" {
			out["portfolioId"] = ev.View.ID
		}
	}
	if ev.Order != nil {
		out["order"] = pendingOrderToDTO(ev.Order)
	}
	if ev.Trade != nil {
		out["trade"] = tradeToDTO(ev.Trade)
	}
	if ev.Orders != nil {
		items := make([]pendingOrderDTO, 0, len(ev.Orders))
		for i := range ev.Orders {
			items = append(items, pendingOrderToDTO(&ev.Orders[i]))
		}
		out["orders"] = items
	}
	return out
}

func parseSymbolRefs(in []struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
}) ([]domain.SymbolRef, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("%w: symbols required", domain.ErrInvalidArgument)
	}
	out := make([]domain.SymbolRef, 0, len(in))
	for _, s := range in {
		ref, err := domain.NormalizeSymbolRef(s.Exchange, s.Symbol)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func sessSend(s *realtime.Session, msg any) {
	if s == nil {
		return
	}
	select {
	case s.Out <- msg:
	default:
	}
}

func sessSendError(s *realtime.Session, code, message string) {
	sessSend(s, realtime.Outbound{"type": domain.RealtimeTypeError, "code": code, "message": message})
}

func errorCode(err error) string {
	_, code, _ := mapError(err)
	return code
}

func publicMessage(err error) string {
	_, _, msg := mapError(err)
	return msg
}

func realtimeInfoDTO() map[string]any {
	return map[string]any{
		"path":        domain.RealtimeWSPath,
		"protocol":    domain.RealtimeProtocolVersion,
		"maxSymbols":  domain.MaxRealtimePriceSymbols,
		"auth":        "Same as REST (Bearer / X-API-Key). Browsers: ?token= and ?clientId= on the WebSocket URL.",
		"reconnect":   "Resubscribe after reconnect; the server snapshots current prices and the selected portfolio.",
		"channels":    []string{"prices", "portfolio"},
		"clientTypes": []string{domain.RealtimeOpSubscribePrices, domain.RealtimeOpUnsubscribePrices, domain.RealtimeOpSubscribePortfolio, domain.RealtimeOpUnsubscribePortfolio, domain.RealtimeOpPing},
		"serverTypes": []string{domain.RealtimeTypeHello, domain.RealtimeTypeAck, domain.RealtimeTypePrice, domain.RealtimeTypePortfolio, domain.RealtimeTypeError, domain.RealtimeTypePong},
		"access":      "Portfolio events are sent only if the clientId can view that book (owner, trader, or viewer share).",
	}
}

func wsCheckOrigin(allowed []string) func(*http.Request) bool {
	allowAll := len(allowed) == 0
	set := map[string]struct{}{}
	for _, o := range allowed {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			break
		}
		set[o] = struct{}{}
	}
	if allowAll {
		return func(*http.Request) bool { return true }
	}
	return func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			return true
		}
		_, ok := set[origin]
		return ok
	}
}
