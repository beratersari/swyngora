package middleware

import (
	"context"
	"net/http"
	"strings"
)

// APIKeyCounter reports how many active named keys a tenant has (bootstrap).
type APIKeyCounter interface {
	CountActive(ctx context.Context, clientID string) (int, error)
}

// MasterTenant blocks a remote process-master token from acting as an arbitrary
// X-Client-Id. Loopback (local Vite, in-process AI HTTP tools) is allowed.
// User keys are already bound and skip this gate.
//
// Exception: POST /api/v1/account/api-keys when that clientId has zero keys
// (first-key bootstrap). Listing/revoking keys and all other tenant routes
// stay forbidden for remote master.
func MasterTenant(strict bool, keys APIKeyCounter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strict {
				next.ServeHTTP(w, r)
				return
			}
			id := IdentityFrom(r.Context())
			if id == nil || id.UserKey || !id.Master || id.Loopback || !id.DenyImpersonate {
				next.ServeHTTP(w, r)
				return
			}
			path := r.URL.Path
			if path == "/mcp" || strings.HasPrefix(path, "/mcp/") {
				// Tool layer enforces clientId (bindMCPTenant).
				next.ServeHTTP(w, r)
				return
			}
			if r.Method == http.MethodPost && strings.HasPrefix(path, "/api/v1/account/api-keys") &&
				!strings.Contains(strings.TrimPrefix(path, "/api/v1/account/api-keys"), "/") {
				clientID := strings.TrimSpace(r.Header.Get("X-Client-Id"))
				if clientID == "" {
					clientID = strings.TrimSpace(r.URL.Query().Get("clientId"))
				}
				if clientID == "" {
					_, _, body := peekTenantIDs(r)
					clientID = body
				}
				if keys != nil && clientID != "" {
					n, err := keys.CountActive(r.Context(), clientID)
					if err == nil && n == 0 {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			writeAPIForbidden(w, "master token cannot select a tenant from this client")
		})
	}
}
