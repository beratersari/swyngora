package binance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	binanceMaxInFlight = 2
	binanceMinGap      = 40 * time.Millisecond
	// Stay under the futures 2,400 / minute cap and leave room under spot 6,000.
	binanceWeightPauseAt = 1800
	binanceWeightPause   = 15 * time.Second
	binanceLimitPause    = 60 * time.Second
	binanceBanPause      = 3 * time.Minute
)

// restGate serializes Binance REST and stops the process from calling again
// after a 429, a 418 IP ban, or a high used-weight header.
type restGate struct {
	sem chan struct{}

	mu         sync.Mutex
	pauseUntil time.Time
	nextStart  time.Time
	lastLog    time.Time
	lastWhy    string
}

func newRestGate() *restGate {
	return &restGate{sem: make(chan struct{}, binanceMaxInFlight)}
}

func restCallErr(err error) error {
	if errors.Is(err, domain.ErrRateLimited) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: request failed: %v", domain.ErrUpstream, err)
}

func (g *restGate) begin(ctx context.Context) error {
	if g == nil || g.sem == nil {
		return nil
	}
	if err := g.refuseIfPaused(); err != nil {
		return err
	}
	g.mu.Lock()
	wait := time.Until(g.nextStart)
	g.mu.Unlock()
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case g.sem <- struct{}{}:
	}
	if err := g.refuseIfPaused(); err != nil {
		<-g.sem
		return err
	}
	g.mu.Lock()
	g.nextStart = time.Now().Add(binanceMinGap)
	g.mu.Unlock()
	return nil
}

func (g *restGate) finish(resp *http.Response) {
	if g == nil || g.sem == nil {
		return
	}
	<-g.sem
	if resp == nil {
		return
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		g.pause(retryAfter(resp, binanceLimitPause), "429")
	case http.StatusTeapot:
		g.pause(retryAfter(resp, binanceBanPause), "418")
	default:
		if w := usedWeight(resp); w >= binanceWeightPauseAt {
			g.pause(binanceWeightPause, "weight "+strconv.Itoa(w))
		}
	}
}

func (g *restGate) refuseIfPaused() error {
	g.mu.Lock()
	until := g.pauseUntil
	g.mu.Unlock()
	if time.Now().Before(until) {
		return fmt.Errorf("%w: binance rest paused until %s", domain.ErrRateLimited, until.UTC().Format(time.RFC3339))
	}
	return nil
}

func (g *restGate) paused() bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return time.Now().Before(g.pauseUntil)
}

func (g *restGate) pause(d time.Duration, why string) {
	if d <= 0 {
		return
	}
	until := time.Now().Add(d)
	g.mu.Lock()
	if !until.After(g.pauseUntil) {
		g.mu.Unlock()
		return
	}
	g.pauseUntil = until
	logNow := g.lastWhy != why || time.Since(g.lastLog) > 30*time.Second
	if logNow {
		g.lastLog = time.Now()
		g.lastWhy = why
	}
	g.mu.Unlock()
	if logNow {
		slog.Warn("binance rest paused", "until", until.UTC().Format(time.RFC3339), "reason", why, "pause", d.String())
	}
}

func retryAfter(resp *http.Response, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if raw == "" {
		return fallback
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs <= 0 {
		return fallback
	}
	d := time.Duration(secs) * time.Second
	if d < fallback {
		return fallback
	}
	return d
}

func usedWeight(resp *http.Response) int {
	best := 0
	for key, vals := range resp.Header {
		if !strings.HasPrefix(strings.ToUpper(key), "X-MBX-USED-WEIGHT") || len(vals) == 0 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(vals[0]))
		if err == nil && n > best {
			best = n
		}
	}
	return best
}
