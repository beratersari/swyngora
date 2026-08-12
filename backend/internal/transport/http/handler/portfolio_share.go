package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type portfolioShareBody struct {
	ClientID        string `json:"clientId"`
	PortfolioID     string `json:"portfolioId"`
	GranteeClientID string `json:"granteeClientId"`
	Role            string `json:"role"`
}

type portfolioShareDTO struct {
	PortfolioID     string `json:"portfolioId"`
	OwnerClientID   string `json:"ownerClientId"`
	GranteeClientID string `json:"granteeClientId"`
	Role            string `json:"role"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func portfolioShareToDTO(sh *domain.PortfolioShare) portfolioShareDTO {
	return portfolioShareDTO{
		PortfolioID: sh.PortfolioID, OwnerClientID: sh.OwnerClientID, GranteeClientID: sh.GranteeClientID,
		Role: string(sh.Role),
		CreatedAt: sh.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: sh.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// SharePortfolio handles POST /api/v1/portfolio/shares
func (h *PortfolioHandler) SharePortfolio(w http.ResponseWriter, r *http.Request) {
	var body portfolioShareBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	owner, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	sh, err := h.svc.Share(r.Context(), owner, coalescePortfolioID(r, body.PortfolioID), body.GranteeClientID, body.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, portfolioShareToDTO(sh))
}

// UpdatePortfolioShare handles PATCH /api/v1/portfolio/shares
func (h *PortfolioHandler) UpdatePortfolioShare(w http.ResponseWriter, r *http.Request) {
	var body portfolioShareBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	owner, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	sh, err := h.svc.UpdateShareRole(r.Context(), owner, coalescePortfolioID(r, body.PortfolioID), body.GranteeClientID, body.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, portfolioShareToDTO(sh))
}

// ListPortfolioShares handles GET /api/v1/portfolio/shares
func (h *PortfolioHandler) ListPortfolioShares(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListShares(r.Context(), clientIDFrom(r), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]portfolioShareDTO, 0, len(list))
	for i := range list {
		items = append(items, portfolioShareToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ownerClientId": clientIDFrom(r), "shares": items, "count": len(items),
	})
}

// RevokePortfolioShare handles DELETE /api/v1/portfolio/shares
func (h *PortfolioHandler) RevokePortfolioShare(w http.ResponseWriter, r *http.Request) {
	grantee := strings.TrimSpace(r.URL.Query().Get("granteeClientId"))
	if err := h.svc.RevokeShare(r.Context(), clientIDFrom(r), portfolioIDFrom(r), grantee); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true, "granteeClientId": grantee})
}

// ListSharedPortfolios handles GET /api/v1/portfolios/shared
func (h *PortfolioHandler) ListSharedPortfolios(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListSharedWithMe(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, b := range list {
		p := b.Portfolio
		items = append(items, map[string]any{
			"id": p.ID, "clientId": p.ClientID, "name": p.Name, "role": string(b.Role),
			"currency": p.Currency, "startingBalance": p.StartingBalance, "cashBalance": p.CashBalance,
			"createdAt": p.CreatedAt.UTC().Format(time.RFC3339Nano),
			"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "portfolios": items, "count": len(items),
	})
}
