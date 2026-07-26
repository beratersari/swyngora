package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// defaultMaxBuckets caps distinct IP buckets to bound memory under unique-IP floods.
const defaultMaxBuckets = 100_000

// RateLimit returns middleware that limits requests per client IP using a token bucket.
// rps is tokens per second; burst is the maximum burst size.
// When rps <= 0, the middleware is a no-op.
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	return RateLimitWithMaxBuckets(rps, burst, defaultMaxBuckets)
}

// RateLimitWithMaxBuckets is like RateLimit but allows tests to set a small bucket cap.
func RateLimitWithMaxBuckets(rps float64, burst, maxBuckets int) func(http.Handler) http.Handler {
	if rps <= 0 || burst <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if maxBuckets <= 0 {
		maxBuckets = defaultMaxBuckets
	}
	lim := &ipLimiter{
		rps:        rps,
		burst:      float64(burst),
		buckets:    make(map[string]*bucket),
		maxBuckets: maxBuckets,
		idleAfter:  10 * time.Minute,
	}
	// Periodic cleanup of idle buckets.
	go lim.cleanupLoop()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !lim.allow(ip) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "rate_limited",
						"message": "too many requests",
					},
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type bucket struct {
	tokens float64
	last   time.Time
}

type ipLimiter struct {
	mu         sync.Mutex
	rps        float64
	burst      float64
	buckets    map[string]*bucket
	maxBuckets int
	idleAfter  time.Duration
}

func (l *ipLimiter) allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[ip]
	if !ok {
		if len(l.buckets) >= l.maxBuckets {
			l.evictIdleLocked(now)
		}
		if len(l.buckets) >= l.maxBuckets {
			// Still full: refuse new IPs rather than grow unbounded.
			return false
		}
		l.buckets[ip] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rps
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *ipLimiter) evictIdleLocked(now time.Time) {
	cutoff := now.Add(-l.idleAfter)
	for ip, b := range l.buckets {
		if b.last.Before(cutoff) {
			delete(l.buckets, ip)
		}
	}
}

func (l *ipLimiter) cleanupLoop() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		l.mu.Lock()
		l.evictIdleLocked(time.Now())
		l.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	// Prefer direct remote address; X-Forwarded-For is not trusted by default
	// (would allow spoofing to bypass limits). Reverse proxies should terminate
	// and set RemoteAddr correctly or we can add trusted-proxy support later.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
