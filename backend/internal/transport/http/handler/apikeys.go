package handler

import (
	"fmt"
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
)

// APIKeyHandler is the HTTP adapter for user API keys.
type APIKeyHandler struct {
	svc *apikey.Service
}

// NewAPIKeyHandler constructs the handler.
func NewAPIKeyHandler(svc *apikey.Service) *APIKeyHandler {
	return &APIKeyHandler{svc: svc}
}

type apiKeyDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Prefix      string  `json:"prefix"`
	Permission  string  `json:"permission"`
	CreatedAt   string  `json:"createdAt"`
	LastUsedAt  *string `json:"lastUsedAt,omitempty"`
	RevokedAt   *string `json:"revokedAt,omitempty"`
	Revoked     bool    `json:"revoked"`
	Secret      string  `json:"secret,omitempty"`
}

func apiKeyToDTO(k *domain.APIKey, secret string) apiKeyDTO {
	d := apiKeyDTO{
		ID: k.ID, Name: k.Name, Prefix: k.Prefix, Permission: string(k.Permission),
		CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339Nano),
		Revoked:   k.IsRevoked(), Secret: secret,
	}
	if k.LastUsedAt != nil {
		s := k.LastUsedAt.UTC().Format(time.RFC3339Nano)
		d.LastUsedAt = &s
	}
	if k.RevokedAt != nil {
		s := k.RevokedAt.UTC().Format(time.RFC3339Nano)
		d.RevokedAt = &s
	}
	return d
}

type createAPIKeyBody struct {
	ClientID   string `json:"clientId"`
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

// Create handles POST /api/v1/account/api-keys
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createAPIKeyBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	created, err := h.svc.Create(r.Context(), apikey.CreateInput{
		ClientID: clientID, Name: body.Name, Permission: body.Permission,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, apiKeyToDTO(created.Key, created.Secret))
}

// List handles GET /api/v1/account/api-keys
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]apiKeyDTO, 0, len(list))
	for i := range list {
		items = append(items, apiKeyToDTO(&list[i], ""))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r),
		"keys":     items,
		"count":    len(items),
	})
}

// Revoke handles DELETE /api/v1/account/api-keys/{id}
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	k, err := h.svc.Revoke(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apiKeyToDTO(k, ""))
}
