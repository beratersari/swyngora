package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestWriteError_Mapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{fmt.Errorf("%w: x", domain.ErrInvalidArgument), http.StatusBadRequest, "invalid_argument"},
		{fmt.Errorf("%w: x", domain.ErrNotFound), http.StatusNotFound, "not_found"},
		{fmt.Errorf("%w: x", domain.ErrRateLimited), http.StatusTooManyRequests, "rate_limited"},
		{fmt.Errorf("%w: x", domain.ErrUpstream), http.StatusBadGateway, "upstream_error"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
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
	}
}
