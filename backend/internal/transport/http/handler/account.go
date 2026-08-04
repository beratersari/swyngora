package handler

import (
	"fmt"
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
)

// AccountHandler handles account close/reopen/status.
type AccountHandler struct {
	svc *account.Service
}

// NewAccountHandler constructs the handler.
func NewAccountHandler(svc *account.Service) *AccountHandler {
	return &AccountHandler{svc: svc}
}

type accountDTO struct {
	ClientID      string  `json:"clientId"`
	Status        string  `json:"status"`
	ClosedAt      *string `json:"closedAt,omitempty"`
	PurgeAt       *string `json:"purgeAt,omitempty"`
	ReopenedAt    *string `json:"reopenedAt,omitempty"`
	CanReopen     bool    `json:"canReopen"`
	GraceDays     int     `json:"graceDays"`
	CreatedAt     string  `json:"createdAt,omitempty"`
	UpdatedAt     string  `json:"updatedAt,omitempty"`
}

func accountToDTO(a *domain.Account) accountDTO {
	d := accountDTO{
		ClientID: a.ClientID, Status: string(a.Status),
		CanReopen: a.CanReopen(time.Now().UTC()),
		GraceDays: int(domain.AccountCloseGrace / (24 * time.Hour)),
	}
	if !a.CreatedAt.IsZero() {
		d.CreatedAt = a.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if !a.UpdatedAt.IsZero() {
		d.UpdatedAt = a.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	if a.ClosedAt != nil {
		s := a.ClosedAt.UTC().Format(time.RFC3339Nano)
		d.ClosedAt = &s
	}
	if a.PurgeAt != nil {
		s := a.PurgeAt.UTC().Format(time.RFC3339Nano)
		d.PurgeAt = &s
	}
	if a.ReopenedAt != nil {
		s := a.ReopenedAt.UTC().Format(time.RFC3339Nano)
		d.ReopenedAt = &s
	}
	return d
}

type accountBody struct {
	ClientID string `json:"clientId"`
}

// Status handles GET /api/v1/account
func (h *AccountHandler) Status(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Status(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountToDTO(a))
}

// Close handles POST /api/v1/account/close
func (h *AccountHandler) Close(w http.ResponseWriter, r *http.Request) {
	clientID := clientIDFrom(r)
	if r.Body != nil && r.ContentLength != 0 {
		var body accountBody
		if err := decodeJSON(r, &body, DefaultMaxJSONBody); err == nil && body.ClientID != "" {
			clientID = body.ClientID
		}
	}
	if clientID == "" {
		writeError(w, fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument))
		return
	}
	a, err := h.svc.Close(r.Context(), clientID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountToDTO(a))
}

// Reopen handles POST /api/v1/account/reopen
func (h *AccountHandler) Reopen(w http.ResponseWriter, r *http.Request) {
	clientID := clientIDFrom(r)
	if r.Body != nil && r.ContentLength != 0 {
		var body accountBody
		if err := decodeJSON(r, &body, DefaultMaxJSONBody); err == nil && body.ClientID != "" {
			clientID = body.ClientID
		}
	}
	if clientID == "" {
		writeError(w, fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument))
		return
	}
	a, err := h.svc.Reopen(r.Context(), clientID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountToDTO(a))
}
