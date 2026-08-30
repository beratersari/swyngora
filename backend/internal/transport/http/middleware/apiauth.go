package middleware

import (
	"context"
	"crypto/subtle"
	"net"
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
	ClientID string
	CanTrade bool
	KeyID    string // empty = master token, open mode, or ticket without a user key
	UserKey  bool
	Master   bool // authenticated with the process master token
	Loopback bool // RemoteAddr is loopback (local Vite, AI HTTP tools)
	// DenyImpersonate is copied from process config: remote master must not
	// select an arbitrary clientId.
	DenyImpersonate bool
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

// APIAuthOptions configures APIAuthWithOptions.
type APIAuthOptions struct {
	Master                string
	Keys                  APIKeyLookup
	Tickets               *WSTicketIssuer
	DenyMasterImpersonate bool
}

// APIAuth requires a shared secret when token is non-empty (legacy single-token mode).
func APIAuth(token string) func(http.Handler) http.Handler {
	return APIAuthWith(token, nil)
}

// APIAuthWith accepts the process master token and/or per-user API keys.
// Master token = full access (any clientId via X-Client-Id) only from loopback
// unless AllowMasterImpersonate is set on the router.
// User key = one clientId + read or trade permission.
// Empty master + no user key = open local/dev mode.
func APIAuthWith(master string, keys APIKeyLookup) func(http.Handler) http.Handler {
	return APIAuthWithOptions(APIAuthOptions{Master: master, Keys: keys})
}

// APIAuthWithOptions is APIAuthWith plus WS tickets and master-impersonation policy.
func APIAuthWithOptions(opts APIAuthOptions) func(http.Handler) http.Handler {
	master := strings.TrimSpace(opts.Master)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions || isPublicAPIPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if opts.Tickets != nil && isRealtimeWSPath(r.URL.Path) {
				if ticket := strings.TrimSpace(r.URL.Query().Get("ticket")); ticket != "" {
					id, err := opts.Tickets.Consume(ticket)
					if err != nil || id == nil {
						writeAPIUnauthorized(w)
						return
					}
					r = r.WithContext(WithIdentity(r.Context(), id))
					if id.ClientID != "" {
						r.Header.Set("X-Client-Id", id.ClientID)
					}
					next.ServeHTTP(w, r)
					return
				}
			}
			got := extractAPIToken(r)
			if master != "" && got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(master)) == 1 {
				id := &AuthIdentity{
					Master: true, Loopback: isLoopbackRemote(r),
					DenyImpersonate: opts.DenyMasterImpersonate,
				}
				r = r.WithContext(WithIdentity(r.Context(), id))
				next.ServeHTTP(w, r)
				return
			}
			if opts.Keys != nil && domain.LooksLikeUserAPIKey(got) {
				k, err := opts.Keys.Authenticate(r.Context(), got)
				if err == nil && k != nil && !k.IsRevoked() {
					id := &AuthIdentity{
						ClientID: k.ClientID, CanTrade: k.CanTrade(), KeyID: k.ID, UserKey: true,
						Loopback: isLoopbackRemote(r),
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
		// Query-string secrets are rejected on every route (they land in
		// access logs, proxies, and history). Browsers mint a one-time
		// ?ticket= via POST /api/v1/realtime/ticket instead.
		return ""
	}
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return strings.TrimSpace(auth[len(prefix):])
	}
	return ""
}

func isRealtimeWSPath(path string) bool {
	return path == "/api/v1/ws" || strings.HasPrefix(path, "/api/v1/ws/")
}

// isLoopbackRemote is true when the peer is loopback (local Vite proxy, AI tools).
func isLoopbackRemote(r *http.Request) bool {
	if r == nil {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
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
	// Protocol description only (the WebSocket itself still requires auth when configured).
	if path == "/api/v1/realtime" || path == "/api/v1/realtime/" {
		return true
	}
	return false
}
