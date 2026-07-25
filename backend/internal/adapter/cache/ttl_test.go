package cache

import (
	"sync"
	"testing"
	"time"
)

func TestTTL_GetSet_Expiry(t *testing.T) {
	c := New[string](50 * time.Millisecond)
	fixed := time.Now()
	c.now = func() time.Time { return fixed }

	c.Set("k", "v")
	if got, ok := c.Get("k"); !ok || got != "v" {
		t.Fatalf("expected hit, got %q ok=%v", got, ok)
	}

	fixed = fixed.Add(100 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestTTL_Cleanup(t *testing.T) {
	c := New[int](time.Millisecond)
	fixed := time.Now()
	c.now = func() time.Time { return fixed }
	c.Set("a", 1)
	c.SetWithTTL("b", 2, time.Hour)

	fixed = fixed.Add(10 * time.Millisecond)
	n := c.Cleanup()
	if n != 1 {
		t.Fatalf("cleanup removed %d, want 1", n)
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("live key should remain")
	}
}

// TestTTL_Get_ConcurrentSetAfterExpiryDoesNotGetEvicted exercises the TOCTOU
// protection added to Get. A Get that observes an expired entry must not delete
// a fresh value that a concurrent Set installed between the read and the delete.
func TestTTL_Get_ConcurrentSetAfterExpiryDoesNotGetEvicted(t *testing.T) {
	c := New[string](time.Hour) // long default so only explicit TTLs matter
	now := time.Now()
	c.now = func() time.Time { return now }

	// Seed with a short-lived value
	c.SetWithTTL("k", "old", 5*time.Millisecond)

	// Advance "time" so any Get will consider it expired
	now = now.Add(10 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine A: Get that will see expired and enter the delete path
	go func() {
		defer wg.Done()
		// This will RUnlock after seeing expired, then Lock for recheck/delete
		_, _ = c.Get("k")
	}()

	// Give A a chance to read the expired snapshot
	time.Sleep(2 * time.Millisecond)

	// Goroutine B: Set a fresh long-lived value in the window
	go func() {
		defer wg.Done()
		c.SetWithTTL("k", "fresh", time.Hour)
	}()

	wg.Wait()

	// The fresh value must survive; the lagging delete must not have removed it.
	got, ok := c.Get("k")
	if !ok || got != "fresh" {
		t.Fatalf("fresh value was evicted by racing Get; got %q (ok=%v)", got, ok)
	}
}

// TestTTL_Get_AfterExpiryDeletes exercises the expiry delete path in Get.
func TestTTL_Get_AfterExpiryDeletes(t *testing.T) {
	c := New[string](time.Hour)
	now := time.Now()
	c.now = func() time.Time { return now }

	c.SetWithTTL("k", "v", 1*time.Millisecond)
	now = now.Add(5 * time.Millisecond)

	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after expiry")
	}

	// Entry should be gone (lazy delete happened)
	if c.Len() != 0 {
		t.Fatalf("expected empty after lazy delete on Get, len=%d", c.Len())
	}
}
