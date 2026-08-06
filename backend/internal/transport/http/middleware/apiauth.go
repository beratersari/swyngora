package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// APIAuth requires a shared secret when token is non-empty.
// Accepts Authorization: Bearer <token> or X-API-Key: <token>.
// When token is empty, the middleware is a no-op (local/dev default).
// Public paths (health + market data GETs) stay open so charts keep working.
func APIAuth(token string) func(http.Handler) http.Handler {
	token = strings.TrimSpace(token)
	if token == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	want := []byte(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			if isPublicAPIPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			got := extractAPIToken(r)
			if subtle.ConstantTimeCompare([]byte(got), want) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="swyngora"`)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				// Same nested envelope as handler.writeError / OpenAPI Error schema.
				_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"valid API token required"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
