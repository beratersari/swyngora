package cache

import (
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

func TestTTL_NeverExpires_WhenTTLZero(t *testing.T) {
	c := New[string](0)
	fixed := time.Now()
	c.now = func() time.Time { return fixed }
	c.Set("k", "v")
	fixed = fixed.Add(24 * time.Hour)
	if got, ok := c.Get("k"); !ok || got != "v" {
		t.Fatalf("never-expire miss: %q ok=%v", got, ok)
	}
	if n := c.Cleanup(); n != 0 {
		t.Fatalf("cleanup removed %d", n)
	}
}

func TestTTL_ReplaceAll_Atomic(t *testing.T) {
	c := New[int](time.Hour)
	c.Set("old", 1)
	c.ReplaceAll(map[string]int{"a": 2, "b": 3})
	if _, ok := c.Get("old"); ok {
		t.Fatal("old key should be gone after ReplaceAll")
	}
	if got, ok := c.Get("a"); !ok || got != 2 {
		t.Fatalf("a=%v ok=%v", got, ok)
	}
	if c.Len() != 2 {
		t.Fatalf("len=%d", c.Len())
	}
}

func TestTTL_MaxEntries(t *testing.T) {
	c := NewWithOptions[int](time.Hour, Options{MaxEntries: 2})
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	if c.Len() > 2 {
		t.Fatalf("len=%d want <=2", c.Len())
	}
}

func TestTTL_GetStale(t *testing.T) {
	c := New[string](time.Millisecond)
	fixed := time.Now()
	c.now = func() time.Time { return fixed }
	c.Set("k", "stale")
	fixed = fixed.Add(time.Second)
	if _, ok := c.Get("k"); ok {
		t.Fatal("fresh get should miss")
	}
	got, ok := c.GetStale("k")
	if !ok || got != "stale" {
		t.Fatalf("stale=%q ok=%v", got, ok)
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

func TestTTL_Get_AfterExpiryRemainsUntilCleanup(t *testing.T) {
	c := New[string](time.Hour)
	now := time.Now()
	c.now = func() time.Time { return now }

	c.SetWithTTL("k", "v", 1*time.Millisecond)
	now = now.Add(5 * time.Millisecond)

	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after expiry")
	}
	// Entry retained for GetStale until Cleanup
	if c.Len() != 1 {
		t.Fatalf("expected retained expired entry, len=%d", c.Len())
	}
	if n := c.Cleanup(); n != 1 {
		t.Fatalf("cleanup removed %d", n)
	}
	if c.Len() != 0 {
		t.Fatalf("want empty after cleanup, len=%d", c.Len())
	}
}
