package middleware

import (
	"net/http"
	"strings"
)

// CORS enables browser access. When allowedOrigins is empty or contains "*",
// Access-Control-Allow-Origin is "*". Otherwise only listed origins are echoed.
// Restrict origins in production via CORS_ALLOW_ORIGINS.
func CORS(next http.Handler) http.Handler {
	return CORSWithOrigins(nil)(next)
}

// CORSWithOrigins returns CORS middleware with an explicit origin allowlist.
// Empty list or a single "*" entry allows any origin (dev default).
func CORSWithOrigins(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := len(allowedOrigins) == 0
	set := map[string]struct{}{}
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			break
		}
		set[o] = struct{}{}
	}
	if allowAll {
		set = nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				if _, ok := set[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, X-Client-Id, Authorization, X-API-Key")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
