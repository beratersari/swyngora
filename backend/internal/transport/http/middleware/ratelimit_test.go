package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimit_BlocksBurst(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := RateLimit(1, 2)(next)

	ok := 0
	blocked := 0
	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusOK {
			ok++
		}
		if rr.Code == http.StatusTooManyRequests {
			blocked++
		}
	}
	if ok != 2 || blocked != 3 {
		t.Fatalf("ok=%d blocked=%d", ok, blocked)
	}
}

func TestRateLimit_Disabled(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := RateLimit(0, 0)(next)
	for i := 0; i < 20; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("disabled limiter blocked: %d", rr.Code)
		}
	}
}
