package cache

import (
	"sync"
	"time"
)

// Entry is a cached value with an absolute expiry.
type Entry[T any] struct {
	Value     T
	ExpiresAt time.Time
}

// TTL is a small in-memory TTL cache for background/API hygiene.
// It is safe for concurrent use. Expired entries are removed on Get
// and periodically via Cleanup.
type TTL[T any] struct {
	mu      sync.RWMutex
	items   map[string]Entry[T]
	defaultTTL time.Duration
	now     func() time.Time
}

// New creates a TTL cache with the given default lifetime.
func New[T any](defaultTTL time.Duration) *TTL[T] {
	return &TTL[T]{
		items:      make(map[string]Entry[T]),
		defaultTTL: defaultTTL,
		now:        time.Now,
	}
}

// Get returns the value if present and not expired.
func (c *TTL[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	if !ok {
		c.mu.RUnlock()
		var zero T
		return zero, false
	}
	if e.ExpiresAt.After(c.now()) {
		val := e.Value
		c.mu.RUnlock()
		return val, true
	}
	c.mu.RUnlock()

	// Expired: re-check under exclusive lock to avoid deleting a value that a
	// concurrent Set may have just installed (TOCTOU race on lazy eviction).
	c.mu.Lock()
	if ee, ok := c.items[key]; ok && !ee.ExpiresAt.After(c.now()) {
		delete(c.items, key)
	}
	c.mu.Unlock()

	var zero T
	return zero, false
}

// Set stores value with the default TTL.
func (c *TTL[T]) Set(key string, value T) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores value with an explicit TTL.
func (c *TTL[T]) SetWithTTL(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = Entry[T]{
		Value:     value,
		ExpiresAt: c.now().Add(ttl),
	}
}

// Delete removes a key.
func (c *TTL[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Cleanup removes all expired entries. Safe to call from a background goroutine.
func (c *TTL[T]) Cleanup() int {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for k, e := range c.items {
		if !e.ExpiresAt.After(now) {
			delete(c.items, k)
			removed++
		}
	}
	return removed
}

// Len returns the number of entries (including not-yet-evicted expired ones until Cleanup/Get).
func (c *TTL[T]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
