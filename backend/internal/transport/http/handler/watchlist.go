package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// WatchlistHandler is the transport adapter for watchlists.
type WatchlistHandler struct {
	svc *watchlist.Service
}

// NewWatchlistHandler constructs the handler.
func NewWatchlistHandler(svc *watchlist.Service) *WatchlistHandler {
	return &WatchlistHandler{svc: svc}
}

func clientIDFrom(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Client-Id")); v != "" {
		return v
	}
	return strings.TrimSpace(r.URL.Query().Get("clientId"))
}

type watchlistItemDTO struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Note     string `json:"note,omitempty"`
	AddedAt  string `json:"addedAt"`
}

type watchlistDTO struct {
	ClientID  string             `json:"clientId"`
	Items     []watchlistItemDTO `json:"items"`
	UpdatedAt string             `json:"updatedAt"`
}

func toDTO(wl *domain.Watchlist) watchlistDTO {
	items := make([]watchlistItemDTO, 0, len(wl.Items))
	for _, it := range wl.Items {
		items = append(items, watchlistItemDTO{
			Exchange: string(it.Exchange),
			Symbol:   it.Symbol,
			Note:     it.Note,
			AddedAt:  it.AddedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return watchlistDTO{
		ClientID:  wl.ClientID,
		Items:     items,
		UpdatedAt: wl.Updated.UTC().Format(time.RFC3339Nano),
	}
}

// Get handles GET /api/v1/watchlist
func (h *WatchlistHandler) Get(w http.ResponseWriter, r *http.Request) {
	wl, err := h.svc.Get(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(wl))
}

type replaceBody struct {
	ClientID string `json:"clientId"`
	Items    []struct {
		Exchange string `json:"exchange"`
		Symbol   string `json:"symbol"`
		Note     string `json:"note"`
	} `json:"items"`
}

// Replace handles PUT /api/v1/watchlist
func (h *WatchlistHandler) Replace(w http.ResponseWriter, r *http.Request) {
	var body replaceBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	items := make([]domain.WatchlistItem, 0, len(body.Items))
	for _, it := range body.Items {
		items = append(items, domain.WatchlistItem{
			Exchange: domain.Exchange(it.Exchange),
			Symbol:   it.Symbol,
			Note:     it.Note,
		})
	}
	wl, err := h.svc.Replace(r.Context(), clientID, items)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(wl))
}

type addBody struct {
	ClientID string `json:"clientId"`
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Note     string `json:"note"`
}

// Add handles POST /api/v1/watchlist/items
func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	var body addBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	wl, err := h.svc.Add(r.Context(), clientID, body.Exchange, body.Symbol, body.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(wl))
}

// Remove handles DELETE /api/v1/watchlist/items
func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := clientIDFrom(r)
	wl, err := h.svc.Remove(r.Context(), clientID, q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(wl))
}
