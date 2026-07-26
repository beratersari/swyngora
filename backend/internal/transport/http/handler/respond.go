package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DefaultMaxJSONBody is the max request body size for JSON POST/PUT handlers.
const DefaultMaxJSONBody = 1 << 20 // 1 MiB

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// decodeJSON reads at most maxBytes from r.Body into dst.
func decodeJSON(r *http.Request, dst any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxJSONBody
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	status, code, publicMsg := mapError(err)
	// Always log the full error server-side; never leak upstream details to clients.
	if status >= 500 || errors.Is(err, domain.ErrUpstream) || errors.Is(err, domain.ErrRateLimited) {
		slog.Error("request error", "status", status, "code", code, "err", err)
	} else {
		slog.Debug("request error", "status", status, "code", code, "err", err)
	}
	writeJSON(w, status, errorBody{Error: errorDetail{
		Code:    code,
		Message: publicMsg,
	}})
}

func mapError(err error) (status int, code, message string) {
	switch {
	case err == nil:
		return http.StatusInternalServerError, "internal_error", "internal error"
	case errors.Is(err, context.Canceled):
		// Client went away — not a server fault. 499 is non-standard; 400 is safe.
		return http.StatusBadRequest, "canceled", "request canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "timeout", "request timed out"
	case errors.Is(err, domain.ErrInvalidArgument):
		return http.StatusBadRequest, "invalid_argument", publicInvalidArgument(err)
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found", "resource not found"
	case errors.Is(err, domain.ErrRateLimited):
		return http.StatusTooManyRequests, "rate_limited", "rate limited; try again later"
	case errors.Is(err, domain.ErrUpstream):
		return http.StatusBadGateway, "upstream_error", "upstream data source unavailable"
	default:
		return http.StatusInternalServerError, "internal_error", "internal error"
	}
}

// publicInvalidArgument returns a short client-safe validation message.
// Domain validation messages are intentionally simple and safe to surface.
func publicInvalidArgument(err error) string {
	msg := err.Error()
	// Strip the sentinel prefix "invalid argument: " when present.
	const prefix = "invalid argument: "
	if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
		return msg[len(prefix):]
	}
	if msg == "invalid argument" {
		return "invalid argument"
	}
	// If wrapping preserved a useful suffix via %w, still avoid leaking nested upstream text.
	return "invalid argument"
}
