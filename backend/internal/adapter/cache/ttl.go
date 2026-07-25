package cache

import (
	"sync"
	"time"
)

// Entry is a cached value with an absolute expiry.
// Zero ExpiresAt means the entry never expires (until deleted or ReplaceAll).
type Entry[T any] struct {
	Value     T
	ExpiresAt time.Time
}

// TTL is a small in-memory TTL cache for background/API hygiene.
// It is safe for concurrent use. Expired entries miss on Get but remain
// available via GetStale until Cleanup (last-good fail-soft paths).
// Optional MaxEntries bounds memory.
type TTL[T any] struct {
	mu         sync.RWMutex
	items      map[string]Entry[T]
	defaultTTL time.Duration
	maxEntries int // 0 = unlimited
	now        func() time.Time
}

// Options configures an optional capacity bound for NewWithOptions.
type Options struct {
	// MaxEntries caps the number of keys. 0 means unlimited.
	// When exceeded after Set, expired entries are purged first; if still over,
	// arbitrary extra keys are dropped (not strict LRU).
	MaxEntries int
}

// New creates a TTL cache with the given default lifetime (unlimited size).
func New[T any](defaultTTL time.Duration) *TTL[T] {
	return NewWithOptions[T](defaultTTL, Options{})
}

// NewWithOptions creates a TTL cache with optional capacity bounds.
func NewWithOptions[T any](defaultTTL time.Duration, opts Options) *TTL[T] {
	return &TTL[T]{
		items:      make(map[string]Entry[T]),
		defaultTTL: defaultTTL,
		maxEntries: opts.MaxEntries,
		now:        time.Now,
	}
}

func (c *TTL[T]) expired(e Entry[T], now time.Time) bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return !e.ExpiresAt.After(now)
}

// Get returns the value if present and not expired.
// Expired entries are left in place for GetStale / Cleanup (no TOCTOU delete).
func (c *TTL[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || c.expired(e, c.now()) {
		var zero T
		return zero, false
	}
	return e.Value, true
}

// GetStale returns a value even if expired (for fail-soft last-good paths).
// ok is false only when the key is absent.
func (c *TTL[T]) GetStale(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok {
		var zero T
		return zero, false
	}
	return e.Value, true
}

// Set stores value with the default TTL.
// If defaultTTL <= 0, the entry never expires.
func (c *TTL[T]) Set(key string, value T) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores value with an explicit TTL.
// If ttl <= 0, the entry never expires (until Delete or ReplaceAll).
func (c *TTL[T]) SetWithTTL(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = c.now().Add(ttl)
	}
	c.items[key] = Entry[T]{
		Value:     value,
		ExpiresAt: exp,
	}
	c.evictLocked()
}

// ReplaceAll atomically replaces all entries with the provided map.
// Each entry uses the default TTL (or never-expires when defaultTTL <= 0).
func (c *TTL[T]) ReplaceAll(values map[string]T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]Entry[T], len(values))
	var exp time.Time
	if c.defaultTTL > 0 {
		exp = c.now().Add(c.defaultTTL)
	}
	for k, v := range values {
		c.items[k] = Entry[T]{Value: v, ExpiresAt: exp}
	}
	c.evictLocked()
}

// Delete removes a key.
func (c *TTL[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Cleanup removes all expired entries. Safe to call from a background goroutine.
// Never-expiring entries (zero ExpiresAt) are retained.
func (c *TTL[T]) Cleanup() int {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for k, e := range c.items {
		if c.expired(e, now) {
			delete(c.items, k)
			removed++
		}
	}
	return removed
}

// Len returns the number of entries (including not-yet-evicted expired ones until Cleanup).
func (c *TTL[T]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// evictLocked enforces MaxEntries. Caller must hold write lock.
func (c *TTL[T]) evictLocked() {
	if c.maxEntries <= 0 || len(c.items) <= c.maxEntries {
		return
	}
	now := c.now()
	for k, e := range c.items {
		if c.expired(e, now) {
			delete(c.items, k)
		}
	}
	for len(c.items) > c.maxEntries {
		for k := range c.items {
			delete(c.items, k)
			break
		}
	}
}
