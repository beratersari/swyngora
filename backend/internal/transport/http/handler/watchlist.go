package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
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
	if id := middleware.IdentityFrom(r.Context()); id != nil && id.ClientID != "" {
		return id.ClientID
	}
	if v := strings.TrimSpace(r.Header.Get("X-Client-Id")); v != "" {
		return v
	}
	return strings.TrimSpace(r.URL.Query().Get("clientId"))
}

func ownerClientIDFrom(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("ownerClientId"))
}

func portfolioIDFrom(r *http.Request) string {
	if v := strings.TrimSpace(r.URL.Query().Get("portfolioId")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("X-Portfolio-Id")); v != "" {
		return v
	}
	return ""
}

func coalescePortfolioID(r *http.Request, bodyID string) string {
	if v := strings.TrimSpace(bodyID); v != "" {
		return v
	}
	return portfolioIDFrom(r)
}

func coalesceOwner(r *http.Request, bodyOwner string) string {
	if v := strings.TrimSpace(bodyOwner); v != "" {
		return v
	}
	return ownerClientIDFrom(r)
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
	Version       int64              `json:"version"`
	Items         []watchlistItemDTO `json:"items"`
	UpdatedAt     string             `json:"updatedAt"`
}

func itemToDTO(it domain.WatchlistItem) watchlistItemDTO {
	return watchlistItemDTO{
		Exchange: string(it.Exchange), Symbol: it.Symbol, Note: it.Note,
		AddedAt: it.AddedAt.UTC().Format(time.RFC3339Nano),
	}
}

func itemsToDTO(items []domain.WatchlistItem) []watchlistItemDTO {
	out := make([]watchlistItemDTO, 0, len(items))
	for _, it := range items {
		out = append(out, itemToDTO(it))
	}
	return out
}

func accessToDTO(acc *domain.WatchlistAccess) watchlistDTO {
	return watchlistDTO{
		ClientID: acc.ClientID, OwnerClientID: acc.OwnerClientID, Role: string(acc.Role),
		Version: acc.Version, Items: itemsToDTO(acc.Items),
		UpdatedAt: acc.Updated.UTC().Format(time.RFC3339Nano),
	}
}

type conflictItemDTO struct {
	Exchange   string            `json:"exchange"`
	Symbol     string            `json:"symbol"`
	Type       string            `json:"type"`
	ServerItem *watchlistItemDTO `json:"serverItem"`
	ClientItem *watchlistItemDTO `json:"clientItem"`
}

type conflictBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Conflict struct {
		BaseVersion    int64             `json:"baseVersion"`
		ServerVersion  int64             `json:"serverVersion"`
		Server         watchlistDTO      `json:"server"`
		ClientProposed []watchlistItemDTO `json:"clientProposed,omitempty"`
		AutoMerged     []watchlistItemDTO `json:"autoMerged"`
		Conflicts      []conflictItemDTO `json:"conflicts"`
	} `json:"conflict"`
}

func writeWatchlistConflict(w http.ResponseWriter, err error) {
	var sc *domain.WatchlistSyncConflict
	if !errors.As(err, &sc) {
		writeError(w, err)
		return
	}
	var body conflictBody
	body.Error.Code = "conflict"
	body.Error.Message = "watchlist version mismatch; resolve symbol conflicts"
	body.Conflict.BaseVersion = sc.BaseVersion
	body.Conflict.ServerVersion = sc.ServerVersion
	body.Conflict.Server = watchlistDTO{
		ClientID: sc.Server.ClientID, OwnerClientID: sc.Server.ClientID, Role: "owner",
		Version: sc.Server.Version, Items: itemsToDTO(sc.Server.Items),
		UpdatedAt: sc.Server.Updated.UTC().Format(time.RFC3339Nano),
	}
	body.Conflict.ClientProposed = itemsToDTO(sc.ClientProposed)
	body.Conflict.AutoMerged = itemsToDTO(sc.AutoMerged)
	for _, c := range sc.Conflicts {
		ci := conflictItemDTO{
			Exchange: string(c.Exchange), Symbol: c.Symbol, Type: string(c.Type),
		}
		if c.ServerItem != nil {
			d := itemToDTO(*c.ServerItem)
			ci.ServerItem = &d
		}
		if c.ClientItem != nil {
			d := itemToDTO(*c.ClientItem)
			ci.ClientItem = &d
		}
		body.Conflict.Conflicts = append(body.Conflict.Conflicts, ci)
	}
	writeJSON(w, http.StatusConflict, body)
}

func parseBaseVersion(raw string, body *int64) int64 {
	// Prefer explicit body field when set (>=0 or -1).
	if body != nil {
		return *body
	}
	if raw == "" {
		// Backward compatible: omit baseVersion → unconditional write.
		return domain.WatchlistUnconditionalVersion
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return domain.WatchlistUnconditionalVersion
	}
	return v
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

type itemIn struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Note     string `json:"note"`
}

type replaceBody struct {
	ClientID      string   `json:"clientId"`
	OwnerClientID string   `json:"ownerClientId"`
	BaseVersion   *int64   `json:"baseVersion"`
	BaseItems     []itemIn `json:"baseItems"`
	Items         []itemIn `json:"items"`
}

// Replace handles PUT /api/v1/watchlist (owner only).
// Send baseVersion from the last GET. On multi-device conflict returns 409 with both sides.
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
	var baseItems []domain.WatchlistItem
	if body.BaseItems != nil {
		baseItems = make([]domain.WatchlistItem, 0, len(body.BaseItems))
		for _, it := range body.BaseItems {
			baseItems = append(baseItems, domain.WatchlistItem{
				Exchange: domain.Exchange(it.Exchange), Symbol: it.Symbol, Note: it.Note,
			})
		}
	}
	baseVer := parseBaseVersion(r.Header.Get("If-Match"), body.BaseVersion)
	// Also accept query baseVersion
	if body.BaseVersion == nil {
		if q := r.URL.Query().Get("baseVersion"); q != "" {
			baseVer = parseBaseVersion(q, nil)
		}
	}
	acc, err := h.svc.Replace(r.Context(), actor, owner, items, baseVer, baseItems)
	if err != nil {
		writeWatchlistConflict(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accessToDTO(acc))
}

type addBody struct {
	ClientID      string `json:"clientId"`
	OwnerClientID string `json:"ownerClientId"`
	BaseVersion   *int64 `json:"baseVersion"`
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
	baseVer := parseBaseVersion(r.Header.Get("If-Match"), body.BaseVersion)
	acc, err := h.svc.Add(r.Context(), actor, owner, body.Exchange, body.Symbol, body.Note, baseVer)
	if err != nil {
		writeWatchlistConflict(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accessToDTO(acc))
}

// Remove handles DELETE /api/v1/watchlist/items (owner or editor).
func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	actor := clientIDFrom(r)
	owner := ownerClientIDFrom(r)
	baseVer := parseBaseVersion(q.Get("baseVersion"), nil)
	if h := r.Header.Get("If-Match"); h != "" {
		baseVer = parseBaseVersion(h, nil)
	}
	acc, err := h.svc.Remove(r.Context(), actor, owner, q.Get("exchange"), q.Get("symbol"), baseVer)
	if err != nil {
		writeWatchlistConflict(w, err)
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
