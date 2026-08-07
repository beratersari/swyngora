package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestWriteError_Mapping(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		code    string
		msgPart string
	}{
		{fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument), http.StatusBadRequest, "invalid_argument", "symbol"},
		{fmt.Errorf("%w: x", domain.ErrNotFound), http.StatusNotFound, "not_found", "not found"},
		{fmt.Errorf("%w: binance 429", domain.ErrRateLimited), http.StatusTooManyRequests, "rate_limited", "try again"},
		{fmt.Errorf("%w: request failed: dial tcp secret", domain.ErrUpstream), http.StatusBadGateway, "upstream_error", "unavailable"},
		{fmt.Errorf("%w: AI service unreachable at http://127.0.0.1:8090", domain.ErrUpstream), http.StatusBadGateway, "ai_unavailable", "Ollama"},
		{errors.New("boom internal stack"), http.StatusInternalServerError, "internal_error", "internal error"},
		{context.Canceled, http.StatusBadRequest, "canceled", "canceled"},
		{context.DeadlineExceeded, http.StatusGatewayTimeout, "timeout", "timed out"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		writeError(rr, tc.err)
		if rr.Code != tc.status {
			t.Fatalf("%v: status %d want %d", tc.err, rr.Code, tc.status)
		}
		var body errorBody
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Error.Code != tc.code {
			t.Fatalf("code %s want %s", body.Error.Code, tc.code)
		}
		if !strings.Contains(strings.ToLower(body.Error.Message), strings.ToLower(tc.msgPart)) {
			t.Fatalf("message %q should contain %q", body.Error.Message, tc.msgPart)
		}
		// Upstream secrets must not leak.
		if strings.Contains(body.Error.Message, "secret") || strings.Contains(body.Error.Message, "dial tcp") {
			t.Fatalf("leaked upstream detail: %q", body.Error.Message)
		}
	}
}

func TestDecodeJSON_MaxBytes(t *testing.T) {
	var dst map[string]any
	// Small valid body
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"a":1}`)))
	if err := decodeJSON(req, &dst, 1024); err != nil {
		t.Fatal(err)
	}
	// Oversized body
	big := bytes.Repeat([]byte("a"), 200)
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(append([]byte(`{"x":"`), append(big, []byte(`"}`)...)...)))
	err := decodeJSON(req, &dst, 50)
	if err == nil {
		t.Fatal("expected max bytes error")
	}
}
