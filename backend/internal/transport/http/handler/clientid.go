package handler

import (
	"fmt"
	"net/http"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

// resolveClientID picks the actor client id for a request.
// User-issued API keys always bind to the key's clientId; a mismatched body/form
// clientId is rejected (prevents cross-tenant IDOR via JSON fields).
// Master token and open mode may still use bodyClientID or headers/query.
func resolveClientID(r *http.Request, bodyClientID string) (string, error) {
	bodyClientID = strings.TrimSpace(bodyClientID)
	if id := middleware.IdentityFrom(r.Context()); id != nil && id.UserKey {
		if id.ClientID == "" {
			return "", fmt.Errorf("%w: API key has no client binding", domain.ErrForbidden)
		}
		if bodyClientID != "" && bodyClientID != id.ClientID {
			return "", fmt.Errorf("%w: clientId does not match API key binding", domain.ErrForbidden)
		}
		return id.ClientID, nil
	}
	header := strings.TrimSpace(r.Header.Get("X-Client-Id"))
	query := strings.TrimSpace(r.URL.Query().Get("clientId"))
	if _, err := middleware.AgreeClientIDs(header, query, bodyClientID); err != nil {
		return "", err
	}
	if bodyClientID != "" {
		return bodyClientID, nil
	}
	return clientIDFrom(r), nil
}

// mustResolveClientID is resolveClientID with writeError on failure.
// Returns false when the response was already written.
func mustResolveClientID(w http.ResponseWriter, r *http.Request, bodyClientID string) (string, bool) {
	id, err := resolveClientID(r, bodyClientID)
	if err != nil {
		writeError(w, err)
		return "", false
	}
	return id, true
}
