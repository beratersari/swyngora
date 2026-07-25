package handler

import (
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	status, code := mapError(err)
	writeJSON(w, status, errorBody{Error: errorDetail{
		Code:    code,
		Message: err.Error(),
	}})
}

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		return http.StatusBadRequest, "invalid_argument"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrRateLimited):
		return http.StatusTooManyRequests, "rate_limited"
	case errors.Is(err, domain.ErrUpstream):
		return http.StatusBadGateway, "upstream_error"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}
