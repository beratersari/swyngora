package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// APIKeyLookup authenticates a user-issued API key secret.
type APIKeyLookup interface {
	Authenticate(ctx context.Context, secret string) (*domain.APIKey, error)
}

type identityCtxKey struct{}

// AuthIdentity is request-scoped auth after APIAuth.
type AuthIdentity struct {
	ClientID  string
	CanTrade  bool
	KeyID     string // empty = master token or open mode
	UserKey   bool
}

// IdentityFrom returns user-key identity when a scoped key authenticated the request.
func IdentityFrom(ctx context.Context) *AuthIdentity {
	v, _ := ctx.Value(identityCtxKey{}).(*AuthIdentity)
	return v
}

// WithIdentity stores identity on the context.
func WithIdentity(ctx context.Context, id *AuthIdentity) context.Context {
	if id == nil {
		return ctx
	}
	return context.WithValue(ctx, identityCtxKey{}, id)
}

// APIAuth requires a shared secret when token is non-empty (legacy single-token mode).
func APIAuth(token string) func(http.Handler) http.Handler {
	return APIAuthWith(token, nil)
}

// APIAuthWith accepts the process master token and/or per-user API keys.
// Master token = full access (any clientId via X-Client-Id).
// User key = one clientId + read or trade permission.
// Empty master + no user key = open local/dev mode.
func APIAuthWith(master string, keys APIKeyLookup) func(http.Handler) http.Handler {
	master = strings.TrimSpace(master)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions || isPublicAPIPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			got := extractAPIToken(r)
			if master != "" && got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(master)) == 1 {
				next.ServeHTTP(w, r)
				return
			}
			if keys != nil && domain.LooksLikeUserAPIKey(got) {
				k, err := keys.Authenticate(r.Context(), got)
				if err == nil && k != nil && !k.IsRevoked() {
					id := &AuthIdentity{
						ClientID: k.ClientID, CanTrade: k.CanTrade(), KeyID: k.ID, UserKey: true,
					}
					r = r.WithContext(WithIdentity(r.Context(), id))
					r.Header.Set("X-Client-Id", k.ClientID)
					next.ServeHTTP(w, r)
					return
				}
				writeAPIUnauthorized(w)
				return
			}
			if master == "" {
				// Open local mode: no token, or a non-user-key header, is allowed.
				next.ServeHTTP(w, r)
				return
			}
			writeAPIUnauthorized(w)
		})
	}
}

func writeAPIUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="swyngora"`)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	// Same nested envelope as handler.writeError / OpenAPI Error schema.
	_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"valid API token required"}}`))
}

// APIKeyScope enforces read vs trade on user-issued keys. Master/open identities skip.
func APIKeyScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := IdentityFrom(r.Context())
		if id == nil || !id.UserKey {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		if isAPIKeyAdminPath(path) || isAccountAdminPath(path) {
			writeAPIForbidden(w, "this API key cannot manage account or other keys")
			return
		}
		if !id.CanTrade && requiresTradePermission(r.Method, path) {
			writeAPIForbidden(w, "this API key is read-only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAPIKeyAdminPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/account/api-keys")
}

func isAccountAdminPath(path string) bool {
	return path == "/api/v1/account/close" || path == "/api/v1/account/reopen"
}

func requiresTradePermission(method, path string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	if strings.HasPrefix(path, "/api/v1/ai/") {
		return false
	}
	if path == "/mcp" || strings.HasPrefix(path, "/mcp/") {
		return true
	}
	return true
}

func writeAPIForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":{"code":"forbidden","message":` + jsonQuote(msg) + `}}`))
}

func extractAPIToken(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-API-Key")); v != "" {
		return v
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return strings.TrimSpace(auth[len(prefix):])
	}
	return ""
}

// isPublicAPIPath allows unauthenticated access to liveness and public market data.
func isPublicAPIPath(path string) bool {
	if path == "/health" || path == "/health/" {
		return true
	}
	// Market data is intentionally public (no tenant state).
	if strings.HasPrefix(path, "/api/v1/market/") {
		return true
	}
	return false
}
