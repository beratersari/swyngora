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
