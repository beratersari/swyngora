package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
)

const maxAccountGatePeek = 1 << 20 // 1 MiB

// AccountGate blocks closed clientIds from user-scoped API routes.
// Allows POST /api/v1/account/reopen and GET /api/v1/account (status) so users can reopen.
// Market routes without a clientId are unaffected.
func AccountGate(svc *account.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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

			header, query, body := peekTenantIDs(r)
			agreed, err := AgreeClientIDs(header, query, body)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_argument", "clientId header, query, and body must match")
				return
			}
			if svc == nil {
				next.ServeHTTP(w, r)
				return
			}
			clientID := agreed
			if id := IdentityFrom(r.Context()); id != nil && id.UserKey && id.ClientID != "" {
				if agreed != "" && agreed != id.ClientID {
					writeJSONError(w, http.StatusForbidden, "forbidden", "clientId does not match API key binding")
					return
				}
				clientID = id.ClientID
			}
			if clientID == "" {
				writeJSONError(w, http.StatusBadRequest, "invalid_argument", "clientId is required")
				return
			}
			if err := svc.RequireActive(r.Context(), clientID); err != nil {
				writeAccountClosed(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AgreeClientIDs returns the single tenant id when every non-empty value matches.
func AgreeClientIDs(ids ...string) (string, error) {
	seen := ""
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if seen == "" {
			seen = id
			continue
		}
		if id != seen {
			return "", fmt.Errorf("%w: clientId header, query, and body must match", domain.ErrInvalidArgument)
		}
	}
	return seen, nil
}

// peekTenantIDs reads header, query, and JSON body clientId (restoring the body).
func peekTenantIDs(r *http.Request) (header, query, body string) {
	header = strings.TrimSpace(r.Header.Get("X-Client-Id"))
	query = strings.TrimSpace(r.URL.Query().Get("clientId"))
	body = peekJSONClientID(r)
	return
}

func peekJSONClientID(r *http.Request) string {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if r.Body == nil || r.ContentLength == 0 {
		return ""
	}
	if !strings.Contains(ct, "application/json") && ct != "" {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxAccountGatePeek+1))
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if err != nil || len(raw) == 0 || len(raw) > maxAccountGatePeek {
		return ""
	}
	var payload struct {
		ClientID string `json:"clientId"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.ClientID)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":` + jsonQuote(code) + `,"message":` + jsonQuote(message) + `}}`))
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
