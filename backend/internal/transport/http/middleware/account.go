package middleware

import (
	"net/http"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
)

// AccountGate blocks closed clientIds from user-scoped API routes.
// Allows POST /api/v1/account/reopen and GET /api/v1/account (status) so users can reopen.
// Market routes without a clientId are unaffected.
func AccountGate(svc *account.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if svc == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			// Always allow health, market public data, reopen + status.
			// /mcp is skipped here: tools send clientId in the JSON body, not headers.
			// In-process MCP enforces RequireActive when clientId is present (see transport/mcp).
			if path == "/health" || strings.HasPrefix(path, "/api/v1/market/") ||
				path == "/mcp" || strings.HasPrefix(path, "/mcp/") {
				next.ServeHTTP(w, r)
				return
			}
			// Account lifecycle endpoints (status / close / reopen) always allowed.
			if path == "/api/v1/account" || path == "/api/v1/account/close" || path == "/api/v1/account/reopen" {
				next.ServeHTTP(w, r)
				return
			}

			clientID := strings.TrimSpace(r.Header.Get("X-Client-Id"))
			if clientID == "" {
				clientID = strings.TrimSpace(r.URL.Query().Get("clientId"))
			}
			// Body clientId is not available here without buffering; handlers still check via service if needed.
			if clientID != "" {
				if err := svc.RequireActive(r.Context(), clientID); err != nil {
					// Defer to handler error mapping via a simple JSON response
					writeAccountClosed(w, err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeAccountClosed(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	msg := err.Error()
	// strip "forbidden: " prefix for message
	if strings.HasPrefix(msg, "forbidden: ") {
		msg = strings.TrimPrefix(msg, "forbidden: ")
	}
	_, _ = w.Write([]byte(`{"error":{"code":"account_closed","message":` + jsonQuote(msg) + `}}`))
}

func jsonQuote(s string) string {
	// minimal JSON string escape
	b := strings.Builder{}
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
