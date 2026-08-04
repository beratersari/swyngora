package handler

import (
	"fmt"
	"net/http"
	"strconv"
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

func ownerClientIDFrom(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("ownerClientId"))
}

type watchlistItemDTO struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Note     string `json:"note,omitempty"`
	AddedAt  string `json:"addedAt"`
}

type watchlistDTO struct {
	ClientID      string             `json:"clientId"`
	OwnerClientID string             `json:"ownerClientId"`
	Role          string             `json:"role"`
	Items         []watchlistItemDTO `json:"items"`
	UpdatedAt     string             `json:"updatedAt"`
}

func accessToDTO(acc *domain.WatchlistAccess) watchlistDTO {
	items := make([]watchlistItemDTO, 0, len(acc.Items))
	for _, it := range acc.Items {
		items = append(items, watchlistItemDTO{
			Exchange: string(it.Exchange), Symbol: it.Symbol, Note: it.Note,
			AddedAt: it.AddedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return watchlistDTO{
		ClientID: acc.ClientID, OwnerClientID: acc.OwnerClientID, Role: string(acc.Role),
		Items: items, UpdatedAt: acc.Updated.UTC().Format(time.RFC3339Nano),
	}
}

// Get handles GET /api/v1/watchlist
// Optional ownerClientId: view a shared list (defaults to own).
func (h *WatchlistHandler) Get(w http.ResponseWriter, r *http.Request) {
	acc, err := h.svc.Get(r.Context(), clientIDFrom(r), ownerClientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accessToDTO(acc))
}

type replaceBody struct {
	ClientID      string `json:"clientId"`
	OwnerClientID string `json:"ownerClientId"`
	Items         []struct {
		Exchange string `json:"exchange"`
		Symbol   string `json:"symbol"`
		Note     string `json:"note"`
	} `json:"items"`
}

// Replace handles PUT /api/v1/watchlist (owner only).
func (h *WatchlistHandler) Replace(w http.ResponseWriter, r *http.Request) {
	var body replaceBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	actor := body.ClientID
	if actor == "" {
		actor = clientIDFrom(r)
	}
	owner := body.OwnerClientID
	if owner == "" {
		owner = ownerClientIDFrom(r)
	}
	items := make([]domain.WatchlistItem, 0, len(body.Items))
	for _, it := range body.Items {
		items = append(items, domain.WatchlistItem{
			Exchange: domain.Exchange(it.Exchange), Symbol: it.Symbol, Note: it.Note,
		})
	}
	acc, err := h.svc.Replace(r.Context(), actor, owner, items)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accessToDTO(acc))
}

type addBody struct {
	ClientID      string `json:"clientId"`
	OwnerClientID string `json:"ownerClientId"`
	Exchange      string `json:"exchange"`
	Symbol        string `json:"symbol"`
	Note          string `json:"note"`
}

// Add handles POST /api/v1/watchlist/items (owner or editor).
func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	var body addBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	actor := body.ClientID
	if actor == "" {
		actor = clientIDFrom(r)
	}
	owner := body.OwnerClientID
	if owner == "" {
		owner = ownerClientIDFrom(r)
	}
	acc, err := h.svc.Add(r.Context(), actor, owner, body.Exchange, body.Symbol, body.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accessToDTO(acc))
}

// Remove handles DELETE /api/v1/watchlist/items (owner or editor).
func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	actor := clientIDFrom(r)
	owner := ownerClientIDFrom(r)
	acc, err := h.svc.Remove(r.Context(), actor, owner, q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accessToDTO(acc))
}

type shareBody struct {
	ClientID        string `json:"clientId"`
	GranteeClientID string `json:"granteeClientId"`
	Role            string `json:"role"`
}

type shareDTO struct {
	OwnerClientID   string `json:"ownerClientId"`
	GranteeClientID string `json:"granteeClientId"`
	Role            string `json:"role"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func shareToDTO(sh *domain.WatchlistShare) shareDTO {
	return shareDTO{
		OwnerClientID: sh.OwnerClientID, GranteeClientID: sh.GranteeClientID, Role: string(sh.Role),
		CreatedAt: sh.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: sh.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// Share handles POST /api/v1/watchlist/shares (owner only; fails if already shared).
func (h *WatchlistHandler) Share(w http.ResponseWriter, r *http.Request) {
	var body shareBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	owner := body.ClientID
	if owner == "" {
		owner = clientIDFrom(r)
	}
	sh, err := h.svc.Share(r.Context(), owner, body.GranteeClientID, body.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, shareToDTO(sh))
}

// UpdateShare handles PATCH /api/v1/watchlist/shares (owner only; change role).
func (h *WatchlistHandler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	var body shareBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	owner := body.ClientID
	if owner == "" {
		owner = clientIDFrom(r)
	}
	sh, err := h.svc.UpdateShareRole(r.Context(), owner, body.GranteeClientID, body.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shareToDTO(sh))
}

// ListShares handles GET /api/v1/watchlist/shares
func (h *WatchlistHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListShares(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]shareDTO, 0, len(list))
	for i := range list {
		items = append(items, shareToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ownerClientId": clientIDFrom(r), "shares": items, "count": len(items),
	})
}

// RevokeShare handles DELETE /api/v1/watchlist/shares
func (h *WatchlistHandler) RevokeShare(w http.ResponseWriter, r *http.Request) {
	grantee := strings.TrimSpace(r.URL.Query().Get("granteeClientId"))
	if grantee == "" {
		writeError(w, fmt.Errorf("%w: granteeClientId is required", domain.ErrInvalidArgument))
		return
	}
	if err := h.svc.RevokeShare(r.Context(), clientIDFrom(r), grantee); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true, "granteeClientId": grantee})
}

// ListSharedWithMe handles GET /api/v1/watchlist/shared
func (h *WatchlistHandler) ListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListSharedWithMe(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]shareDTO, 0, len(list))
	for i := range list {
		items = append(items, shareToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "shares": items, "count": len(items),
	})
}

type auditDTO struct {
	ID            string `json:"id"`
	OwnerClientID string `json:"ownerClientId"`
	ActorClientID string `json:"actorClientId"`
	Action        string `json:"action"`
	Exchange      string `json:"exchange,omitempty"`
	Symbol        string `json:"symbol,omitempty"`
	Detail        string `json:"detail,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

// ListAudit handles GET /api/v1/watchlist/audit
func (h *WatchlistHandler) ListAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.ListAudit(r.Context(), clientIDFrom(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]auditDTO, 0, len(list))
	for _, ev := range list {
		items = append(items, auditDTO{
			ID: ev.ID, OwnerClientID: ev.OwnerClientID, ActorClientID: ev.ActorClientID,
			Action: string(ev.Action), Exchange: ev.Exchange, Symbol: ev.Symbol, Detail: ev.Detail,
			CreatedAt: ev.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ownerClientId": clientIDFrom(r), "events": items, "count": len(items),
	})
}
