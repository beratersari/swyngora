package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeIdempotencyKey(t *testing.T) {
	k, err := NormalizeIdempotencyKey("  abc-123_X  ")
	if err != nil || k != "abc-123_X" {
		t.Fatalf("%q %v", k, err)
	}
	if _, err := NormalizeIdempotencyKey("bad key"); err == nil || !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("space should fail: %v", err)
	}
	if _, err := NormalizeIdempotencyKey(strings.Repeat("a", MaxIdempotencyKeyLen+1)); err == nil {
		t.Fatal("expected too long")
	}
	empty, err := NormalizeIdempotencyKey("  ")
	if err != nil || empty != "" {
		t.Fatalf("empty %q %v", empty, err)
	}
}

func TestIdempotencyRequestHashStable(t *testing.T) {
	a := IdempotencyRequestHash("market", "binance", "BTCUSDT", "buy", "1")
	b := IdempotencyRequestHash("market", "binance", "BTCUSDT", "buy", "1")
	c := IdempotencyRequestHash("market", "binance", "BTCUSDT", "buy", "2")
	if a != b || a == c || len(a) != 64 {
		t.Fatalf("a=%s b=%s c=%s", a, b, c)
	}
}
