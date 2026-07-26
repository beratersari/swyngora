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

func TestRateLimit_MaxBucketsCapsNewIPs(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := RateLimitWithMaxBuckets(100, 10, 2)(next)

	// Fill capacity with two IPs.
	for i, ip := range []string{"10.0.0.1:1", "10.0.0.2:1"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = ip
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("ip %d: want 200 got %d", i, rr.Code)
		}
	}
	// Third distinct IP must be refused (map full).
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "10.0.0.3:1"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("third IP: want 429 got %d", rr.Code)
	}
	// Existing IP still allowed.
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	req2.RemoteAddr = "10.0.0.1:1"
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("existing IP blocked: %d", rr2.Code)
	}
}
